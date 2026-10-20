package images

import (
	"encoding/binary"
	"image"
	"image/draw"
)

// jpegOrientation liest die EXIF-Ausrichtung (1–8) aus den Bytes einer
// JPEG-Datei. Kameras speichern ein hochkant gehaltenes Foto oft quer und
// vermerken nur, wie es zu drehen ist. Fehlt der Vermerk oder ist die
// Datei kein JPEG, ist das Ergebnis 1 (unverändert).
//
// Aufbau: JPEG besteht aus Segmenten (Marker 0xFF 0xXX, Länge, Daten). Die
// EXIF-Daten liegen im Segment APP1 (0xFFE1) nach der Kennung "Exif\0\0"
// im TIFF-Format: Kopf mit Byte-Reihenfolge, dann eine Liste von Einträgen
// (IFD0). Eintrag 0x0112 ist die Ausrichtung.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return 1
		}
		marker := data[pos+1]
		// Ab "Start of Scan" folgen die Bilddaten; EXIF käme nur davor.
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		length := int(binary.BigEndian.Uint16(data[pos+2:]))
		end := pos + 2 + length
		if length < 2 || end > len(data) {
			return 1
		}
		if marker == 0xE1 {
			if o := exifOrientation(data[pos+4 : end]); o != 0 {
				return o
			}
		}
		pos = end
	}
	return 1
}

// exifOrientation liest die Ausrichtung aus einem APP1-Segment; 0, wenn
// sie dort nicht steht.
func exifOrientation(seg []byte) int {
	if len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := seg[6:]

	// "II" = Intel (Little Endian), "MM" = Motorola (Big Endian).
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}

	ifd := int(order.Uint32(tiff[4:]))
	if ifd+2 > len(tiff) {
		return 0
	}
	count := int(order.Uint16(tiff[ifd:]))
	for i := 0; i < count; i++ {
		entry := ifd + 2 + i*12 // jeder Eintrag ist 12 Bytes lang
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:]) == 0x0112 {
			// Typ SHORT: Der Wert steht in den ersten 2 der 4 Wertbytes.
			o := int(order.Uint16(tiff[entry+8:]))
			if o >= 1 && o <= 8 {
				return o
			}
			return 0
		}
	}
	return 0
}

// applyOrientation dreht bzw. spiegelt img so, dass es aufrecht steht.
// Die Werte 1–8 sind im EXIF-Standard festgelegt.
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Bei 5–8 tauschen Breite und Höhe.
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	// Für jeden Bildpunkt (x, y) der Quelle die Zielposition berechnen.
	// Langsam für Riesenbilder, aber hier läuft es erst nach dem
	// Verkleinern auf höchstens 1600 px.
	src := toRGBA(img)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var tx, ty int
			switch orientation {
			case 2: // gespiegelt
				tx, ty = w-1-x, y
			case 3: // 180°
				tx, ty = w-1-x, h-1-y
			case 4: // senkrecht gespiegelt
				tx, ty = x, h-1-y
			case 5: // gespiegelt + 90° gegen den Uhrzeigersinn
				tx, ty = y, x
			case 6: // 90° im Uhrzeigersinn
				tx, ty = h-1-y, x
			case 7: // gespiegelt + 90° im Uhrzeigersinn
				tx, ty = h-1-y, w-1-x
			case 8: // 90° gegen den Uhrzeigersinn
				tx, ty = y, w-1-x
			}
			i := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			j := dst.PixOffset(tx, ty)
			copy(dst.Pix[j:j+4], src.Pix[i:i+4])
		}
	}
	return dst
}

// toRGBA liefert img als *image.RGBA (ohne Kopie, wenn es das schon ist).
func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	rgba := image.NewRGBA(img.Bounds())
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return rgba
}
