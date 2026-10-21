package homeassistant

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiscoveryService ist der Dienstname der Discovery. Home Assistant gibt die
// Meldung an die Integration mit genau diesem Namen (Domain) weiter; er muss
// auch unter "discovery" in config.yaml stehen.
const DiscoveryService = "rezeptplaner"

// tokenFile liegt im Datenordner und enthält den Schlüssel der Integration.
const tokenFile = "integration_token"

// LoadOrCreateToken liefert den Schlüssel, mit dem die Integration lesend
// auf die API zugreift. Beim ersten Start wird er zufällig erzeugt und im
// Datenordner gespeichert, damit er Neustarts übersteht.
func LoadOrCreateToken(dataDir string) (string, error) {
	path := filepath.Join(dataDir, tokenFile)
	data, err := os.ReadFile(path)
	if err == nil && len(strings.TrimSpace(string(data))) >= 32 {
		return strings.TrimSpace(string(data)), nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("schlüssel der integration lesen: %w", err)
	}
	// 32 zufällige Bytes aus dem Zufallsgenerator des Betriebssystems,
	// als 64 Hex-Zeichen.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("schlüssel der integration erzeugen: %w", err)
	}
	token := hex.EncodeToString(raw)
	// 0o600: nur der Besitzer darf die Datei lesen und schreiben.
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("schlüssel der integration speichern: %w", err)
	}
	return token, nil
}

// DiscoveryConfig ist das, was die Integration zum Verbinden braucht.
type DiscoveryConfig struct {
	Host  string `json:"host"`
	Port  int    `json:"port"`
	Token string `json:"token"`
}

// Announce meldet die App beim Supervisor an. Home Assistant bietet die
// Integration daraufhin unter "Entdeckt" an bzw. aktualisiert eine schon
// eingerichtete (z. B. wenn sich die Adresse geändert hat).
func Announce(ctx context.Context, supervisorToken string, cfg DiscoveryConfig) error {
	body, err := json.Marshal(map[string]any{"service": DiscoveryService, "config": cfg})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://supervisor/discovery", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+supervisorToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("discovery: status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}
