// Umwandlung zwischen Rezept (wie es die API liefert) und Formular.
//
// Im Formular sind Zutaten eine Liste von Kärtchen: Zutaten und
// Gruppenüberschriften. Die API kennt keine Überschriften, sondern
// group_name an jeder Zutat. Beim Laden werden daraus Überschriften, beim
// Speichern bekommt jede Zutat den Namen der Überschrift darüber.

let nextKey = 1;

/** Eindeutiger Schlüssel für {#each}, damit Svelte Zeilen auseinanderhält. */
function key() {
  return nextKey++;
}

export function newIngredient() {
  return { key: key(), kind: 'ingredient', quantity: '', unit: '', name: '', note: '', shopping_relevant: true };
}

export function newHeading() {
  return { key: key(), kind: 'heading', name: '' };
}

export function newStep() {
  return { key: key(), text: '' };
}

/** Formularzustand für ein neues bzw. ein vorhandenes Rezept. */
export function toForm(recipe) {
  if (!recipe) {
    return {
      title: '',
      description: '',
      base_servings: 4,
      prep_minutes: '',
      total_minutes: '',
      source_url: '',
      notes: '',
      is_favorite: false,
      tagIds: [],
      items: [newIngredient()],
      steps: [newStep()],
    };
  }

  const items = [];
  let group = '';
  for (const ing of recipe.ingredients) {
    if (ing.group_name !== group) {
      group = ing.group_name;
      // Wechsel zurück zu "ohne Gruppe" ist mitten in der Liste nicht
      // darstellbar; dann bleibt es bei der vorigen Überschrift.
      if (group !== '') items.push({ ...newHeading(), name: group });
    }
    items.push({
      ...newIngredient(),
      quantity: ing.quantity == null ? '' : formatInputNumber(ing.quantity),
      unit: ing.unit,
      name: ing.name,
      note: ing.note,
      shopping_relevant: ing.shopping_relevant,
    });
  }

  return {
    title: recipe.title,
    description: recipe.description,
    base_servings: recipe.base_servings,
    prep_minutes: recipe.prep_minutes ?? '',
    total_minutes: recipe.total_minutes ?? '',
    source_url: recipe.source_url,
    notes: recipe.notes,
    is_favorite: recipe.is_favorite,
    tagIds: recipe.tags.map((t) => t.id),
    items: items.length > 0 ? items : [newIngredient()],
    steps: recipe.steps.length > 0 ? recipe.steps.map((s) => ({ ...newStep(), text: s.text })) : [newStep()],
  };
}

/** Zahl für ein Eingabefeld: 0.5 -> "0,5". */
function formatInputNumber(value) {
  return String(value).replace('.', ',');
}

const unicodeFractions = { '¼': 0.25, '½': 0.5, '¾': 0.75, '⅓': 1 / 3, '⅔': 2 / 3 };

/**
 * Liest eine Mengenangabe: "1,5", "1.5", "1/2", "1 1/2", "½", "1½".
 * Leer -> null. Nicht lesbar -> undefined.
 */
export function parseQuantity(text) {
  let s = String(text).trim();
  if (s === '') return null;

  let value = 0;
  // Unicode-Bruch am Ende, z. B. "1½" oder "½"
  const last = s.at(-1);
  if (last in unicodeFractions) {
    value += unicodeFractions[last];
    s = s.slice(0, -1).trim();
    if (s === '') return value;
  }

  // "1 1/2" oder "1/2"
  const fraction = s.match(/^(?:(\d+)\s+)?(\d+)\s*\/\s*(\d+)$/);
  if (fraction) {
    const [, whole, num, den] = fraction;
    if (Number(den) === 0) return undefined;
    return value + Number(whole ?? 0) + Number(num) / Number(den);
  }

  // Dezimalzahl mit Komma oder Punkt
  if (!/^\d+([.,]\d+)?$/.test(s)) return undefined;
  return value + Number(s.replace(',', '.'));
}

/** Liest eine Minutenangabe; leer -> null. */
function parseMinutes(value) {
  if (value === '' || value == null) return null;
  return Number(value);
}

/**
 * Prüft das Formular und baut daraus die Daten für die API.
 * Ergebnis: { payload } oder { errors } mit Fehlertexten je Feld bzw. je
 * Zeilenschlüssel.
 */
export function toPayload(form) {
  const errors = {};

  if (form.title.trim() === '') errors.title = 'Bitte einen Titel eingeben.';
  if (!Number.isInteger(Number(form.base_servings)) || Number(form.base_servings) < 1) {
    errors.base_servings = 'Mindestens 1 Portion.';
  }
  for (const field of ['prep_minutes', 'total_minutes']) {
    const v = form[field];
    if (v !== '' && v != null && (!Number.isInteger(Number(v)) || Number(v) < 0)) {
      errors[field] = 'Ganze Minuten, nicht negativ.';
    }
  }

  const ingredients = [];
  let group = '';
  for (const item of form.items) {
    if (item.kind === 'heading') {
      group = item.name.trim();
      continue;
    }
    const isEmpty = [item.quantity, item.unit, item.name, item.note].every((v) => String(v).trim() === '');
    if (isEmpty) continue; // leere Zeilen einfach weglassen

    const quantity = parseQuantity(item.quantity);
    if (quantity === undefined) {
      errors[item.key] = 'Menge nicht lesbar, z. B. 1,5 oder 1/2.';
    } else if (item.name.trim() === '') {
      errors[item.key] = 'Bitte einen Namen eingeben.';
    }
    ingredients.push({
      group_name: group,
      quantity: quantity ?? null,
      unit: item.unit.trim(),
      name: item.name.trim(),
      note: item.note.trim(),
      shopping_relevant: item.shopping_relevant,
    });
  }

  const steps = form.steps.map((s) => s.text.trim()).filter((t) => t !== '').map((text) => ({ text }));

  if (Object.keys(errors).length > 0) return { errors };

  return {
    payload: {
      title: form.title.trim(),
      description: form.description.trim(),
      base_servings: Number(form.base_servings),
      prep_minutes: parseMinutes(form.prep_minutes),
      total_minutes: parseMinutes(form.total_minutes),
      source_url: form.source_url.trim(),
      is_favorite: form.is_favorite,
      notes: form.notes.trim(),
      ingredients,
      steps,
      tags: form.tagIds.map((id) => ({ id })),
    },
  };
}
