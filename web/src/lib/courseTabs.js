// Schnellfilter-Reiter nach Gang (docs/PLANUNG.md, Schritt 9.1). Die Reiter
// fassen Gänge der Gang-Gruppe (role "course") über ihren festen Schlüssel
// (course_key, Migration 018) zusammen – die Namen sind Daten des Nutzers.
// Ein Reiter ohne passende Kategorie erscheint nicht.
import { t } from './i18n.svelte.js';

const TABS = [
  { key: 'main', courses: ['main'] },
  { key: 'starters', courses: ['starter', 'soup', 'salad'] },
  { key: 'desserts', courses: ['dessert'] },
  { key: 'baking', courses: ['cake', 'pastry', 'bread'] },
];

/**
 * Reiter mit den IDs ihrer Gänge: [{ key, label, ids }], vorn „Alle“.
 * withNone: hinten „Ohne Gang“ (key 'none', nur Rezeptübersicht im Web).
 */
export function courseTabs(groups, withNone = false) {
  const course = groups.find((g) => g.role === 'course');
  const tabs = [{ key: 'all', label: t('courses.all'), ids: [] }];
  if (!course) return tabs;
  for (const tab of TABS) {
    const ids = course.tags.filter((tag) => tab.courses.includes(tag.course_key)).map((tag) => tag.id);
    if (ids.length > 0) tabs.push({ key: tab.key, label: t(`courses.${tab.key}`), ids });
  }
  if (withNone) tabs.push({ key: 'none', label: t('courses.none'), ids: [] });
  return tabs;
}
