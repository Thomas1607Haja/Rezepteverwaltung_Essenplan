package fileimport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/importer"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// Foto von der Quellseite (Schritt 8.B, siehe docs/importformat.md):
// Hat ein Rezept eine source.url, aber kein Bild oder nur eine
// Illustration, lädt die Vorschau das Rezeptbild dieser Seite. Von der
// Seite wird nur das Bild übernommen, nie andere Daten. Ein echtes Foto
// aus der Datei wird nie ersetzt.

// webImageTimeout: So lange darf das Laden aller Seiten einer Vorschau
// zusammen dauern. Die App bricht eine Anfrage nach 30 Sekunden ab; was bis
// dahin nicht geklappt hat, bleibt beim bisherigen Bild.
const webImageTimeout = 20 * time.Second

// webImageParallel: so viele Seiten gleichzeitig. Wenig, weil jedes Bild
// zum Prüfen dekodiert wird und der Green wenig Arbeitsspeicher hat.
const webImageParallel = 2

// webImagePath ist das geladene Bild eines Rezepts im Ordner des Uploads.
// Liegt es dort, verwendet das Bestätigen dieses Bild als Foto.
func webImagePath(dir string, index int) string {
	return filepath.Join(dir, "web-"+strconv.Itoa(index)+".img")
}

// wantsWebImage: Soll für das Rezept ein Foto von der Quellseite geladen
// werden? hasPhoto ist true, wenn das Bild aus der Datei ein brauchbares
// Foto ist.
func wantsWebImage(c *candidate, hasPhoto bool) bool {
	return len(c.Errors) == 0 && c.Recipe.SourceURL != "" && !hasPhoto
}

// loadWebImages lädt für die Rezepte cands die Bilder ihrer Quellseiten und
// legt Bild und Vorschaubild im Ordner dir ab. Zurück kommen die Rezepte,
// bei denen es geklappt hat (Index → Name der Seite). Fehler sind nur
// Hinweise im Log: Das Rezept behält dann sein bisheriges Bild.
func (im *Importer) loadWebImages(ctx context.Context, dir string, cands []*candidate) map[int]string {
	found := make(map[int]string)
	if im.fetcher == nil || len(cands) == 0 {
		return found
	}
	ctx, cancel := context.WithTimeout(ctx, webImageTimeout)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	// Ein Kanal mit Puffer als Zähler: Wer etwas hineinlegt, darf laden;
	// ist er voll, wartet der nächste, bis einer fertig ist.
	slots := make(chan struct{}, webImageParallel)
	for _, c := range cands {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-slots }()
			if err := im.loadWebImage(ctx, dir, c); err != nil {
				slog.Info("Datei-Import: kein Foto von der Quellseite", "url", c.Recipe.SourceURL, "grund", err)
				return
			}
			mu.Lock()
			found[c.Index] = siteName(c.Recipe.SourceURL)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return found
}

// loadWebImage lädt Seite und Bild eines Rezepts und speichert beides.
func (im *Importer) loadWebImage(ctx context.Context, dir string, c *candidate) error {
	pageURL, err := url.Parse(c.Recipe.SourceURL)
	if err != nil {
		return err
	}
	page, _, err := im.fetcher.Get(ctx, pageURL.String(), importer.MaxPageBytes)
	if err != nil {
		return err
	}
	imageURL := importer.PageImage(page, pageURL)
	if imageURL == "" {
		return errors.New("die seite nennt kein bild")
	}
	data, _, err := im.fetcher.Get(ctx, imageURL, images.MaxUploadBytes)
	if err != nil {
		return err
	}
	// Preview prüft den echten Bildtyp am Inhalt und dekodiert das Bild.
	thumb, err := im.images.Preview(data)
	if err != nil {
		return fmt.Errorf("kein brauchbares bild: %w", err)
	}
	if err := os.WriteFile(webImagePath(dir, c.Index), data, 0o600); err != nil {
		return err
	}
	return os.WriteFile(thumbPath(dir, c.Index), thumb, 0o600)
}

// siteName ist der Name der Seite für die Anzeige, z. B. "chefkoch.de".
func siteName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

// saveWebImage speichert beim Bestätigen das in der Vorschau geladene Foto.
// ok ist false, wenn es keins gibt.
func (im *Importer) saveWebImage(ctx context.Context, dir string, c *candidate, recipeID int64) (ok bool, warning string) {
	data, err := os.ReadFile(webImagePath(dir, c.Index))
	if err != nil {
		return false, ""
	}
	large, thumb, err := im.images.Save(bytes.NewReader(data))
	if err != nil {
		return true, fmt.Sprintf("Foto von der Quellseite nicht gespeichert: %v", err)
	}
	if _, err := im.recipes.SetImage(ctx, recipeID, large, thumb, recipes.ImagePhoto); err != nil {
		im.images.Delete(large, thumb)
		return true, fmt.Sprintf("Foto von der Quellseite nicht gespeichert: %v", err)
	}
	return true, ""
}
