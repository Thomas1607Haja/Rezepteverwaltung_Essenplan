// Gemeinsame Einstellungen für Drag and Drop (Bibliothek svelte-dnd-action).

import { setAriaStrings, SHADOW_ITEM_MARKER_PROPERTY_NAME } from 'svelte-dnd-action';
import { currentLanguage, t } from './i18n.svelte.js';

// Dauer der Animation, mit der die anderen Kärtchen Platz machen (ms).
export const flipDurationMs = 150;

// Rahmen um die Liste, solange gezogen wird.
export const dropTargetStyle = { outline: '2px dashed var(--accent)', outlineOffset: '4px' };

/** Ist item der Platzhalter, den die Bibliothek beim Ziehen einfügt? */
export function isShadow(item) {
  return Boolean(item[SHADOW_ITEM_MARKER_PROPERTY_NAME]);
}

// Ansagen für Screenreader in der Sprache der Oberfläche. Gilt für alle
// Listen der Seite. Die Texte werden erst beim Ansagen übersetzt; wechselt
// die Sprache, setzt der nächste Aufruf die festen Hinweise neu.
let ariaLanguage = '';

function syncAria() {
  if (ariaLanguage === currentLanguage()) return;
  ariaLanguage = currentLanguage();
  applyAriaStrings();
}

function applyAriaStrings() {
  setAriaStrings({
    dragStarted: ({ itemLabel, zoneLabel }) => (syncAria(), t('plan.dnd.drag_started', { item: itemLabel, zone: zoneLabel })),
    movedToPosition: ({ itemLabel, position }) => (syncAria(), t('plan.dnd.moved_to_position', { item: itemLabel, position })),
    movedToZoneEnd: ({ itemLabel }) => (syncAria(), t('plan.dnd.moved_to_end', { item: itemLabel })),
    movedToZoneStart: ({ itemLabel }) => (syncAria(), t('plan.dnd.moved_to_start', { item: itemLabel })),
    dropped: ({ itemLabel, position, count }) => (syncAria(), t('plan.dnd.dropped', { item: itemLabel, position, count })),
    zoneActiveInstruction: () => (syncAria(), t('plan.dnd.zone_active')),
    zoneDragDisabledInstruction: t('plan.dnd.zone_disabled'),
  });
}
applyAriaStrings();
