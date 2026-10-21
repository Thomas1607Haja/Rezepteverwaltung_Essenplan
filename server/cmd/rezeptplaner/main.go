// Rezeptplaner-Server: stellt die REST-API für Rezepte, Essensplan und
// Einkaufsliste sowie die eingebettete Weboberfläche bereit.
package main

import (
	"context"
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

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", handleHealth)
	imgs, err := images.NewStore(filepath.Join(cfg.DataDir, "images"))
	if err != nil {
		return err
	}
	// Lädt fremde Seiten und Bilder für den URL-Import; Heimnetz gesperrt.
	fetcher := fetch.New(20*time.Second, false)
	recipeStore := recipes.NewStore(db)
	recipes.NewHandler(recipeStore, imgs, fetcher).Register(mux)
	pantryStore := pantry.NewStore(db)
	pantry.NewHandler(pantryStore).Register(mux)
	importer.NewHandler(recipeStore, pantryStore, fetcher).Register(mux)
	settingsStore := settings.NewStore(db)
	settings.NewHandler(settingsStore).Register(mux)
	planStore := mealplan.NewStore(db, settingsStore)
	mealplan.NewHandler(planStore).Register(mux)
	shopping.NewHandler(shopping.NewStore(db, planStore)).Register(mux)
	system.NewHandler(buildVersion, cfg.DataDir).Register(mux)
	// SUPERVISOR_TOKEN setzt Home Assistant in der App. Lokal lässt sich
	// mit REZEPTPLANER_HA_URL und REZEPTPLANER_HA_TOKEN (langlebiger Token
	// aus dem Benutzerprofil) eine echte Instanz nutzen, sonst gilt der
	// Testmodus.
	ha := homeassistant.New(os.Getenv("SUPERVISOR_TOKEN"), os.Getenv("REZEPTPLANER_HA_URL"), os.Getenv("REZEPTPLANER_HA_TOKEN"))
	slog.Info("Home Assistant", "modus", ha.Mode())
	homeassistant.NewHandler(ha).Register(mux)
	// "GET /" passt auf alle Pfade, die kein genaueres Muster haben: Das ist
	// die Weboberfläche. Die API-Muster oben sind genauer und gewinnen.
	mux.Handle("GET /", web.Handler())

	// handler ist vom Typ http.Handler, einem Interface: Jeder Typ mit der
	// Methode ServeHTTP passt hinein – der Router genauso wie eine
	// Middleware, die ihn umhüllt.
	var handler http.Handler = mux
	if cfg.TrustedProxy != "" {
		proxy, err := netip.ParseAddr(cfg.TrustedProxy)
		if err != nil {
			return fmt.Errorf("REZEPTPLANER_TRUSTED_PROXY: %w", err)
		}
		handler = httpx.AllowOnlyFrom(proxy, handler)
		slog.Info("Nur Verbindungen über Ingress erlaubt", "proxy", proxy)
	}

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: httpx.LogRequests(handler),
		// Schutz gegen Verbindungen, die ihre Anfrage nie zu Ende senden.
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Port zuerst belegen: Ist er schon vergeben, bricht der Start hier mit
	// einer klaren Fehlermeldung ab.
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("server starten: %w", err)
	}
	slog.Info("Server gestartet", "version", buildVersion, "url", "http://"+cfg.ListenAddr, "data_dir", cfg.DataDir)

	// Der Server läuft in einer eigenen Goroutine, damit run() parallel auf
	// das Stoppsignal warten kann. Sein Ergebnis kommt über den Kanal zurück.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Serve(listener)
	}()

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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server beenden: %w", err)
	}
	// Nach Shutdown liefert Serve immer ErrServerClosed – das ist
	// der normale Weg und kein Fehler.
	if err := <-serverErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server beenden: %w", err)
	}
	slog.Info("Server sauber beendet")
	return nil
}

// handleHealth ist das Lebenszeichen: antwortet immer mit "ok", solange der
// Server läuft.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
