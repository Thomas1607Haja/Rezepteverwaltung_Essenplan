// Package database öffnet die SQLite-Datenbank und bringt ihr Schema per
// Migrationen auf den aktuellen Stand.
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"strconv"
	"strings"

	// Der Unterstrich-Import registriert nur den Treiber "sqlite" bei
	// database/sql; direkt aufgerufen wird aus dem Paket nichts.
	_ "modernc.org/sqlite"
)

// migrationFiles enthält alle SQL-Dateien aus migrations/. Die
// go:embed-Anweisung bettet sie beim Kompilieren in die Programmdatei ein.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// Einstellungen, die für jede Verbindung gelten:
//   - foreign_keys: Verweise prüfen, ON DELETE CASCADE ausführen
//   - journal_mode WAL: Lesen und Schreiben blockieren sich kaum
//   - synchronous NORMAL: mit WAL sicher und deutlich schneller als FULL
//   - busy_timeout: bei gesperrter Datenbank bis zu 5 s warten statt Fehler
//   - _txlock=immediate: Transaktionen sperren sofort zum Schreiben
const connectionOptions = "_pragma=foreign_keys(1)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=busy_timeout(5000)" +
	"&_txlock=immediate"

// Open öffnet die Datenbankdatei unter path (":memory:" für eine
// Datenbank nur im Arbeitsspeicher, z. B. in Tests).
func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?"+connectionOptions)
	if err != nil {
		return nil, fmt.Errorf("datenbank öffnen: %w", err)
	}

	// Genau eine Verbindung: Für einen Familienhaushalt reicht das
	// mühelos, und gleichzeitige Schreibzugriffe sind damit ausgeschlossen.
	// Wichtig: Innerhalb einer Transaktion nur über tx abfragen, nie über
	// db – sonst wartet die Abfrage ewig auf die belegte Verbindung.
	db.SetMaxOpenConns(1)

	// sql.Open verbindet noch nicht; erst Ping prüft, ob alles klappt.
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("datenbank öffnen: %w", err)
	}
	return db, nil
}

// migration ist eine SQL-Datei aus migrations/.
type migration struct {
	version int    // fortlaufende Nummer aus dem Dateinamen
	name    string // Dateiname
	sql     string // Inhalt
}

// Migrate spielt alle noch fehlenden Migrationen ein und gibt die danach
// erreichte Schemaversion zurück. Die aktuelle Version merkt sich SQLite
// selbst in PRAGMA user_version.
func Migrate(ctx context.Context, db *sql.DB) (int, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return 0, err
	}

	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return 0, fmt.Errorf("schemaversion lesen: %w", err)
	}
	if current > len(migrations) {
		return current, fmt.Errorf("datenbank hat schemaversion %d, dieses programm kennt nur bis %d", current, len(migrations))
	}

	for _, m := range migrations[current:] {
		if err := applyMigration(ctx, db, m); err != nil {
			return current, err
		}
		current = m.version
		slog.Info("Migration eingespielt", "datei", m.name)
	}
	return current, nil
}

// loadMigrations liest die eingebetteten SQL-Dateien. Die Dateinamen müssen
// mit einer lückenlos fortlaufenden Nummer beginnen: 001_..., 002_..., ...
func loadMigrations() ([]migration, error) {
	// ReadDir liefert die Einträge nach Namen sortiert.
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("migrationen lesen: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	for i, entry := range entries {
		prefix, _, _ := strings.Cut(entry.Name(), "_")
		version, err := strconv.Atoi(prefix)
		if err != nil || version != i+1 {
			return nil, fmt.Errorf("migration %q: name muss mit %03d_ beginnen", entry.Name(), i+1)
		}
		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("migration %q lesen: %w", entry.Name(), err)
		}
		migrations = append(migrations, migration{version: version, name: entry.Name(), sql: string(content)})
	}
	return migrations, nil
}

// applyMigration führt eine Migration in einer Transaktion aus: entweder
// wird sie vollständig eingespielt oder gar nicht.
func applyMigration(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migration %s: %w", m.name, err)
	}
	// Rollback nach erfolgreichem Commit ist wirkungslos; bei jedem
	// vorzeitigen return macht es dagegen alle Änderungen rückgängig.
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("migration %s: %w", m.name, err)
	}
	// PRAGMA erlaubt keine ?-Platzhalter. version ist eine Zahl aus
	// unserem eigenen Dateinamen, daher ist Sprintf hier unbedenklich.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
		return fmt.Errorf("migration %s: %w", m.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %s: %w", m.name, err)
	}
	return nil
}
