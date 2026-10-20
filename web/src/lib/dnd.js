// Gemeinsame Einstellungen für Drag and Drop (Bibliothek svelte-dnd-action).

import { setAriaStrings, SHADOW_ITEM_MARKER_PROPERTY_NAME } from 'svelte-dnd-action';

// Dauer der Animation, mit der die anderen Kärtchen Platz machen (ms).
export const flipDurationMs = 150;

// Rahmen um die Liste, solange gezogen wird.
export const dropTargetStyle = { outline: '2px dashed var(--accent)', outlineOffset: '4px' };

/** Ist item der Platzhalter, den die Bibliothek beim Ziehen einfügt? */
export function isShadow(item) {
  return Boolean(item[SHADOW_ITEM_MARKER_PROPERTY_NAME]);
}

// Ansagen für Screenreader auf Deutsch. Gilt für alle Listen der Seite.
setAriaStrings({
  dragStarted: ({ itemLabel, zoneLabel }) =>
    `${itemLabel} aufgenommen. Mit den Pfeiltasten in der Liste ${zoneLabel} verschieben.`,
  movedToPosition: ({ itemLabel, position }) => `${itemLabel} an Position ${position} verschoben.`,
  movedToZoneEnd: ({ itemLabel }) => `${itemLabel} ans Ende verschoben.`,
  movedToZoneStart: ({ itemLabel }) => `${itemLabel} an den Anfang verschoben.`,
  dropped: ({ itemLabel, position, count }) => `${itemLabel} abgelegt, Position ${position} von ${count}.`,
  zoneActiveInstruction: 'Mit Tab zu einem Griff wechseln und mit Leertaste aufnehmen.',
  zoneDragDisabledInstruction: 'Verschieben ist hier nicht möglich.',
});
