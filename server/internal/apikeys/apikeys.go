// Package apikeys verwaltet die API-Schlüssel des App-Zugangs: einer pro
// Gerät, nur als Hash gespeichert, einzeln sperrbar (löschen).
package apikeys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// prefix kennzeichnet Schlüssel des Rezeptplaners. Das hilft beim
// Erkennen, falls einer versehentlich irgendwo auftaucht (z. B. in einem
// Protokoll oder Repository).
const prefix = "rp_"

// maxName begrenzt die Länge des Gerätenamens.
const maxName = 60

// touchInterval: last_used_at wird höchstens so oft geschrieben, damit
// nicht jede Anfrage der App einen Schreibzugriff auslöst.
const touchInterval = time.Hour

// Key ist ein gespeicherter Schlüssel – ohne den Schlüssel selbst.
type Key struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"` // nil = noch nie benutzt
}

// ErrNotFound wird zurückgegeben, wenn es einen Schlüssel nicht gibt.
var ErrNotFound = errors.New("api-schlüssel nicht gefunden")

// ErrInvalidName: Der Gerätename ist leer oder zu lang.
var ErrInvalidName = fmt.Errorf("name darf nicht leer und höchstens %d zeichen lang sein", maxName)

// Store liest und schreibt die Tabelle api_keys.
type Store struct {
	db *sql.DB

	mu      sync.Mutex
	touched map[int64]time.Time // wann last_used_at zuletzt geschrieben wurde
}

// NewStore erzeugt den Store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db, touched: map[int64]time.Time{}}
}

// hash liefert den SHA-256-Hash eines Schlüssels als Hex-Text. Ein
// schneller Hash genügt: Die Schlüssel sind 32 Byte Zufall, da hilft
// Durchprobieren nicht (anders als bei Passwörtern).
func hash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Create legt einen Schlüssel für das Gerät name an. Der Schlüssel selbst
// kommt nur hier einmal zurück und wird nirgends gespeichert.
func (s *Store) Create(ctx context.Context, name string) (Key, string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" || len([]rune(name)) > maxName {
		return Key{}, "", ErrInvalidName
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Key{}, "", fmt.Errorf("schlüssel erzeugen: %w", err)
	}
	// RawURLEncoding: ohne "=", "+" und "/", lässt sich gefahrlos kopieren
	// und in einen QR-Code packen.
	secret := prefix + base64.RawURLEncoding.EncodeToString(raw)

	now := time.Now().UTC().Truncate(time.Millisecond)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO api_keys (name, key_hash, created_at) VALUES (?, ?, ?)`,
		name, hash(secret), database.FormatTime(now))
	if err != nil {
		return Key{}, "", fmt.Errorf("schlüssel speichern: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Key{}, "", err
	}
	return Key{ID: id, Name: name, CreatedAt: now}, secret, nil
}

// List liefert alle Schlüssel, die neuesten zuerst.
func (s *Store) List(ctx context.Context) ([]Key, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, created_at, last_used_at FROM api_keys ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("schlüssel lesen: %w", err)
	}
	defer rows.Close()

	keys := []Key{}
	for rows.Next() {
		var k Key
		var created string
		var lastUsed sql.NullString
		if err := rows.Scan(&k.ID, &k.Name, &created, &lastUsed); err != nil {
			return nil, fmt.Errorf("schlüssel lesen: %w", err)
		}
		if k.CreatedAt, err = database.ParseTime(created); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			t, err := database.ParseTime(lastUsed.String)
			if err != nil {
				return nil, err
			}
			k.LastUsedAt = &t
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// Delete sperrt einen Schlüssel endgültig.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("schlüssel löschen: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Check prüft einen Schlüssel aus einer Anfrage. Gesucht wird über den
// Hash: Die Antwortzeit verrät so nichts über den Schlüssel selbst, denn
// schon ein anderes Zeichen ergibt einen völlig anderen Hash.
func (s *Store) Check(ctx context.Context, key string) (bool, error) {
	if !strings.HasPrefix(key, prefix) {
		return false, nil
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM api_keys WHERE key_hash = ?`, hash(key)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("schlüssel prüfen: %w", err)
	}
	s.touch(ctx, id)
	return true, nil
}

// touch vermerkt die Nutzung, aber höchstens einmal je touchInterval.
// Scheitert das Schreiben, bleibt die Anfrage trotzdem gültig.
func (s *Store) touch(ctx context.Context, id int64) {
	now := time.Now()
	s.mu.Lock()
	if now.Sub(s.touched[id]) < touchInterval {
		s.mu.Unlock()
		return
	}
	s.touched[id] = now
	s.mu.Unlock()
	_, _ = s.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ? WHERE id = ?`, database.FormatTime(now), id)
}
