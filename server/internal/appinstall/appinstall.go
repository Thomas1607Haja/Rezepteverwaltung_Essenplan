// Package appinstall liefert die Links zur Installation der Android-App:
// die neueste APK aus den GitHub-Releases und einen Link, der die App in
// Obtainium einträgt (Updates). Die Weboberfläche zeigt sie als QR-Codes
// unter Einstellungen → App installieren. Siehe docs/ANDROID.md.
package appinstall

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// DefaultRepository ist das GitHub-Repository mit den Releases der App.
// In Home Assistant lässt es sich mit der Option app_repository der App
// ändern (z. B. für einen eigenen Fork).
const DefaultRepository = "Thomas1607Haja/Rezepteverwaltung_Essenplan"

// APKName ist der feste Dateiname der APK in jedem Release (siehe
// .github/workflows/android.yml). Nur so funktioniert der Link
// …/releases/latest/download/rezeptplaner.apk dauerhaft.
const APKName = "rezeptplaner.apk"

// versionRegex löst die Version aus dem Tag des Releases.
const versionRegex = `[0-9.]+$`

// PackageName der Android-App (applicationId in android/app/build.gradle.kts).
const PackageName = "io.github.thomas1607haja.rezeptplaner"

// validRepository: "besitzer/name", wie GitHub es erlaubt.
var validRepository = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9._-]{1,100}$`)

// LoadRepository liest die Option app_repository aus options.json im
// Datenordner (dorthin schreibt Home Assistant die Optionen der App).
// Fehlt sie oder ist sie ungültig, gilt DefaultRepository.
func LoadRepository(dataDir string) string {
	data, err := os.ReadFile(filepath.Join(dataDir, "options.json"))
	if err != nil {
		return DefaultRepository
	}
	var opts struct {
		AppRepository string `json:"app_repository"`
	}
	if err := json.Unmarshal(data, &opts); err != nil {
		slog.Warn("options.json nicht lesbar", "error", err)
		return DefaultRepository
	}
	repo := strings.Trim(strings.TrimSpace(opts.AppRepository), "/")
	repo = strings.TrimPrefix(repo, "https://github.com/")
	if repo == "" {
		return DefaultRepository
	}
	if !validRepository.MatchString(repo) {
		slog.Warn("Option app_repository ungültig, nehme Standard", "wert", repo, "standard", DefaultRepository)
		return DefaultRepository
	}
	return repo
}

// Links sind die Antwort von GET /api/v1/app-install.
type Links struct {
	Repository string `json:"repository"`
	// APKURL zeigt immer auf die APK des neuesten Releases.
	APKURL string `json:"apk_url"`
	// ReleasesURL ist die Übersicht aller Releases.
	ReleasesURL string `json:"releases_url"`
	// ObtainiumURL trägt die App in Obtainium ein (über die
	// Weiterleitungsseite von Obtainium, weil Kamera-Apps Links mit
	// obtainium:// meist nicht öffnen).
	ObtainiumURL string `json:"obtainium_url"`
	// ObtainiumDownloadURL: Obtainium selbst, direkt die APK für 64-Bit-ARM
	// (alle aktuellen Handys, z. B. Galaxy A54) – die Release-Seite
	// enthält über 30 Dateien.
	ObtainiumDownloadURL string `json:"obtainium_download_url"`
}

// NewLinks baut die Links für das Repository repo ("besitzer/name").
func NewLinks(repo string) Links {
	base := "https://github.com/" + repo
	return Links{
		Repository:           repo,
		APKURL:               base + "/releases/latest/download/" + APKName,
		ReleasesURL:          base + "/releases",
		ObtainiumURL:         obtainiumURL(repo),
		ObtainiumDownloadURL: "https://github.com/ImranR98/Obtainium/releases/latest/download/app-arm64-v8a-release.apk",
	}
}

// obtainiumURL baut einen Link im Format
// https://apps.obtainium.imranr.dev/redirect?r=obtainium://app/<JSON>
// (siehe https://wiki.obtainium.imranr.dev/deep_links/). Das JSON ist die
// Konfiguration, wie Obtainium sie exportiert; additionalSettings ist
// dort ein JSON-Text, kein Objekt.
func obtainiumURL(repo string) string {
	// So knapp wie möglich, damit der QR-Code gut lesbar bleibt. Filter
	// für Release-Titel und APK braucht es nicht: Releases gibt es nur für
	// die Android-App (Tags v… der Home-Assistant-App bekommen keine), und
	// jedes enthält genau eine APK. Nur die Version muss aus dem Tag
	// "android-v1.2.3" gelöst werden, damit sie zu versionName passt.
	settings, _ := json.Marshal(map[string]string{"versionExtractionRegEx": versionRegex})
	owner, _, _ := strings.Cut(repo, "/")
	config, _ := json.Marshal(map[string]string{
		"id":                 PackageName,
		"url":                "https://github.com/" + repo,
		"author":             owner,
		"name":               "Rezeptplaner",
		"additionalSettings": string(settings),
	})
	// Nur einmal kodiert: Die Weiterleitungsseite liest r und kodiert den
	// Link für obtainium:// selbst (so geprüft 2026-10-23). Doppelt kodiert
	// wäre der QR-Code fast doppelt so dicht.
	return "https://apps.obtainium.imranr.dev/redirect?r=obtainium://app/" + url.QueryEscape(string(config))
}

// Handler stellt GET /api/v1/app-install und POST /api/v1/app-install/check bereit.
type Handler struct {
	links  Links
	prober prober
}

// NewHandler erzeugt die Endpunkte für das Repository repo; p ruft beim
// Prüfen die Adresse für die App ab.
func NewHandler(repo string, p prober) *Handler {
	return &Handler{links: NewLinks(repo), prober: p}
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/app-install", h.get)
	mux.HandleFunc("POST /api/v1/app-install/check", h.check)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.links)
}

// check prüft eine Adresse für die App, Anfrage {"url": "https://…"}.
func (h *Handler) check(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL string `json:"url"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, Check(r.Context(), h.prober, in.URL))
}
