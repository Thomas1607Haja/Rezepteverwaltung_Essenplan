// Gemeinsames für die Kurzvorschau beim Einplanen (components/PlanPreview.svelte).

import { api } from './api.js';

/** Ist die Kurzvorschau eingeschaltet (Einstellung plan_preview)? Fehler = ja. */
export async function previewEnabled() {
  try {
    return (await api.getSettings()).plan_preview !== false;
  } catch {
    return true;
  }
}

/**
 * Plant ein Rezept nach der Kurzvorschau ein (bzw. ohne Vorschau mit
 * choice = null). „Alles vorhanden“ setzt danach den Status – das Anlegen
 * selbst kennt nur geplante Einträge.
 */
export async function createWithChoice(date, slot, recipeId, choice) {
  const entry = await api.createEntry({
    date,
    slot,
    recipe_id: recipeId,
    ...(choice ? { servings: choice.servings, excluded_ingredients: choice.excluded } : {}),
  });
  if (choice?.allAvailable) {
    return api.updateEntry(entry.id, {
      recipe_id: recipeId,
      servings: entry.servings,
      note: entry.note,
      status: 'have_ingredients',
      excluded_ingredients: [],
    });
  }
  return entry;
}

/**
 * Setzt ein (anderes) Rezept in einen bestehenden Eintrag (Ersetzen). Ohne
 * Vorschau verfällt die alte Abwahl auf dem Server von selbst.
 */
export function replaceWithChoice(entry, recipeId, choice) {
  // „Zutaten vorhanden“ galt für das alte Gericht.
  let status = entry.status === 'have_ingredients' ? 'planned' : entry.status;
  if (choice?.allAvailable && status !== 'transferred') status = 'have_ingredients';
  return api.updateEntry(entry.id, {
    recipe_id: recipeId,
    servings: choice ? choice.servings : entry.servings,
    // Ohne bisheriges Rezept war die Notiz der Titel – sie entfällt dann.
    note: entry.recipe_id ? entry.note : '',
    status,
    ...(choice ? { excluded_ingredients: choice.excluded } : {}),
  });
}
