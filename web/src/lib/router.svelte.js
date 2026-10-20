// Kleiner Hash-Router: Die aktuelle Seite steht hinter dem # in der Adresse,
// z. B. .../#/recipes/12. Der Teil vor dem # bleibt immer gleich, deshalb
// funktioniert das auch unter dem wechselnden Ingress-Pfad von Home
// Assistant, und der Server muss nur die index.html ausliefern.
//
// Die Endung .svelte.js erlaubt hier Svelte-Runen wie $state außerhalb
// einer Komponente.

/** Liest den Pfad aus dem Hash; ohne Hash gilt "/". */
function readPath() {
  const path = window.location.hash.slice(1); // "#/recipes" -> "/recipes"
  return path.startsWith('/') ? path : '/';
}

// $state macht den Wert reaktiv: Jede Komponente, die ihn liest, wird bei
// einer Änderung automatisch neu dargestellt.
const route = $state({ path: readPath() });

window.addEventListener('hashchange', () => {
  route.path = readPath();
});

/** Aktueller Pfad, z. B. "/recipes/12". */
export function currentPath() {
  return route.path;
}

/** Wechselt zur Seite path, z. B. navigate('/recipes'). */
export function navigate(path) {
  window.location.hash = path;
}

/**
 * Prüft, ob path zum Muster pattern passt. Platzhalter beginnen mit ":".
 * Beispiel: match('/recipes/:id', '/recipes/12') -> { id: '12' }.
 * Passt es nicht, ist das Ergebnis null.
 */
export function match(pattern, path) {
  const want = pattern.split('/');
  const got = path.split('/');
  if (want.length !== got.length) return null;

  const params = {};
  for (let i = 0; i < want.length; i++) {
    if (want[i].startsWith(':')) {
      params[want[i].slice(1)] = decodeURIComponent(got[i]);
    } else if (want[i] !== got[i]) {
      return null;
    }
  }
  return params;
}
