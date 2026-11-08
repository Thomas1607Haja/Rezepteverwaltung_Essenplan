// Package apierr beschreibt Fehler und Hinweise der API als stabile Codes
// mit Parametern. Die Oberflächen (Weboberfläche, App) übersetzen den Code
// in die Sprache des Nutzers; der Server liefert keine deutschen Sätze.
//
// Zu jedem Code gehört ein englischer Text im Katalog (catalog.go). Er geht
// als "error" bzw. "message" mit hinaus – für das Protokoll und für ältere
// App-Versionen, die die Codes noch nicht kennen.
//
// Fehlerantwort:
//
//	{"error": "Must be between 1 and 50", "code": "out_of_range",
//	 "field": "servings", "params": {"min": 1, "max": 50}}
package apierr

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
)

// Error ist ein Fehler oder Hinweis mit Code. Field nennt bei Eingabefehlern
// das betroffene Feld (z. B. "servings" oder "ingredients[2].name").
type Error struct {
	Code   string
	Field  string
	Params map[string]any
}

// New erzeugt einen Fehler. params sind Paare aus Name und Wert:
// New("too_many_files", "max", 20).
func New(code string, params ...any) *Error {
	return &Error{Code: code, Params: pairs(params)}
}

// Field erzeugt einen Eingabefehler zu einem Feld.
func Field(field, code string, params ...any) *Error {
	return &Error{Code: code, Field: field, Params: pairs(params)}
}

func pairs(params []any) map[string]any {
	if len(params) == 0 {
		return nil
	}
	if len(params)%2 != 0 {
		panic("apierr: Parameter müssen Paare aus Name und Wert sein")
	}
	m := make(map[string]any, len(params)/2)
	for i := 0; i < len(params); i += 2 {
		m[params[i].(string)] = params[i+1]
	}
	return m
}

// With liefert eine Kopie mit zusätzlichem Parameter, z. B. für einen
// vordefinierten Fehler, der erst später einen Namen bekommt.
func (e *Error) With(name string, value any) *Error {
	c := *e
	c.Params = maps.Clone(e.Params)
	if c.Params == nil {
		c.Params = map[string]any{}
	}
	c.Params[name] = value
	return &c
}

// Message ist der englische Text mit eingesetzten Parametern.
func (e *Error) Message() string {
	text, ok := catalog[e.Code]
	if !ok {
		text = e.Code
	}
	for name, value := range e.Params {
		text = strings.ReplaceAll(text, "{"+name+"}", fmt.Sprint(value))
	}
	return text
}

// Error gehört zum Interface error; mit Feldnamen davor, wie im Protokoll.
func (e *Error) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message()
	}
	return e.Message()
}

// Is vergleicht nur den Code: errors.Is(err, ErrNotFound) gilt auch für
// eine Kopie mit Parametern.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

// MarshalJSON schreibt den Hinweis so, wie ihn die Oberflächen erwarten:
// {"code": …, "field": …, "params": …, "message": "<englisch>"}.
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    string         `json:"code"`
		Field   string         `json:"field,omitempty"`
		Params  map[string]any `json:"params,omitempty"`
		Message string         `json:"message"`
	}{e.Code, e.Field, e.Params, e.Message()})
}

// Messages liefert die englischen Texte einer Liste – für die alten
// Textfelder ("warnings"), die ältere App-Versionen noch lesen.
func Messages(list []*Error) []string {
	out := make([]string, len(list))
	for i, e := range list {
		out[i] = e.Message()
	}
	return out
}

// Known meldet, ob es zu code einen Text im Katalog gibt.
func Known(code string) bool {
	_, ok := catalog[code]
	return ok
}

// Codes liefert alle Codes des Katalogs (für Tests der Sprachdateien).
func Codes() []string {
	out := make([]string, 0, len(catalog))
	for code := range catalog {
		out = append(out, code)
	}
	return out
}

// AsField liefert den Fehler, wenn err ein Eingabefehler zu einem Feld ist
// (früher je Paket ein eigener ValidationError). Fehler ohne Feld wie
// "recipe_not_found" zählen nicht dazu.
func AsField(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) && e.Field != "" {
		return e, true
	}
	return nil, false
}

// From liefert den Code-Fehler in err (auch eingepackt) oder, wenn err
// keinen Code hat, einen neuen Fehler mit fallback.
func From(err error, fallback string) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return New(fallback)
}
