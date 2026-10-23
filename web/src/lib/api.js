// Zugriff auf die JSON-API des Go-Servers.
//
// Die Adressen sind relativ ("api/v1/...", ohne führenden Schrägstrich),
// damit sie unter dem Ingress-Pfad von Home Assistant genauso funktionieren
// wie lokal.

/** Fehler der API mit HTTP-Status und ggf. betroffenem Feld. */
export class ApiError extends Error {
  constructor(status, message, field) {
    super(message);
    this.status = status;
    this.field = field ?? '';
  }
}

/**
 * Schickt eine Anfrage an die API und liefert die JSON-Antwort
 * (bzw. null bei 204 ohne Inhalt). Bei einem Fehler wird ApiError geworfen.
 */
export async function request(method, path, body) {
  const options = { method, headers: {} };
  if (body instanceof FormData) {
    // Mehrere Dateien: Der Browser setzt Content-Type samt Trennzeichen.
    options.body = body;
  } else if (body instanceof Blob) {
    // Dateien (z. B. ein Bild) gehen unverändert als Anfragekörper raus.
    options.headers['Content-Type'] = body.type || 'application/octet-stream';
    options.body = body;
  } else if (body !== undefined) {
    options.headers['Content-Type'] = 'application/json';
    options.body = JSON.stringify(body);
  }

  let response;
  try {
    response = await fetch('api/v1' + path, options);
  } catch {
    throw new ApiError(0, 'Server nicht erreichbar');
  }

  if (response.status === 204) return null;

  const data = await response.json().catch(() => null);
  if (!response.ok) {
    throw new ApiError(response.status, data?.error ?? `Fehler ${response.status}`, data?.field);
  }
  return data;
}

/** Adresse einer Bilddatei, z. B. aus recipe.thumbnail. */
export function imageUrl(name) {
  return `api/v1/images/${name}`;
}

// Gleiche Grenze wie auf dem Server (images.MaxUploadBytes).
export const MAX_IMAGE_BYTES = 20 * 1024 * 1024;

/**
 * Baut die Parameter für die Rezeptliste. Leere Werte werden weggelassen.
 * filter: { query, tagIds, favoritesOnly, sort }
 */
function listParams(filter) {
  const params = new URLSearchParams();
  if (filter.query?.trim()) params.set('q', filter.query.trim());
  for (const id of filter.tagIds ?? []) params.append('tag', id);
  if (filter.favoritesOnly) params.set('favorite', 'true');
  if (filter.sort) params.set('sort', filter.sort);
  const text = params.toString();
  return text ? '?' + text : '';
}

export const api = {
  health: () => request('GET', '/health'),

  /**
   * Einstellungen: { family_size, shopping_day, bring_entity, app_url }.
   * updateSettings ändert nur die übergebenen Felder.
   */
  getSettings: () => request('GET', '/settings'),
  updateSettings: (settings) => request('PUT', '/settings', settings),

  /**
   * Essensplan von from bis to (beide "2026-10-21", eingeschlossen).
   * Ergebnis: { entries, today, shopping_period: { from, to } }.
   */
  mealPlan: (from, to) => request('GET', `/meal-plan?from=${from}&to=${to}`),

  getEntry: (id) => request('GET', `/meal-plan/${id}`),

  /** Neuer Eintrag: { date, slot?, recipe_id?, servings?, note? }. */
  createEntry: (entry) => request('POST', '/meal-plan', entry),

  /** Ändert { recipe_id, servings, note, status } eines Eintrags. */
  updateEntry: (id, changes) => request('PUT', `/meal-plan/${id}`, changes),

  deleteEntry: (id) => request('DELETE', `/meal-plan/${id}`),

  /**
   * Verschiebt einen Eintrag: { date, slot, swap_with?, confirm?,
   * have_ingredients? }. Ergebnis { applied, warnings, entries }; bei
   * Warnungen ohne confirm ist nichts gespeichert.
   */
  moveEntry: (id, move) => request('POST', `/meal-plan/${id}/move`, move),

  /** Vorratsliste: [{ id, name, synonyms: [...] }], alphabetisch. */
  listPantry: () => request('GET', '/pantry'),
  createPantryItem: (item) => request('POST', '/pantry', item),
  updatePantryItem: (id, item) => request('PUT', `/pantry/${id}`, item),
  deletePantryItem: (id) => request('DELETE', `/pantry/${id}`),

  /** API-Schlüssel des App-Zugangs: [{ id, name, created_at, last_used_at }]. */
  listApiKeys: () => request('GET', '/api-keys'),
  /** Legt einen Schlüssel an; nur diese Antwort enthält ihn (Feld key). */
  createApiKey: (name) => request('POST', '/api-keys', { name }),
  deleteApiKey: (id) => request('DELETE', `/api-keys/${id}`),

  /** Prüft Zutatennamen gegen den Vorrat; Ergebnis [true, false, …]. */
  matchPantry: async (names) => (await request('POST', '/pantry/match', { names })).in_pantry,

  /**
   * Gleicht alle Rezepte mit der Vorratsliste ab. Ohne confirm nur
   * Vorschau. Ergebnis { applied, changes: [{ recipe_title,
   * ingredient_name, shopping_relevant, … }], recipes }.
   */
  applyPantry: (confirm) => request('POST', '/pantry/apply', { confirm }),

  /**
   * Einkaufsliste des aktuellen (next = false) bzw. folgenden
   * Einkaufszeitraums. Ergebnis { period, today, entries: [{ id, date,
   * title, servings, status, included, … }], items: [{ key, name, spec,
   * recipes, … }] }.
   */
  shoppingPreview: (next = false) => request('GET', `/shopping/preview?period=${next ? 'next' : 'current'}`),

  /**
   * Sendet die Einkaufsliste an die gewählte Bring-Liste; deselected sind
   * die keys abgewählter Artikel. options: { missing: true } sendet nur die
   * Differenz, { entries: [id] } nur diese Gerichte. Ergebnis { list,
   * sent: [{ name, spec, merged }], failed: [{ name, spec, error }], entries }.
   */
  sendShopping: (next, deselected, options = {}) =>
    request('POST', '/shopping/send', { period: next ? 'next' : 'current', deselected, ...options }),

  /**
   * Verbindung zu Home Assistant: { mode: "supervisor" | "direct" | "test",
   * lists: [{ entity_id, name }], error? }.
   */
  homeAssistantStatus: () => request('GET', '/homeassistant/status'),

  /** Links zur Installation der Android-App: { apk_url, obtainium_url, … }. */
  appInstall: () => request('GET', '/app-install'),
  /** Prüft, ob unter url der App-Zugang antwortet: { ok, message }. */
  checkAppUrl: (url) => request('POST', '/app-install/check', { url }),

  /** Version, Arbeitsspeicher und Platzbedarf der Daten. */
  system: () => request('GET', '/system'),

  /** Liste von Kurzfassungen (id, title, Zeiten, thumbnail, is_favorite). */
  listRecipes: async (filter = {}) =>
    (await request('GET', '/recipes' + listParams(filter))).recipes,

  /** Vollständiges Rezept mit Zutaten, Schritten und Kategorien. */
  getRecipe: (id) => request('GET', `/recipes/${id}`),

  /** Legt ein Rezept an und liefert es mit ID zurück. */
  createRecipe: (recipe) => request('POST', '/recipes', recipe),

  /** Ersetzt ein Rezept vollständig und liefert den neuen Stand. */
  updateRecipe: (id, recipe) => request('PUT', `/recipes/${id}`, recipe),

  setFavorite: (id, isFavorite) =>
    request('PUT', `/recipes/${id}/favorite`, { is_favorite: isFavorite }),

  deleteRecipe: (id) => request('DELETE', `/recipes/${id}`),
  /** Legt eine Kopie als Variante an (der Server kopiert alles inkl. Bild). */
  createVariant: (id) => request('POST', `/recipes/${id}/variants`),

  /** Lädt file als Rezeptbild hoch; der Server verkleinert es. */
  setImage: (id, file) => request('PUT', `/recipes/${id}/image`, file),

  deleteImage: (id) => request('DELETE', `/recipes/${id}/image`),

  /** Lässt den Server das Bild unter url laden und als Rezeptbild speichern. */
  fetchImage: (id, url) => request('POST', `/recipes/${id}/image/fetch`, { url }),

  /**
   * Liest ein Rezept von einer Webseite. Ergebnis: { recipe, image_url,
   * duplicates } – recipe ist ein noch nicht gespeicherter Entwurf,
   * duplicates sind vorhandene Rezepte mit derselben Quell-URL.
   */
  importUrl: (url) => request('POST', '/import/url', { url }),

  /**
   * Lädt Importdateien (ZIP/JSON) hoch. Ergebnis ist nur eine Vorschau:
   * { token, expires_at, recipes: [{ index, file, recipe, thumbnail,
   * errors, warnings, duplicates }], file_errors: [{ file, error }] }.
   */
  importFiles: (files) => {
    const form = new FormData();
    for (const file of files) form.append('file', file, file.name);
    return request('POST', '/import/file', form);
  },
  /** Legt die ausgewählten Rezepte der Vorschau an: { created, failed }. */
  confirmImport: (token, selected) => request('POST', `/import/file/${token}`, { selected }),
  /** Vorschaubild eines Rezepts der Vorschau. */
  importThumbnailUrl: (token, index) => `api/v1/import/file/${token}/images/${index}`,

  /** Kategorien nach Gruppen: [{ name, tags: [{ id, name, ... }] }]. */
  listTagGroups: async () => (await request('GET', '/tags')).groups,
};
