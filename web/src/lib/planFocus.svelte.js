// Merkt sich, welchen Tag der Essensplan beim nächsten Öffnen zeigen soll –
// z. B. nach dem Einplanen eines Rezepts den Tag, an dem es nun steht.
// Ohne Angabe zeigt der Plan heute.

export const planFocus = $state({ date: null });

/** Plan beim nächsten Öffnen bei date ("2026-10-24") beginnen lassen. */
export function focusPlanOn(date) {
  planFocus.date = date;
}

/** Liefert den gemerkten Tag einmalig und vergisst ihn dann. */
export function takePlanFocus() {
  const date = planFocus.date;
  planFocus.date = null;
  return date;
}
