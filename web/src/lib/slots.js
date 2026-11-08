// Mahlzeiten eines Tages (Schritt 9.3): Mittag, Kaffee & Kuchen, Abend.
// Der Kaffee-Slot lässt sich in den Einstellungen abschalten. Gespeichert
// wird nur der Schlüssel (value); die Texte kommen aus locales/slots.
import { t } from './i18n.svelte.js';

/** Ein Slot mit Texten in der Sprache der Oberfläche (Getter: reagieren auf Sprachwechsel). */
function slot(value) {
  return {
    value,
    get label() {
      return t(`slots.${value}`);
    },
    get short() {
      return t(`slots.${value}_short`);
    },
    get when() {
      return t(`slots.${value}_when`);
    },
  };
}

export const SLOTS = [slot('dinner'), slot('lunch'), slot('coffee')];

/** Slots zur Auswahl; coffee nur, wenn eingeschaltet (oder schon gewählt). */
export function slotChoices(coffeeSlot, current = '') {
  return SLOTS.filter((s) => s.value !== 'coffee' || coffeeSlot || current === 'coffee');
}

/** Anzeigename eines Slots, z. B. „Kaffee & Kuchen“. */
export const slotLabel = (slot) => (SLOTS.some((s) => s.value === slot) ? t(`slots.${slot}`) : t('slots.dinner'));
