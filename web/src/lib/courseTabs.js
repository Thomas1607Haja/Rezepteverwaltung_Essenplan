// Schnellfilter-Reiter nach Gang (docs/PLANUNG.md, Schritt 9.1). Die Reiter
// fassen Gänge der Gang-Gruppe (role "course") zusammen; Kategorien, die es
// unter diesem Namen nicht (mehr) gibt, fallen weg – ein Reiter ohne
// passende Kategorie erscheint nicht.

const TABS = [
  { key: 'main', label: 'Hauptgerichte', names: ['Hauptgericht'] },
  { key: 'starters', label: 'Vorspeisen, Suppen & Salate', names: ['Vorspeise', 'Suppe/Eintopf', 'Salat'] },
  { key: 'desserts', label: 'Nachspeisen', names: ['Nachspeise'] },
  { key: 'baking', label: 'Backen', names: ['Kuchen & Torten', 'Gebäck', 'Brot'] },
];

const fold = (s) => s.toLocaleLowerCase('de').replace(/ä/g, 'a').replace(/ö/g, 'o').replace(/ü/g, 'u').replace(/ß/g, 'ss');

/**
 * Reiter mit den IDs ihrer Gänge: [{ key, label, ids }], vorn „Alle“.
 * withNone: hinten „Ohne Gang“ (key 'none', nur Rezeptübersicht im Web).
 */
export function courseTabs(groups, withNone = false) {
  const course = groups.find((g) => g.role === 'course');
  const tabs = [{ key: 'all', label: 'Alle', ids: [] }];
  if (!course) return tabs;
  for (const tab of TABS) {
    const ids = course.tags.filter((t) => tab.names.some((n) => fold(n) === fold(t.name))).map((t) => t.id);
    if (ids.length > 0) tabs.push({ ...tab, ids });
  }
  if (withNone) tabs.push({ key: 'none', label: 'Ohne Gang', ids: [] });
  return tabs;
}
