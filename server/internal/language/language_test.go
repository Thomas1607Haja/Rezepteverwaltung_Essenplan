package language

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"de": "de", "de-DE": "de", "de_CH": "de", "FR": "fr", "en-GB": "en",
		"es-419": "es", "it": "it", "nl": "", "": "", "zh-Hans": "",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, erwartet %q", in, got, want)
		}
	}
}

type fakeSource struct {
	lang  string
	err   error
	calls int
}

func (f *fakeSource) Language(context.Context) (string, error) {
	f.calls++
	return f.lang, f.err
}

func TestResolver(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	src := &fakeSource{lang: "fr-CA"}
	r := NewResolver(src, "de")
	r.now = func() time.Time { return now }
	if got := r.Default(ctx); got != "fr" {
		t.Errorf("Home Assistant zuerst: %q", got)
	}
	// Zwischengespeichert: kein zweiter Abruf innerhalb von cacheFor.
	src.lang = "it"
	r.HomeAssistant(ctx)
	if src.calls != 1 {
		t.Errorf("Abrufe = %d, erwartet 1", src.calls)
	}
	now = now.Add(cacheFor + time.Second)
	if got := r.HomeAssistant(ctx); got != "it" || src.calls != 2 {
		t.Errorf("nach Ablauf: %q, Abrufe %d", got, src.calls)
	}

	// Nicht erreichbar: Umgebungsvariable, sonst Englisch.
	down := NewResolver(&fakeSource{err: errors.New("weg")}, "es")
	if got := down.Default(ctx); got != "es" {
		t.Errorf("Umgebungsvariable: %q", got)
	}
	if got := NewResolver(nil, "").Default(ctx); got != "en" {
		t.Errorf("ohne alles: %q", got)
	}
	// Nicht unterstützte Sprache von Home Assistant zählt nicht.
	if got := NewResolver(&fakeSource{lang: "nl"}, "").Default(ctx); got != "en" {
		t.Errorf("nl: %q", got)
	}
}
