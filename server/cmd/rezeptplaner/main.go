// Rezeptplaner-Server: stellt die REST-API für Rezepte, Essensplan und
// Einkaufsliste bereit.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// config enthält alle Einstellungen, die beim Start gelesen werden.
type config struct {
	ListenAddr string // Adresse, auf der der Server lauscht
	DataDir    string // Ordner für Datenbank und Bilder
}

// loadConfig liest die Einstellungen aus Umgebungsvariablen. Fehlt eine
// Variable, gilt der Standardwert. Vorgabe ist localhost, damit der Server
// lokal nur vom eigenen Rechner aus erreichbar ist.
func loadConfig() config {
	return config{
		ListenAddr: getEnv("REZEPTPLANER_LISTEN_ADDR", "127.0.0.1:8080"),
		DataDir:    getEnv("REZEPTPLANER_DATA_DIR", "data"),
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", handleHealth)

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
		// Schutz gegen Verbindungen, die ihre Anfrage nie zu Ende senden.
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Port zuerst belegen: Ist er schon vergeben, bricht der Start hier mit
	// einer klaren Fehlermeldung ab.
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("server starten: %w", err)
	}
	slog.Info("Server gestartet", "url", "http://"+cfg.ListenAddr, "data_dir", cfg.DataDir)

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
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		slog.Warn("Antwort konnte nicht gesendet werden", "error", err)
	}
}
