// Package images verkleinert hochgeladene Rezeptbilder einmalig und legt
// sie als JPEG im Datenordner ab: ein großes Bild und ein Vorschaubild.
package images

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sync"

	// Wie beim SQLite-Treiber: Die Importe melden nur die Formate PNG,
	// GIF und WebP bei image.Decode an. JPEG ist oben schon importiert.
	_ "image/gif"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	// MaxUploadBytes begrenzt die Dateigröße eines Uploads (20 MB).
	MaxUploadBytes = 20 << 20
	// maxPixels schützt vor Bildern, deren entpackte Fassung zu viel
	// Arbeitsspeicher bräuchte (40 Megapixel ≈ 160 MB).
	maxPixels = 40_000_000

	largeSize = 1600 // längste Seite des großen Bildes
	thumbSize = 480  // längste Seite des Vorschaubildes

	largeQuality = 82
	thumbQuality = 75
)

// ErrUnsupported: Die Datei ist kein lesbares Bild.
var ErrUnsupported = errors.New("kein unterstütztes bildformat (jpeg, png, webp, gif)")

// ErrTooLarge: Das Bild hat zu viele Bildpunkte.
var ErrTooLarge = errors.New("bild ist zu groß (höchstens 40 megapixel)")

// validName prüft Dateinamen beim Ausliefern, damit niemand mit Namen wie
// "../rezeptplaner.db" an andere Dateien kommt.
var validName = regexp.MustCompile(`^[0-9a-f]{32}(_thumb)?\.jpg$`)

// Store verwaltet die Bilddateien in einem Ordner.
type Store struct {
	dir string
	// mu sorgt dafür, dass immer nur ein Bild gleichzeitig verarbeitet
	// wird: Jedes entpackte Foto belegt kurzzeitig viel Speicher.
	mu sync.Mutex
}

// NewStore legt den Ordner dir bei Bedarf an.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("bildordner anlegen: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Save liest ein Bild aus r, verkleinert es und speichert großes Bild und
// Vorschaubild. Zurück kommen die beiden Dateinamen.
func (s *Store) Save(r io.Reader) (large, thumb string, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", "", fmt.Errorf("bild lesen: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Nach der Verarbeitung den Speicher gleich wieder an das System
	// zurückgeben, statt auf die Go-Laufzeit zu warten. defer-Aufrufe
	// laufen in umgekehrter Reihenfolge: erst das, dann Unlock.
	defer debug.FreeOSMemory()

	largeData, thumbData, err := process(data)
	if err != nil {
		return "", "", err
	}

	name, err := randomName()
	if err != nil {
		return "", "", err
	}
	large, thumb = name+".jpg", name+"_thumb.jpg"
	if err := s.write(large, largeData); err != nil {
		return "", "", err
	}
	if err := s.write(thumb, thumbData); err != nil {
		s.Delete(large)
		return "", "", err
	}
	return large, thumb, nil
}

// Preview prüft ein Bild wie Save und liefert nur das Vorschaubild, ohne
// etwas zu speichern – für die Vorschau beim Datei-Import.
func (s *Store) Preview(data []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer debug.FreeOSMemory()
	_, thumb, err := process(data)
	return thumb, err
}

// process macht aus den Bytes eines Bildes die JPEG-Daten für großes Bild
// und Vorschaubild.
func process(data []byte) (largeJPEG, thumbJPEG []byte, err error) {
	// Erst nur die Abmessungen lesen – das braucht kaum Speicher.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, nil, ErrUnsupported
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, nil, ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, ErrUnsupported
	}
	orientation := jpegOrientation(data)

	// Die längste Seite ändert sich beim Drehen nicht. Daher erst
	// verkleinern, dann das viel kleinere Ergebnis drehen.
	largeImg := applyOrientation(resize(src, largeSize), orientation)
	src = nil // das große Original wird nicht mehr gebraucht
	thumbImg := resize(largeImg, thumbSize)

	if largeJPEG, err = encode(largeImg, largeQuality); err != nil {
		return nil, nil, err
	}
	if thumbJPEG, err = encode(thumbImg, thumbQuality); err != nil {
		return nil, nil, err
	}
	return largeJPEG, thumbJPEG, nil
}

// resize verkleinert img so, dass die längste Seite höchstens maxSide
// Bildpunkte hat. Kleinere Bilder werden nicht vergrößert.
//
// Verfahren: so oft halbieren, bis das Bild weniger als doppelt so groß
// ist wie das Ziel, dann den Rest bilinear skalieren. Beim exakten
// Halbieren mittelt ApproxBiLinear genau 2×2 Bildpunkte – das ergibt
// saubere Kanten ohne Treppen. Hochwertigere Verfahren wie CatmullRom
// legen einen Zwischenspeicher von 32 Byte je Bildpunkt an (bei einem
// Handyfoto über 150 MB); das wäre für den Green zu viel.
func resize(img image.Image, maxSide int) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	cur := img
	for max(w, h)/2 >= maxSide {
		w, h = max(w/2, 1), max(h/2, 1)
		cur = scale(cur, w, h)
	}
	if longest := max(w, h); longest > maxSide {
		w, h = max(w*maxSide/longest, 1), max(h*maxSide/longest, 1)
		cur = scale(cur, w, h)
	}
	if rgba, ok := cur.(*image.RGBA); ok && cur != img {
		return rgba
	}
	// Nichts zu verkleinern: trotzdem einmal umkopieren, damit
	// durchsichtige Stellen weiß werden.
	return scale(cur, w, h)
}

// scale zeichnet src auf eine neue w×h-Fläche mit weißem Hintergrund.
// Durchsichtige Stellen (PNG) werden so weiß, denn JPEG kennt keine
// Transparenz.
func scale(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func encode(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("jpeg schreiben: %w", err)
	}
	return buf.Bytes(), nil
}

// randomName liefert 32 zufällige Hex-Zeichen. Zufällige Namen ändern sich
// mit jedem neuen Bild, daher darf der Browser Bilder dauerhaft
// zwischenspeichern.
func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("dateiname erzeugen: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// write schreibt erst in eine temporäre Datei und benennt sie dann um.
// So gibt es nie eine halb geschriebene Bilddatei unter dem echten Namen.
func (s *Store) write(name string, data []byte) error {
	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("bild speichern: %w", err)
	}
	defer os.Remove(tmp.Name()) // nach erfolgreichem Umbenennen ohne Wirkung
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("bild speichern: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("bild speichern: %w", err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, name)); err != nil {
		return fmt.Errorf("bild speichern: %w", err)
	}
	return nil
}

// Copy legt von einem großen Bild und seinem Vorschaubild Kopien unter
// neuen Namen an (für Rezeptvarianten). So lässt sich das Bild der Kopie
// später ersetzen oder löschen, ohne das Original zu berühren. Die Dateien
// sind schon verkleinert und werden unverändert kopiert.
func (s *Store) Copy(large, thumb string) (newLarge, newThumb string, err error) {
	if !validName.MatchString(large) || !validName.MatchString(thumb) {
		return "", "", fmt.Errorf("bild kopieren: ungültiger dateiname %q / %q", large, thumb)
	}
	largeData, err := os.ReadFile(filepath.Join(s.dir, large))
	if err != nil {
		return "", "", fmt.Errorf("bild kopieren: %w", err)
	}
	thumbData, err := os.ReadFile(filepath.Join(s.dir, thumb))
	if err != nil {
		return "", "", fmt.Errorf("bild kopieren: %w", err)
	}
	name, err := randomName()
	if err != nil {
		return "", "", err
	}
	newLarge, newThumb = name+".jpg", name+"_thumb.jpg"
	if err := s.write(newLarge, largeData); err != nil {
		return "", "", err
	}
	if err := s.write(newThumb, thumbData); err != nil {
		s.Delete(newLarge)
		return "", "", err
	}
	return newLarge, newThumb, nil
}

// Delete entfernt die genannten Dateien. Leere Namen und fehlende Dateien
// werden übergangen; Fehler werden nur zurückgegeben, nicht abgebrochen.
func (s *Store) Delete(names ...string) error {
	var errs []error
	for _, name := range names {
		if name == "" || !validName.MatchString(name) {
			continue
		}
		err := os.Remove(filepath.Join(s.dir, name))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	// errors.Join fasst mehrere Fehler zu einem zusammen (nil, wenn keiner).
	return errors.Join(errs...)
}

// ServeHTTP liefert ein Bild unter GET /api/v1/images/{name} aus.
func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !validName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.dir, name)
	if _, err := os.Stat(path); err != nil {
		// Ohne Cache-Kopf, sonst merkte sich der Browser das 404 ein Jahr.
		http.NotFound(w, r)
		return
	}
	// Der Inhalt zu einem Namen ändert sich nie (siehe randomName).
	// "private": Nur der Browser darf speichern, kein Zwischenspeicher
	// unterwegs (z. B. Cloudflare) – der würde das Bild sonst auch ohne
	// API-Schlüssel ausliefern.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeFile(w, r, path)
}
