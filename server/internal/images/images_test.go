package images

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// testImage erzeugt ein w×h-Bild: links rot, rechts blau. So lässt sich
// nach dem Drehen prüfen, wo die Seiten gelandet sind.
func testImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if x >= w/2 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation fügt einem JPEG ein EXIF-Segment mit der Ausrichtung o
// direkt hinter dem Dateianfang ein (Big-Endian-TIFF, ein Eintrag).
func withOrientation(data []byte, o uint16) []byte {
	var tiff bytes.Buffer
	tiff.WriteString("MM")
	binary.Write(&tiff, binary.BigEndian, uint16(42))     // TIFF-Kennung
	binary.Write(&tiff, binary.BigEndian, uint32(8))      // IFD0 beginnt bei 8
	binary.Write(&tiff, binary.BigEndian, uint16(1))      // ein Eintrag
	binary.Write(&tiff, binary.BigEndian, uint16(0x0112)) // Ausrichtung
	binary.Write(&tiff, binary.BigEndian, uint16(3))      // Typ SHORT
	binary.Write(&tiff, binary.BigEndian, uint32(1))      // Anzahl
	binary.Write(&tiff, binary.BigEndian, o)
	binary.Write(&tiff, binary.BigEndian, uint16(0)) // Auffüllen auf 4 Bytes
	binary.Write(&tiff, binary.BigEndian, uint32(0)) // kein weiteres IFD

	seg := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	var out bytes.Buffer
	out.Write(data[:2]) // SOI
	out.Write([]byte{0xFF, 0xE1})
	binary.Write(&out, binary.BigEndian, uint16(len(seg)+2))
	out.Write(seg)
	out.Write(data[2:])
	return out.Bytes()
}

func TestJPEGOrientation(t *testing.T) {
	plain := encodeJPEG(t, testImage(8, 4))
	if got := jpegOrientation(plain); got != 1 {
		t.Errorf("ohne EXIF = %d, erwartet 1", got)
	}
	for o := uint16(1); o <= 8; o++ {
		if got := jpegOrientation(withOrientation(plain, o)); got != int(o) {
			t.Errorf("Ausrichtung %d gelesen als %d", o, got)
		}
	}
	if got := jpegOrientation([]byte("kein jpeg")); got != 1 {
		t.Errorf("Unsinn = %d, erwartet 1", got)
	}
}

func TestApplyOrientation(t *testing.T) {
	src := testImage(4, 2) // links rot, rechts blau
	red := color.RGBA{255, 0, 0, 255}

	tests := []struct {
		orientation  int
		wantW, wantH int
		redAt        image.Point // ein Punkt, der danach rot sein muss
	}{
		{1, 4, 2, image.Pt(0, 0)},
		{3, 4, 2, image.Pt(3, 1)}, // 180°: rot nach rechts
		{6, 2, 4, image.Pt(0, 0)}, // 90° im Uhrzeigersinn: rot nach oben
		{8, 2, 4, image.Pt(0, 3)}, // 90° gegen den Uhrzeigersinn: rot nach unten
	}
	for _, tt := range tests {
		got := applyOrientation(src, tt.orientation)
		if b := got.Bounds(); b.Dx() != tt.wantW || b.Dy() != tt.wantH {
			t.Errorf("Ausrichtung %d: Größe %dx%d, erwartet %dx%d", tt.orientation, b.Dx(), b.Dy(), tt.wantW, tt.wantH)
			continue
		}
		if c := color.RGBAModel.Convert(got.At(tt.redAt.X, tt.redAt.Y)); c != red {
			t.Errorf("Ausrichtung %d: Punkt %v = %v, erwartet rot", tt.orientation, tt.redAt, c)
		}
	}
}

func TestSaveResizesAndRotates(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// 3200×1600 quer gespeichert, laut EXIF um 90° zu drehen (Handy hochkant)
	data := withOrientation(encodeJPEG(t, testImage(3200, 1600)), 6)
	large, thumb, err := s.Save(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	for _, tt := range []struct {
		name         string
		wantW, wantH int
	}{
		{large, 800, 1600}, // gedreht und auf 1600 px längste Seite
		{thumb, 240, 480},
	} {
		f, err := os.Open(filepath.Join(s.dir, tt.name))
		if err != nil {
			t.Fatalf("%s fehlt: %v", tt.name, err)
		}
		cfg, err := jpeg.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Width != tt.wantW || cfg.Height != tt.wantH {
			t.Errorf("%s: %dx%d, erwartet %dx%d", tt.name, cfg.Width, cfg.Height, tt.wantW, tt.wantH)
		}
	}

	if err := s.Delete(large, thumb, ""); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	entries, _ := os.ReadDir(s.dir)
	if len(entries) != 0 {
		t.Errorf("%d Dateien übrig, erwartet 0", len(entries))
	}
}

func TestSaveSmallPNGIsNotEnlarged(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	var buf bytes.Buffer
	png.Encode(&buf, testImage(300, 200))

	large, _, err := s.Save(&buf)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	f, _ := os.Open(filepath.Join(s.dir, large))
	defer f.Close()
	cfg, _ := jpeg.DecodeConfig(f)
	if cfg.Width != 300 || cfg.Height != 200 {
		t.Errorf("%dx%d, erwartet unverändert 300x200", cfg.Width, cfg.Height)
	}
}

func TestSaveRejectsNonImages(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	if _, _, err := s.Save(bytes.NewReader([]byte("kein bild"))); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Fehler = %v, erwartet ErrUnsupported", err)
	}
}

func TestServeHTTP(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	large, _, err := s.Save(bytes.NewReader(encodeJPEG(t, testImage(10, 10))))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/images/{name}", s)

	tests := []struct {
		path       string
		wantStatus int
		wantCache  bool
	}{
		{"/api/v1/images/" + large, 200, true},
		{"/api/v1/images/00000000000000000000000000000000.jpg", 404, false},
		{"/api/v1/images/..%2Frezeptplaner.db", 404, false},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", tt.path, nil))
		if rec.Code != tt.wantStatus {
			t.Errorf("%s: Status %d, erwartet %d", tt.path, rec.Code, tt.wantStatus)
		}
		if hasCache := rec.Header().Get("Cache-Control") != ""; hasCache != tt.wantCache {
			t.Errorf("%s: Cache-Control %q", tt.path, rec.Header().Get("Cache-Control"))
		}
	}
}
