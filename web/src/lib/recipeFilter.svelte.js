// Filter der Rezeptübersicht. Der Zustand liegt außerhalb der Seite, damit
// Suche und Filter erhalten bleiben, wenn man ein Rezept öffnet und zur
// Übersicht zurückkehrt.

export const recipeFilter = $state({
  query: '',
  tagIds: [],
  favoritesOnly: false,
  illustrationsOnly: false, // nur gezeichnete Bilder (zum Ersetzen durch Fotos)
  sort: 'name', // 'name', 'newest', 'last_planned' oder 'most_planned'
});

/** Schaltet eine Kategorie im Filter an oder aus. */
export function toggleTag(id) {
  const i = recipeFilter.tagIds.indexOf(id);
  if (i === -1) recipeFilter.tagIds.push(id);
  else recipeFilter.tagIds.splice(i, 1);
}

/** Filtert nur noch nach der Kategorie id (z. B. aus der Kochansicht). */
export function filterByTag(id) {
  resetFilter();
  recipeFilter.tagIds = [id];
}

/** Setzt Suche und Filter zurück (die Sortierung bleibt). */
export function resetFilter() {
  recipeFilter.query = '';
  recipeFilter.tagIds = [];
  recipeFilter.favoritesOnly = false;
  recipeFilter.illustrationsOnly = false;
}

/** Ist irgendein Filter aktiv? */
export function hasActiveFilter() {
  return recipeFilter.query.trim() !== '' || recipeFilter.tagIds.length > 0 || recipeFilter.favoritesOnly ||
    recipeFilter.illustrationsOnly;
}
