package fileimport

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// Grenzen gegen zu große Dateien und ZIP-Bomben. Eine ZIP-Datei kann sehr
// klein sein und entpackt Gigabytes ergeben; deshalb zählt readLimited beim
// Entpacken mit, statt den Größenangaben in der ZIP-Datei zu glauben.
const (
	maxUploadBytes   = 30 << 20  // alle Dateien eines Uploads zusammen
	maxFiles         = 10        // Dateien je Upload
	maxEntries       = 200       // Einträge je ZIP-Datei
	maxJSONBytes     = 2 << 20   // recipes.json bzw. eine .json-Datei
	maxImageBytes    = 20 << 20  // ein Bild, entpackt
	maxUnpackedBytes = 150 << 20 // alles Entpackte eines Uploads zusammen
)

// budget zählt, wie viele Bytes beim Entpacken noch erlaubt sind.
type budget struct {
	left int64
}

func newBudget() *budget {
	return &budget{left: maxUnpackedBytes}
}

// readLimited liest r vollständig, aber höchstens limit Bytes und nicht
// mehr, als das Budget erlaubt. what benennt die Datei in Meldungen.
func readLimited(r io.Reader, limit int64, b *budget, what string) ([]byte, error) {
	allowed := min(limit, b.left)
	// Ein Byte mehr lesen als erlaubt: Kommt es an, ist die Datei zu groß.
	data, err := io.ReadAll(io.LimitReader(r, allowed+1))
	if err != nil {
		return nil, fmt.Errorf("%s nicht lesbar: %w", what, err)
	}
	if int64(len(data)) > allowed {
		if allowed < limit {
			return nil, fmt.Errorf("Entpackt zu groß (höchstens %d MB je Import)", maxUnpackedBytes>>20)
		}
		return nil, fmt.Errorf("%s ist zu groß (höchstens %d MB)", what, limit>>20)
	}
	b.left -= int64(len(data))
	return data, nil
}

// source ist eine hochgeladene Datei, aus der Rezepte kommen.
type source struct {
	doc document
	// zip ist nil bei einer einzelnen .json-Datei (ohne Bilder).
	zip *zip.ReadCloser
	// files ordnet den Pfaden relativ zu recipes.json die Einträge zu.
	files map[string]*zip.File
}

// Close gibt die ZIP-Datei frei; mehrfaches Aufrufen schadet nicht.
func (s *source) Close() {
	if s.zip != nil {
		s.zip.Close()
		s.zip = nil
	}
}

// image liest ein Bild aus der ZIP-Datei; rel ist der Pfad aus recipes.json.
func (s *source) image(rel string, b *budget) ([]byte, error) {
	if s.zip == nil {
		return nil, errors.New("Bilder gibt es nur in ZIP-Dateien, das Bild wird weggelassen")
	}
	f := s.files[rel]
	if f == nil {
		return nil, fmt.Errorf("Bild %s fehlt in der ZIP-Datei", rel)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("Bild %s nicht lesbar", rel)
	}
	defer rc.Close()
	return readLimited(rc, maxImageBytes, b, "Bild "+rel)
}

// openSource erkennt das Format am Inhalt (nicht an der Endung, die beim
// Teilen auf dem Handy oft fehlt oder falsch ist) und liest recipes.json.
func openSource(filePath string, b *budget) (*source, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	head := make([]byte, 4)
	n, _ := io.ReadFull(f, head)
	f.Close()
	head = head[:n]

	switch {
	case bytes.HasPrefix(head, []byte("PK\x03\x04")), bytes.HasPrefix(head, []byte("PK\x05\x06")):
		return openZip(filePath, b)
	case looksLikeJSON(head):
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		data, err := readLimited(f, maxJSONBytes, b, "Die JSON-Datei")
		if err != nil {
			return nil, err
		}
		doc, err := parseDocument(data)
		if err != nil {
			return nil, err
		}
		return &source{doc: doc}, nil
	}
	return nil, errors.New("Weder ZIP- noch JSON-Datei")
}

// looksLikeJSON: beginnt (nach BOM und Leerzeichen) mit "{".
func looksLikeJSON(head []byte) bool {
	head = bytes.TrimPrefix(head, []byte("\xef\xbb\xbf"))
	head = bytes.TrimLeft(head, " \t\r\n")
	return len(head) == 0 || head[0] == '{'
}

func openZip(filePath string, b *budget) (*source, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, errors.New("Die ZIP-Datei ist beschädigt")
	}
	src := &source{zip: zr, files: make(map[string]*zip.File)}
	ok := false
	defer func() {
		if !ok {
			zr.Close()
		}
	}()

	if len(zr.File) > maxEntries {
		return nil, fmt.Errorf("Zu viele Einträge in der ZIP-Datei (höchstens %d)", maxEntries)
	}
	var docEntries []*zip.File
	for _, f := range zr.File {
		if err := checkEntry(f); err != nil {
			return nil, err
		}
		if f.FileInfo().IsDir() || ignoredEntry(f.Name) {
			continue
		}
		if path.Base(f.Name) == "recipes.json" && strings.Count(f.Name, "/") <= 1 {
			docEntries = append(docEntries, f)
		}
	}
	switch len(docEntries) {
	case 0:
		return nil, errors.New("recipes.json fehlt in der ZIP-Datei")
	case 1:
	default:
		return nil, errors.New("Mehr als eine recipes.json in der ZIP-Datei")
	}

	// Liegt alles in einem Ordner (ZIP eines Ordners), gelten die Pfade
	// relativ zu diesem Ordner.
	root := path.Dir(docEntries[0].Name)
	for _, f := range zr.File {
		rel := f.Name
		if root != "." {
			var found bool
			if rel, found = strings.CutPrefix(f.Name, root+"/"); !found {
				continue
			}
		}
		src.files[rel] = f
	}

	rc, err := docEntries[0].Open()
	if err != nil {
		return nil, errors.New("recipes.json nicht lesbar")
	}
	data, err := readLimited(rc, maxJSONBytes, b, "recipes.json")
	rc.Close()
	if err != nil {
		return nil, err
	}
	if src.doc, err = parseDocument(data); err != nil {
		return nil, err
	}
	ok = true
	return src, nil
}

// checkEntry lehnt Einträge ab, die beim Entpacken außerhalb des Ziels
// landen könnten (Zip Slip) oder auf andere Dateien zeigen. Entpackt wird
// hier zwar nie auf die Festplatte, aber eine solche Datei ist sicher
// keine echte Importdatei.
func checkEntry(f *zip.File) error {
	name := f.Name
	unsafe := name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) ||
		strings.Contains(name, ":") || f.Mode()&fs.ModeSymlink != 0
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			unsafe = true
		}
	}
	if unsafe {
		return fmt.Errorf("Unzulässiger Pfad in der ZIP-Datei: %q", name)
	}
	return nil
}

// ignoredEntry: Beiwerk, das Packprogramme mancher Systeme hinzufügen.
func ignoredEntry(name string) bool {
	return strings.HasPrefix(name, "__MACOSX/") || strings.HasPrefix(path.Base(name), ".")
}
