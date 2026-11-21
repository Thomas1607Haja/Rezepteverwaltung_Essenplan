// Planungszeitraum (Einstellung plan_weeks): bis zu welchem Tag sich der
// Essensplan im Voraus planen lässt. Schon weiter geplante Einträge bleiben
// erhalten; die Grenze gilt nur für Auswahl und Scrollen.

import { api } from './api.js';
import { addDays, todayString } from './format.js';

/** Größte erlaubte Wochenzahl (wie settings.MaxPlanWeeks im Server). */
export const MAX_PLAN_WEEKS = 12;

/** Letzter planbarer Tag ab today: heute + Wochen × 7 − 1. */
export function planEnd(today, weeks) {
  return addDays(today, (weeks || 4) * 7 - 1);
}

/** Letzter planbarer Tag laut Einstellungen; Fehler = Standard 4 Wochen. */
export async function loadPlanEnd() {
  let weeks = 4;
  try {
    weeks = (await api.getSettings()).plan_weeks;
  } catch {
    // Standard
  }
  return planEnd(todayString(), weeks);
}
