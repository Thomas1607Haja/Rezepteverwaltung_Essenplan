package fileimport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/pantry"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// pendingTTL: So lange liegen hochgeladene Dateien bis zur Bestätigung.
const pendingTTL = time.Hour

// maxRecipesPerImport begrenzt die Rezepte aller Dateien eines Uploads.
const maxRecipesPerImport = 100

// validToken prüft die Kennung aus der Adresse, bevor sie zum Ordnernamen
// wird – so kommt niemand mit "../" an andere Ordner.
var validToken = regexp.MustCompile(`^[0-9a-f]{32}$`)

// Importer verwaltet die hochgeladenen Dateien bis zur Bestätigung. Sie
// liegen in einem eigenen Ordner im Datenordner, je Upload ein Unterordner
// mit zufälliger Kennung. Im Arbeitsspeicher wird nichts gehalten.
type Importer struct {
	dir     string
	recipes *recipes.Store
	pantry  *pantry.Store
	images  *images.Store
	// fetcher lädt Fotos von den Quellseiten (webimage.go); nil = aus.
	fetcher *fetch.Client
	now     func() time.Time
	// mu: immer nur ein Upload bzw. Import gleichzeitig. Das hält die
	// Speicherspitze klein und verhindert, dass zweimal Bestätigen
	// dieselben Rezepte doppelt anlegt.
	mu sync.Mutex
}

// NewImporter legt den Ordner dir an und räumt Reste früherer Läufe weg.
// fetcher lädt Fotos von den Quellseiten der Rezepte; nil schaltet das ab.
func NewImporter(dir string, r *recipes.Store, p *pantry.Store, img *images.Store, fetcher *fetch.Client) (*Importer, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("importordner anlegen: %w", err)
	}
	im := &Importer{dir: dir, recipes: r, pantry: p, images: img, fetcher: fetcher, now: time.Now}
	im.cleanup()
	return im, nil
}

// cleanup löscht Uploads, die älter als pendingTTL sind. Läuft bei jedem
// Upload statt im Hintergrund – im Leerlauf passiert nichts.
func (im *Importer) cleanup() {
	entries, err := os.ReadDir(im.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err == nil && im.now().Sub(info.ModTime()) > pendingTTL {
			os.RemoveAll(filepath.Join(im.dir, e.Name()))
		}
	}
}

// pendingDir liefert den Ordner zur Kennung oder "", wenn es ihn nicht
// (mehr) gibt.
func (im *Importer) pendingDir(token string) string {
	if !validToken.MatchString(token) {
		return ""
	}
	dir := filepath.Join(im.dir, token)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() || im.now().Sub(info.ModTime()) > pendingTTL {
		return ""
	}
	return dir
}

func newToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("kennung erzeugen: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// uploadedFile ist eine gespeicherte Datei eines Uploads. Die Liste liegt
// als files.json im Ordner des Uploads.
type uploadedFile struct {
	Name string `json:"name"` // Name beim Hochladen, nur zur Anzeige
	Path string `json:"path"` // Dateiname im Ordner, z. B. "01.upload"
}

// Preview ist die Antwort auf einen Upload. Gespeichert ist noch nichts.
type Preview struct {
	// Token ist beim Bestätigen anzugeben; leer, wenn nichts importierbar ist.
	Token      string          `json:"token"`
	ExpiresAt  *time.Time      `json:"expires_at"`
	Recipes    []PreviewRecipe `json:"recipes"`
	FileErrors []FileError     `json:"file_errors"`
	// UnknownCategories: alle unbekannten Kategorien des Uploads, jede nur
	// einmal (ohne Rücksicht auf Groß-/Kleinschreibung und Umlaute) – so
	// fragt die Oberfläche je Name nur einmal.
	UnknownCategories []string `json:"unknown_categories"`
}

// PreviewRecipe ist ein Rezept der Vorschau. Mit Errors lässt es sich
// nicht importieren; Warnings sind nur Hinweise.
type PreviewRecipe struct {
	Index int    `json:"index"`
	File  string `json:"file"`
	// Recipe ist das Rezept, wie es gespeichert würde (Kategorien
	// aufgelöst, Vorrat abgeglichen), noch ohne ID.
	Recipe recipes.Recipe `json:"recipe"`
	// Thumbnail: Es gibt ein Vorschaubild unter
	// GET /api/v1/import/file/{token}/images/{index}.
	Thumbnail bool `json:"thumbnail"`
	// WebImage: Das Foto kommt von der Quellseite (source.url), z. B.
	// "chefkoch.de"; leer = Bild aus der Datei bzw. keins.
	WebImage   string            `json:"web_image"`
	Errors     []string          `json:"errors"`
	Warnings   []string          `json:"warnings"`
	Duplicates []recipes.Summary `json:"duplicates"`
	// UnknownCategories: Kategorien der Datei, die es im Rezeptplaner nicht
	// gibt. Beim Bestätigen lassen sie sich zuordnen oder neu anlegen
	// (CategoryChoice), sonst fallen sie weg.
	UnknownCategories []string `json:"unknown_categories"`
}

// CategoryChoice sagt beim Bestätigen, was mit einer unbekannten Kategorie
// geschieht: TagID > 0 = dieser vorhandenen zuordnen, NewInGroup > 0 = unter
// ihrem Namen in dieser Gruppe neu anlegen, beides 0 = weglassen.
type CategoryChoice struct {
	TagID      int64 `json:"tag_id"`
	NewInGroup int64 `json:"new_in_group"`
}

// FileError: Eine Datei ließ sich gar nicht lesen.
type FileError struct {
	File  string `json:"file"`
	Error string `json:"error"`
}

// candidate ist ein Rezept beim Auswerten eines Uploads.
type candidate struct {
	PreviewRecipe
	src   *source
	image string
}

// analyze liest alle Dateien des Uploads im Ordner dir und prüft die
// Rezepte. Vorschau und Import rufen es gleichermaßen auf; so gelten beim
// Bestätigen genau die Regeln der Vorschau. close gibt die ZIP-Dateien frei.
func (im *Importer) analyze(ctx context.Context, dir string, b *budget) (cands []*candidate, fileErrors []FileError, close func(), err error) {
	var sources []*source
	close = func() {
		for _, s := range sources {
			s.Close()
		}
	}
	fileErrors = []FileError{}

	data, err := os.ReadFile(filepath.Join(dir, "files.json"))
	if err != nil {
		return nil, nil, close, fmt.Errorf("upload lesen: %w", err)
	}
	var files []uploadedFile
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, nil, close, fmt.Errorf("upload lesen: %w", err)
	}

	for _, f := range files {
		src, err := openSource(filepath.Join(dir, f.Path), b)
		if err != nil {
			fileErrors = append(fileErrors, FileError{File: f.Name, Error: err.Error()})
			continue
		}
		sources = append(sources, src)
		for _, raw := range src.doc.Recipes {
			p := parseRecipe(raw)
			c := &candidate{src: src, image: p.image, PreviewRecipe: PreviewRecipe{
				Index: len(cands), File: f.Name, Recipe: p.recipe,
				Errors: append([]string{}, p.errors...), Warnings: append([]string{}, p.warnings...),
				Duplicates: []recipes.Summary{}, UnknownCategories: []string{},
			}}
			if len(cands) >= maxRecipesPerImport {
				c.Errors = append(c.Errors, fmt.Sprintf("Mehr als %d Rezepte in einem Import", maxRecipesPerImport))
			}
			c.Recipe.Tags = []recipes.Tag{}
			if err := im.resolveCategories(ctx, c, p.categories); err != nil {
				return nil, nil, close, err
			}
			cands = append(cands, c)
		}
	}

	matcher, err := im.pantry.Matcher(ctx)
	if err != nil {
		return nil, nil, close, err
	}
	seenTitles := make(map[string]int)
	for _, c := range cands {
		// Zutaten aus dem Vorrat nicht einkaufen – wie beim URL-Import.
		for i := range c.Recipe.Ingredients {
			if matcher.Matches(c.Recipe.Ingredients[i].Name) {
				c.Recipe.Ingredients[i].ShoppingRelevant = false
			}
		}
		if c.Recipe.Title == "" {
			continue
		}
		if c.Duplicates, err = im.recipes.FindDuplicates(ctx, c.Recipe.Title, c.Recipe.SourceURL); err != nil {
			return nil, nil, close, err
		}
		key := database.Fold(c.Recipe.Title)
		if first, ok := seenTitles[key]; ok {
			c.Warnings = append(c.Warnings, fmt.Sprintf("Gleicher Titel wie Rezept %d dieses Imports", first+1))
		} else {
			seenTitles[key] = c.Index
		}
	}
	return cands, fileErrors, close, nil
}

// resolveCategories ordnet die Kategorienamen den vorhandenen Kategorien
// zu. Unbekannte werden nicht angelegt, sondern gemeldet und weggelassen.
func (im *Importer) resolveCategories(ctx context.Context, c *candidate, names []string) error {
	if len(names) == 0 {
		return nil
	}
	groups, err := im.recipes.ListTags(ctx)
	if err != nil {
		return err
	}
	byName := make(map[string]recipes.Tag)
	for _, g := range groups {
		for _, t := range g.Tags {
			if _, dup := byName[database.Fold(t.Name)]; !dup {
				byName[database.Fold(t.Name)] = t
			}
		}
	}
	for _, name := range names {
		t, ok := byName[database.Fold(name)]
		switch {
		case !ok:
			c.Warnings = append(c.Warnings, fmt.Sprintf("Kategorie „%s“ gibt es nicht – zuordnen oder weglassen", name))
			c.UnknownCategories = append(c.UnknownCategories, name)
		case !slices.ContainsFunc(c.Recipe.Tags, func(x recipes.Tag) bool { return x.ID == t.ID }):
			c.Recipe.Tags = append(c.Recipe.Tags, t)
		}
	}
	return nil
}

// thumbPath ist das Vorschaubild eines Rezepts im Ordner des Uploads.
func thumbPath(dir string, index int) string {
	return filepath.Join(dir, "thumb-"+strconv.Itoa(index)+".jpg")
}

// preview wertet einen frisch gespeicherten Upload aus und legt die
// Vorschaubilder ab.
func (im *Importer) preview(ctx context.Context, token, dir string) (Preview, error) {
	b := newBudget()
	cands, fileErrors, closeAll, err := im.analyze(ctx, dir, b)
	defer closeAll()
	if err != nil {
		return Preview{}, err
	}

	result := Preview{Recipes: []PreviewRecipe{}, FileErrors: fileErrors, UnknownCategories: []string{}}
	seenUnknown := make(map[string]bool)
	// Hinweise zum Bild erst nach dem Versuch mit der Quellseite.
	imageWarnings := make(map[int]string)
	var fromWeb []*candidate
	for _, c := range cands {
		if c.image != "" {
			if thumb, err := im.loadImage(c, b); err != nil {
				imageWarnings[c.Index] = err.Error()
			} else if err := os.WriteFile(thumbPath(dir, c.Index), thumb, 0o600); err != nil {
				return Preview{}, fmt.Errorf("vorschaubild speichern: %w", err)
			} else {
				c.Thumbnail = true
			}
		} else {
			imageWarnings[c.Index] = "Kein Bild"
		}
		hasPhoto := c.Thumbnail && c.Recipe.ImageKind != recipes.ImageIllustration
		if wantsWebImage(c, hasPhoto) {
			fromWeb = append(fromWeb, c)
		}
	}
	found := im.loadWebImages(ctx, dir, fromWeb)

	importable := false
	for _, c := range cands {
		if site, ok := found[c.Index]; ok {
			hint := fmt.Sprintf("Foto von %s wird übernommen", site)
			if c.Recipe.ImageKind == recipes.ImageIllustration && c.Thumbnail {
				hint += " (ersetzt die Illustration)"
			}
			c.Warnings = append(c.Warnings, hint)
			c.Thumbnail, c.WebImage, c.Recipe.ImageKind = true, site, recipes.ImagePhoto
		} else if w, ok := imageWarnings[c.Index]; ok && len(c.Errors) == 0 {
			c.Warnings = append(c.Warnings, w)
		}
		if len(c.Errors) == 0 {
			importable = true
			for _, name := range c.UnknownCategories {
				if key := database.Fold(name); !seenUnknown[key] {
					seenUnknown[key] = true
					result.UnknownCategories = append(result.UnknownCategories, name)
				}
			}
		}
		result.Recipes = append(result.Recipes, c.PreviewRecipe)
	}
	if importable {
		expires := im.now().Add(pendingTTL)
		result.Token, result.ExpiresAt = token, &expires
	}
	return result, nil
}

// loadImage liest das Bild eines Rezepts und liefert das Vorschaubild.
// Dabei wird der echte Bildtyp am Inhalt geprüft.
func (im *Importer) loadImage(c *candidate, b *budget) ([]byte, error) {
	data, err := c.src.image(c.image, b)
	if err != nil {
		return nil, err
	}
	thumb, err := im.images.Preview(data)
	if err != nil {
		return nil, fmt.Errorf("Bild %s ist unbrauchbar (%v), das Rezept kommt ohne Bild", c.image, err)
	}
	return thumb, nil
}

// Result ist die Antwort auf das Bestätigen.
type Result struct {
	Created []CreatedRecipe `json:"created"`
	Failed  []FailedRecipe  `json:"failed"`
	// Warnings: z. B. eine Kategorie, die sich nicht anlegen ließ.
	Warnings []string `json:"warnings"`
}

// applyCategoryChoices legt die gewünschten neuen Kategorien an und liefert
// je unbekanntem Namen (Fold) die Kategorie, die die Rezepte bekommen.
// Probleme werden Hinweise; die Rezepte kommen dann ohne diese Kategorie.
func (im *Importer) applyCategoryChoices(ctx context.Context, choices map[string]CategoryChoice) (map[string]recipes.Tag, []string) {
	mapped := make(map[string]recipes.Tag)
	var warnings []string
	if len(choices) == 0 {
		return mapped, warnings
	}
	groups, err := im.recipes.ListTags(ctx)
	if err != nil {
		return mapped, []string{err.Error()}
	}
	byID := make(map[int64]recipes.Tag)
	byName := make(map[string]recipes.Tag)
	for _, g := range groups {
		for _, t := range g.Tags {
			byID[t.ID] = t
			byName[database.Fold(t.Name)] = t
		}
	}
	// Feste Reihenfolge, damit Hinweise und neue IDs nicht zufällig sind.
	names := make([]string, 0, len(choices))
	for name := range choices {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		choice := choices[name]
		key := database.Fold(name)
		switch {
		case choice.TagID > 0:
			if t, ok := byID[choice.TagID]; ok {
				mapped[key] = t
			} else {
				warnings = append(warnings, fmt.Sprintf("Kategorie %d für „%s“ gibt es nicht mehr – weggelassen", choice.TagID, name))
			}
		case choice.NewInGroup > 0:
			// Inzwischen angelegt (z. B. zweimal bestätigt)? Dann diese nehmen.
			if t, ok := byName[key]; ok {
				mapped[key] = t
				continue
			}
			t, err := im.recipes.CreateTag(ctx, name, choice.NewInGroup)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("Kategorie „%s“ nicht angelegt: %v", name, err))
				continue
			}
			byName[key] = t
			mapped[key] = t
		}
	}
	return mapped, warnings
}

// CreatedRecipe ist ein angelegtes Rezept; Warnings z. B. bei einem Bild,
// das sich nicht speichern ließ.
type CreatedRecipe struct {
	Index    int      `json:"index"`
	ID       int64    `json:"id"`
	Title    string   `json:"title"`
	Warnings []string `json:"warnings"`
}

// FailedRecipe ist ein ausgewähltes Rezept, das nicht angelegt wurde.
type FailedRecipe struct {
	Index int    `json:"index"`
	Title string `json:"title"`
	Error string `json:"error"`
}

// errExpired: Die Kennung ist unbekannt oder abgelaufen.
var errExpired = errors.New("Import nicht gefunden oder abgelaufen – bitte die Datei erneut hochladen")

// confirm legt die ausgewählten Rezepte an und räumt den Upload weg.
// choices sagt, was mit unbekannten Kategorien geschieht.
func (im *Importer) confirm(ctx context.Context, token string, selected []int, choices map[string]CategoryChoice) (Result, error) {
	dir := im.pendingDir(token)
	if dir == "" {
		return Result{}, errExpired
	}
	b := newBudget()
	cands, _, closeAll, err := im.analyze(ctx, dir, b)
	defer closeAll()
	if err != nil {
		return Result{}, err
	}

	result := Result{Created: []CreatedRecipe{}, Failed: []FailedRecipe{}, Warnings: []string{}}
	mapped, warnings := im.applyCategoryChoices(ctx, choices)
	result.Warnings = append(result.Warnings, warnings...)
	for _, index := range selected {
		if index < 0 || index >= len(cands) {
			result.Failed = append(result.Failed, FailedRecipe{Index: index, Error: "Kein solches Rezept in diesem Import"})
			continue
		}
		c := cands[index]
		if len(c.Errors) > 0 {
			result.Failed = append(result.Failed, FailedRecipe{Index: index, Title: c.Recipe.Title, Error: c.Errors[0]})
			continue
		}
		for _, name := range c.UnknownCategories {
			if tag, ok := mapped[database.Fold(name)]; ok &&
				!slices.ContainsFunc(c.Recipe.Tags, func(x recipes.Tag) bool { return x.ID == tag.ID }) {
				c.Recipe.Tags = append(c.Recipe.Tags, tag)
			}
		}
		created, err := im.recipes.Create(ctx, c.Recipe)
		if err != nil {
			slog.Warn("Datei-Import: Rezept nicht angelegt", "titel", c.Recipe.Title, "error", err)
			result.Failed = append(result.Failed, FailedRecipe{Index: index, Title: c.Recipe.Title, Error: err.Error()})
			continue
		}
		done := CreatedRecipe{Index: index, ID: created.ID, Title: created.Title, Warnings: []string{}}
		// Foto von der Quellseite aus der Vorschau, sonst das Bild der Datei.
		fromWeb, warning := im.saveWebImage(ctx, dir, c, created.ID)
		if warning != "" {
			done.Warnings = append(done.Warnings, warning)
		}
		if (!fromWeb || warning != "") && c.image != "" {
			if warning := im.saveImage(ctx, c, created.ID, b); warning != "" {
				done.Warnings = append(done.Warnings, warning)
			}
		}
		result.Created = append(result.Created, done)
	}

	// Erledigt: Der Upload wird nicht mehr gebraucht. Ein zweites
	// Bestätigen liefert "abgelaufen" statt doppelter Rezepte.
	closeAll()
	if err := os.RemoveAll(dir); err != nil {
		slog.Warn("Datei-Import: Upload nicht gelöscht", "ordner", dir, "error", err)
	}
	return result, nil
}

// saveImage speichert das Bild eines angelegten Rezepts; zurück kommt ein
// Hinweis, falls das nicht klappt (das Rezept bleibt dann ohne Bild).
func (im *Importer) saveImage(ctx context.Context, c *candidate, recipeID int64, b *budget) string {
	data, err := c.src.image(c.image, b)
	if err != nil {
		return err.Error()
	}
	large, thumb, err := im.images.Save(bytes.NewReader(data))
	if err != nil {
		return fmt.Sprintf("Bild nicht gespeichert: %v", err)
	}
	if _, err := im.recipes.SetImage(ctx, recipeID, large, thumb, c.Recipe.ImageKind); err != nil {
		im.images.Delete(large, thumb)
		return fmt.Sprintf("Bild nicht gespeichert: %v", err)
	}
	return ""
}
