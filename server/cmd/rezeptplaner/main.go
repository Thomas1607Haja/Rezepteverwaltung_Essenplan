// Rezeptplaner-Server: stellt die REST-API für Rezepte, Essensplan und
// Einkaufsliste sowie die eingebettete Weboberfläche bereit.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apikeys"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/appinstall"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/appsync"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fileimport"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/importer"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/pantry"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/settings"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/shopping"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/system"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/web"
)

// buildVersion wird beim Bauen gesetzt: go build -ldflags="-X main.buildVersion=0.1.0".
// Ohne diese Angabe (z. B. bei go run) bleibt es bei "dev".
var buildVersion = "dev"

// config enthält alle Einstellungen, die beim Start gelesen werden.
type config struct {
	ListenAddr string // Adresse, auf der der Server lauscht
	DataDir    string // Ordner für Datenbank und Bilder
	// TrustedProxy: Ist sie gesetzt, nimmt der Server nur Verbindungen von
	// dieser Adresse an (in Home Assistant der Ingress-Vermittler
	// 172.30.32.2). Leer = keine Einschränkung, z. B. beim lokalen Testen.
	TrustedProxy string
	// AppAddr: Adresse des App-Zugangs (Android-App). Dort verlangt jede
	// Anfrage einen API-Schlüssel. Leer = kein App-Zugang.
	AppAddr string
}

// loadConfig liest die Einstellungen aus Umgebungsvariablen. Fehlt eine
// Variable, gilt der Standardwert. Vorgabe ist localhost, damit der Server
// lokal nur vom eigenen Rechner aus erreichbar ist.
func loadConfig() config {
	return config{
		ListenAddr: getEnv("REZEPTPLANER_LISTEN_ADDR", "127.0.0.1:8080"),
		DataDir:    getEnv("REZEPTPLANER_DATA_DIR", "data"),
		// Ohne Standardwert: getEnv liefert dann "", also keine Einschränkung.
		TrustedProxy: getEnv("REZEPTPLANER_TRUSTED_PROXY", ""),
		AppAddr:      getEnv("REZEPTPLANER_APP_ADDR", ""),
	}
}

// getEnv liefert den Wert der Umgebungsvariable key oder fallback, wenn sie
// nicht gesetzt oder leer ist.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	// Die eigentliche Arbeit steckt in run(), damit Fehler dort einfach per
	// return gemeldet werden können. Nur hier wird das Programm beendet.
	if err := run(); err != nil {
		slog.Error("Server mit Fehler beendet", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := loadConfig()

	// ctx wird "beendet", sobald Strg+C (Interrupt) oder SIGTERM eintrifft.
	// SIGTERM schickt Home Assistant beim Stoppen des Add-ons.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("datenordner anlegen: %w", err)
	}
	db, err := database.Open(ctx, filepath.Join(cfg.DataDir, "rezeptplaner.db"))
	if err != nil {
		return err
	}
	// Wird erst ausgeführt, wenn run() endet – also nach dem Herunterfahren
	// des Servers, wenn keine Anfrage mehr die Datenbank braucht.
	defer db.Close()

	version, err := database.Migrate(ctx, db)
	if err != nil {
		return err
	}
	slog.Info("Datenbank bereit", "schemaversion", version)

	imgs, err := images.NewStore(filepath.Join(cfg.DataDir, "images"))
	if err != nil {
		return err
	}
	// SUPERVISOR_TOKEN setzt Home Assistant in der App. Lokal lässt sich
	// mit REZEPTPLANER_HA_URL und REZEPTPLANER_HA_TOKEN (langlebiger Token
	// aus dem Benutzerprofil) eine echte Instanz nutzen, sonst gilt der
	// Testmodus.
	ha := homeassistant.New(os.Getenv("SUPERVISOR_TOKEN"), os.Getenv("REZEPTPLANER_HA_URL"), os.Getenv("REZEPTPLANER_HA_TOKEN"))
	slog.Info("Home Assistant", "modus", ha.Mode())
	keys := apikeys.NewStore(db)

	mux := http.NewServeMux()
	if err := registerRoutes(mux, services{
		db:      db,
		images:  imgs,
		fetcher: fetch.New(20*time.Second, false), // Heimnetz gesperrt
		ha:      ha,
		keys:    keys,
		version: buildVersion,
		dataDir: cfg.DataDir,
	}); err != nil {
		return err
	}

	// handler ist vom Typ http.Handler, einem Interface: Jeder Typ mit der
	// Methode ServeHTTP passt hinein – der Router genauso wie eine
	// Middleware, die ihn umhüllt.
	var handler http.Handler = mux
	// In der App bekommt die Home-Assistant-Integration einen eigenen
	// Schlüssel für lesenden Zugriff (siehe announce unten).
	supervisorToken := os.Getenv("SUPERVISOR_TOKEN")
	var integrationToken string
	if supervisorToken != "" {
		if integrationToken, err = homeassistant.LoadOrCreateToken(cfg.DataDir); err != nil {
			return err
		}
	}
	if cfg.TrustedProxy != "" {
		proxy, err := netip.ParseAddr(cfg.TrustedProxy)
		if err != nil {
			return fmt.Errorf("REZEPTPLANER_TRUSTED_PROXY: %w", err)
		}
		handler = httpx.AllowOnlyFrom(proxy, integrationToken, handler)
		slog.Info("Nur Verbindungen über Ingress erlaubt", "proxy", proxy, "integration", integrationToken != "")
	}

	// Zwei getrennte Zugänge auf eigenen Ports:
	// - Weboberfläche über Ingress (handler oben, Anmeldung durch Home
	//   Assistant) – Port 8099, nach außen nicht freigegeben.
	// - App-Zugang: jede Anfrage nur mit gültigem API-Schlüssel.
	servers := []*http.Server{newServer(cfg.ListenAddr, handler)}
	if cfg.AppAddr != "" {
		servers = append(servers, newServer(cfg.AppAddr, apikeys.Require(keys, mux)))
	}

	// Ports zuerst belegen: Ist einer schon vergeben, bricht der Start hier
	// mit einer klaren Fehlermeldung ab.
	listeners := make([]net.Listener, len(servers))
	for i, srv := range servers {
		if listeners[i], err = net.Listen("tcp", srv.Addr); err != nil {
			return fmt.Errorf("server starten: %w", err)
		}
	}
	slog.Info("Server gestartet", "version", buildVersion, "url", "http://"+cfg.ListenAddr, "data_dir", cfg.DataDir)
	if cfg.AppAddr != "" {
		slog.Info("App-Zugang aktiv (nur mit API-Schlüssel)", "adresse", cfg.AppAddr)
	}

	// Jeder Server läuft in einer eigenen Goroutine, damit run() parallel
	// auf das Stoppsignal warten kann. Die Ergebnisse kommen über den Kanal
	// zurück; er hat für jeden Server Platz, damit keine Goroutine hängt.
	serverErr := make(chan error, len(servers))
	for i, srv := range servers {
		go func() {
			serverErr <- srv.Serve(listeners[i])
		}()
	}
	if supervisorToken != "" {
		go announce(ctx, supervisorToken, listeners[0].Addr(), integrationToken)
	}

	// Warten, was zuerst passiert: unerwarteter Serverfehler oder Stoppsignal.
	select {
	case err := <-serverErr:
		return fmt.Errorf("server läuft nicht mehr: %w", err)
	case <-ctx.Done():
	}

	// Laufende Anfragen dürfen noch bis zu 5 Sekunden zu Ende laufen.
	slog.Info("Stoppsignal empfangen, beende Server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, srv := range servers {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server beenden: %w", err)
		}
	}
	// Nach Shutdown liefert Serve immer ErrServerClosed – das ist
	// der normale Weg und kein Fehler.
	for range servers {
		if err := <-serverErr; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server beenden: %w", err)
		}
	}
	slog.Info("Server sauber beendet")
	return nil
}

// newServer erzeugt einen HTTP-Server mit Protokoll jeder Anfrage.
func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: httpx.LogRequests(httpx.SecurityHeaders(handler)),
		// Schutz gegen Verbindungen, die ihre Anfrage nie zu Ende senden.
		ReadHeaderTimeout: 10 * time.Second,
	}
}

// services bündelt, was die Endpunkte brauchen.
type services struct {
	db      *sql.DB
	images  *images.Store
	fetcher *fetch.Client
	ha      homeassistant.Client
	keys    *apikeys.Store
	version string
	dataDir string
}

// registerRoutes meldet alle Endpunkte an. Beide Zugänge (Ingress und App)
// teilen sich diesen Router; was über den App-Zugang erreichbar ist,
// regelt apikeys.Require. Der Test in main_test.go durchläuft alle hier
// angemeldeten Muster.
func registerRoutes(mux httpx.Router, s services) error {
	mux.HandleFunc("GET /api/v1/health", handleHealth)
	recipeStore := recipes.NewStore(s.db)
	recipes.NewHandler(recipeStore, s.images, s.fetcher).Register(mux)
	pantryStore := pantry.NewStore(s.db)
	pantry.NewHandler(pantryStore).Register(mux)
	importer.NewHandler(recipeStore, pantryStore, s.fetcher).Register(mux)
	fileImporter, err := fileimport.NewImporter(filepath.Join(s.dataDir, "import"), recipeStore, pantryStore, s.images)
	if err != nil {
		return err
	}
	fileImporter.Register(mux)
	settingsStore := settings.NewStore(s.db)
	settings.NewHandler(settingsStore).Register(mux)
	planStore := mealplan.NewStore(s.db, settingsStore)
	mealplan.NewHandler(planStore).Register(mux)
	appsync.NewHandler(s.db, recipeStore, planStore, pantryStore, settingsStore).Register(mux)
	system.NewHandler(s.version, s.dataDir).Register(mux)
	homeassistant.NewHandler(s.ha).Register(mux)
	shopping.NewHandler(shopping.NewStore(s.db, planStore, settingsStore, s.ha)).Register(mux)
	apikeys.NewHandler(s.keys).Register(mux)
	appinstall.NewHandler(appinstall.LoadRepository(s.dataDir)).Register(mux)
	// "GET /" passt auf alle Pfade, die kein genaueres Muster haben: Das ist
	// die Weboberfläche. Die API-Muster oben sind genauer und gewinnen.
	mux.Handle("GET /", web.Handler())
	return nil
}

// handleHealth ist das Lebenszeichen: antwortet immer mit "ok", solange der
// Server läuft.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// announce meldet die App beim Supervisor, damit Home Assistant die
// Integration anbietet. Die Adresse im internen Netz ist der Hostname des
// Containers (die App-ID mit "-" statt "_"). Klappt es nicht sofort (z. B.
// weil der Supervisor gerade beschäftigt ist), wird es einige Male erneut
// versucht; ohne Erfolg fehlt nur die Integration, die App läuft trotzdem.
func announce(ctx context.Context, supervisorToken string, addr net.Addr, token string) {
	host, err := os.Hostname()
	if err != nil {
		slog.Warn("Discovery: Hostname unbekannt", "error", err)
		return
	}
	// addr.(*net.TCPAddr) ist eine Typzusicherung: Ein TCP-Listener liefert
	// immer eine *net.TCPAddr, aus der sich der Port lesen lässt.
	tcp, ok := addr.(*net.TCPAddr)
	if !ok {
		return
	}
	cfg := homeassistant.DiscoveryConfig{Host: host, Port: tcp.Port, Token: token}
	for attempt := 1; attempt <= 5; attempt++ {
		err := homeassistant.Announce(ctx, supervisorToken, cfg)
		if err == nil {
			slog.Info("Bei Home Assistant angemeldet (Discovery)", "host", host, "port", tcp.Port)
			return
		}
		slog.Warn("Discovery fehlgeschlagen", "versuch", attempt, "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(attempt) * 10 * time.Second):
		}
	}
}
