package recipes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// groupID sucht die ID einer Gruppe über ihren Namen.
func groupID(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	var id int64
	if err := s.db.QueryRow("SELECT id FROM tag_groups WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatalf("Gruppe %q: %v", name, err)
	}
	return id
}

// tagNames liefert die Kategorien einer Gruppe in Reihenfolge.
func tagNames(t *testing.T, s *Store, group string) []string {
	t.Helper()
	groups, err := s.AdminTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range groups {
		if g.Name == group {
			names := []string{}
			for _, tag := range g.Tags {
				names = append(names, tag.Name)
			}
			return names
		}
	}
	t.Fatalf("Gruppe %q fehlt", group)
	return nil
}

func TestTagAdminNamesAreUniqueAcrossGroups(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	art := groupID(t, s, "Art")
	var verr *ValidationError
	for _, name := range []string{"pasta", "  PASTA ", "Geflugel", "geflügel"} {
		if _, err := s.CreateTag(ctx, name, art); !errors.As(err, &verr) {
			t.Errorf("%q: Fehler %v, erwartet Namenskonflikt", name, err)
		}
	}
	if _, err := s.CreateTagGroup(ctx, "hauptzutat"); !errors.As(err, &verr) {
		t.Errorf("Gruppe doppelt: %v", err)
	}
	tag, err := s.CreateTag(ctx, "  Ofen   gericht ", art)
	if err != nil || tag.Name != "Ofen gericht" || tag.Group != "Art" {
		t.Fatalf("CreateTag: %+v %v", tag, err)
	}
	// Umbenennen in den eigenen Namen (andere Schreibweise) ist erlaubt.
	if err := s.UpdateTag(ctx, tag.ID, "Ofengericht", art); err != nil {
		t.Errorf("umbenennen: %v", err)
	}
	if err := s.UpdateTag(ctx, tag.ID, "Rind", art); !errors.As(err, &verr) {
		t.Errorf("umbenennen auf vorhandenen Namen: %v", err)
	}
}

// Umbenennen, Verschieben, Löschen und Zusammenführen müssen über
// updated_at (und deletions) in die App kommen.
func TestTagAdminChangesReachTheApp(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	// Später als die echte Uhrzeit, mit der Create die Rezepte anlegt – sonst
	// hinge der Test von der Tageszeit ab.
	clock := time.Now().Add(time.Hour).Truncate(time.Second)
	s.now = func() time.Time { return clock }
	created, err := s.Create(ctx, lasagne(t, s)) // Hackfleisch, Auflauf/Ofen
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.Create(ctx, Recipe{Title: "Salat", BaseServings: 2, Tags: []Tag{{ID: tagID(t, s, "Salat")}}})
	if err != nil {
		t.Fatal(err)
	}

	changed := func(step string) []int64 {
		t.Helper()
		list, err := s.ChangedSince(ctx, database.FormatTime(clock))
		if err != nil {
			t.Fatal(err)
		}
		ids := []int64{}
		for _, r := range list {
			ids = append(ids, r.ID)
		}
		return ids
	}
	step := func(name string, fn func() error, want []int64) {
		t.Helper()
		clock = clock.Add(time.Minute)
		if err := fn(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := changed(name)
		if len(got) != len(want) || (len(want) > 0 && got[0] != want[0]) {
			t.Errorf("%s: geänderte Rezepte %v, erwartet %v", name, got, want)
		}
	}
	hack, ofen := tagID(t, s, "Hackfleisch"), tagID(t, s, "Auflauf/Ofen")
	art, haupt := groupID(t, s, "Art"), groupID(t, s, "Hauptzutat")

	step("Kategorie umbenennen", func() error { return s.UpdateTag(ctx, hack, "Hack", haupt) }, []int64{created.ID})
	step("Gruppe umbenennen", func() error { return s.RenameTagGroup(ctx, art, "Gerichtart") }, []int64{created.ID, other.ID})
	step("Gruppen sortieren (ändert keine Rezepte)", func() error {
		groups, _ := s.AdminTags(ctx)
		ids := []int64{}
		for i := len(groups) - 1; i >= 0; i-- {
			ids = append(ids, groups[i].ID)
		}
		return s.OrderTagGroups(ctx, ids)
	}, nil)
	step("Zusammenführen", func() error { return s.MergeTag(ctx, ofen, tagID(t, s, "Salat")) }, []int64{created.ID})

	got, _ := s.Get(ctx, created.ID)
	names := []string{}
	for _, tag := range got.Tags {
		names = append(names, tag.Group+"/"+tag.Name)
	}
	if len(names) != 2 || names[0] != "Gerichtart/Salat" || names[1] != "Hauptzutat/Hack" {
		// Gruppen jetzt umgekehrt sortiert: Gerichtart vor Hauptzutat.
		t.Errorf("Kategorien nach Zusammenführen: %v", names)
	}
	var deleted int
	s.db.QueryRow("SELECT count(*) FROM deletions WHERE table_name = 'tags' AND record_id = ?", ofen).Scan(&deleted)
	if deleted != 1 {
		t.Errorf("Löschung von Auflauf/Ofen nicht vermerkt")
	}

	step("Kategorie löschen", func() error { return s.DeleteTag(ctx, hack) }, []int64{created.ID})
	got, _ = s.Get(ctx, created.ID)
	if len(got.Tags) != 1 || got.Title != "Lasagne" {
		t.Errorf("nach Löschen: %+v", got.Tags)
	}
}

func TestTagAdminGroupsAndOrder(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	geschmack := groupID(t, s, "Geschmack")
	anlass, err := s.CreateTagGroup(ctx, "Anlass")
	if err != nil {
		t.Fatal(err)
	}
	// Ziehen zwischen Gruppen: „Süß“ wandert nach Anlass.
	suess, herzhaft := tagID(t, s, "Süß"), tagID(t, s, "Herzhaft")
	if err := s.OrderTags(ctx, anlass, []int64{suess}); err != nil {
		t.Fatal(err)
	}
	if got := tagNames(t, s, "Anlass"); len(got) != 1 || got[0] != "Süß" {
		t.Errorf("Anlass: %v", got)
	}
	// Eine Gruppe darf beim Sortieren keine Kategorie verlieren.
	var verr *ValidationError
	if err := s.OrderTags(ctx, anlass, []int64{herzhaft}); !errors.As(err, &verr) {
		t.Errorf("Kategorie fehlt in der Liste: %v", err)
	}
	if err := s.DeleteTagGroup(ctx, anlass, 0, false); !errors.Is(err, ErrGroupNotEmpty) {
		t.Errorf("nicht leere Gruppe: %v", err)
	}
	if err := s.DeleteTagGroup(ctx, anlass, geschmack, false); err != nil {
		t.Fatal(err)
	}
	if got := tagNames(t, s, "Geschmack"); len(got) != 2 || got[1] != "Süß" {
		t.Errorf("Geschmack nach Verschieben: %v", got)
	}
	if err := s.DeleteTagGroup(ctx, geschmack, 0, true); err != nil {
		t.Fatal(err)
	}
	groups, _ := s.ListTags(ctx)
	for _, g := range groups {
		if g.Name == "Geschmack" {
			t.Errorf("Gruppe noch da")
		}
	}
	// Leere Gruppen fehlen in ListTags (App), nicht aber in der Verwaltung.
	empty, _ := s.CreateTagGroup(ctx, "Leer")
	list, _ := s.ListTags(ctx)
	admin, _ := s.AdminTags(ctx)
	if len(admin) != len(list)+1 || admin[len(admin)-1].ID != empty {
		t.Errorf("leere Gruppe: ListTags %d, AdminTags %d", len(list), len(admin))
	}
}
