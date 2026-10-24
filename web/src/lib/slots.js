// Mahlzeiten eines Tages (Schritt 9.3): Mittag, Kaffee & Kuchen, Abend.
// Der Kaffee-Slot lässt sich in den Einstellungen abschalten.

export const SLOTS = [
  { value: 'dinner', label: 'Abendessen', short: 'Abend', when: 'abends' },
  { value: 'lunch', label: 'Mittagessen', short: 'Mittag', when: 'mittags' },
  { value: 'coffee', label: 'Kaffee & Kuchen', short: 'Kaffee', when: 'nachmittags' },
];

/** Slots zur Auswahl; coffee nur, wenn eingeschaltet (oder schon gewählt). */
export function slotChoices(coffeeSlot, current = '') {
  return SLOTS.filter((s) => s.value !== 'coffee' || coffeeSlot || current === 'coffee');
}

/** Anzeigename eines Slots, z. B. „Kaffee & Kuchen“. */
export const slotLabel = (slot) => SLOTS.find((s) => s.value === slot)?.label ?? 'Abendessen';
