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
  if (body instanceof Blob) {
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

  /** Kategorien nach Gruppen: [{ name, tags: [{ id, name, ... }] }]. */
  listTagGroups: async () => (await request('GET', '/tags')).groups,
};
