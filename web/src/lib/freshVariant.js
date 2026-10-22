// Merkt sich die gerade angelegte Variante, bis ihre Bearbeiten-Seite sie
// übernimmt. Bricht man dort sofort ab, fragt die Seite, ob die Variante
// wieder verworfen werden soll, und kehrt zum Ausgangsrezept zurück.

const fresh = { id: null, sourceId: null };

/** Variante id wurde eben aus dem Rezept sourceId angelegt. */
export function markFreshVariant(id, sourceId) {
  fresh.id = id;
  fresh.sourceId = sourceId;
}

/**
 * Liefert die ID des Ausgangsrezepts, wenn id die gerade angelegte
 * Variante ist (sonst null), und vergisst den Merker.
 */
export function takeFreshVariant(id) {
  const sourceId = fresh.id === id ? fresh.sourceId : null;
  fresh.id = fresh.sourceId = null;
  return sourceId;
}
