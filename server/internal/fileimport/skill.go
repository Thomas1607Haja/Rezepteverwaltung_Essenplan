package fileimport

import (
	"strconv"
	"strings"
)

// CurrentSkillVersion ist die Version der Claude-Skill, die zu diesem Server
// gehört (GENERATOR in skills/rezeptplaner-import/scripts/build_import.py;
// TestSkillVersionMatchesBuildScript prüft, dass beide gleich sind).
//
// Die Skill schreibt ihre Version ins Feld "generator", z. B.
// "claude-skill rezeptplaner-import 2.4". Ein eigenes Feld gibt es bewusst
// nicht: Ältere Server lehnen unbekannte Felder ab (decodeStrict), neue
// Skill-Dateien sollen dort aber weiter funktionieren.
const CurrentSkillVersion = "2.4"

// skillGenerator ist der Anfang von "generator" in Dateien der Skill.
const skillGenerator = "claude-skill rezeptplaner-import "

// SkillCheck steht in der Vorschau, wenn mindestens eine Datei von der
// Claude-Skill stammt.
type SkillCheck struct {
	// Version der Skill aus der Datei; bei mehreren Dateien die älteste.
	Version string `json:"version"`
	// Current ist die Version der Skill, die zu diesem Server gehört.
	Current string `json:"current"`
	// UpdateAvailable: Die Datei kommt von einer älteren Skill.
	UpdateAvailable bool `json:"update_available"`
	// DownloadURL lädt immer die neueste Skill (Release-Datei).
	DownloadURL string `json:"download_url"`
}

// skillVersion liest die Version der Skill aus "generator"; ok = false,
// wenn die Datei nicht von der Skill stammt.
func skillVersion(generator string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(generator), skillGenerator)
	if !ok {
		return "", false
	}
	v := strings.TrimSpace(rest)
	if _, ok := parseVersion(v); !ok {
		return "", false
	}
	return v, true
}

// checkSkill bewertet die Generatoren aller Dateien eines Uploads; nil, wenn
// keine davon von der Skill stammt.
func checkSkill(generators []string, downloadURL string) *SkillCheck {
	oldest := ""
	for _, g := range generators {
		v, ok := skillVersion(g)
		if ok && (oldest == "" || compareVersions(v, oldest) < 0) {
			oldest = v
		}
	}
	if oldest == "" {
		return nil
	}
	return &SkillCheck{
		Version:         oldest,
		Current:         CurrentSkillVersion,
		UpdateAvailable: compareVersions(oldest, CurrentSkillVersion) < 0,
		DownloadURL:     downloadURL,
	}
}

// parseVersion zerlegt "2.4" oder "2.4.1" in Zahlen.
func parseVersion(v string) ([]int, bool) {
	parts := strings.Split(v, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, false
		}
		nums[i] = n
	}
	return nums, true
}

// compareVersions vergleicht zwei gültige Versionen: <0, 0 oder >0.
// Fehlende Stellen zählen als 0 ("2.4" == "2.4.0").
func compareVersions(a, b string) int {
	x, _ := parseVersion(a)
	y, _ := parseVersion(b)
	for i := 0; i < len(x) || i < len(y); i++ {
		var p, q int
		if i < len(x) {
			p = x[i]
		}
		if i < len(y) {
			q = y[i]
		}
		if p != q {
			return p - q
		}
	}
	return 0
}
