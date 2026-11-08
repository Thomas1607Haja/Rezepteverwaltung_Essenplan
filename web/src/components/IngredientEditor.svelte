<script>
  import { flip } from 'svelte/animate';
  import { dragHandleZone } from 'svelte-dnd-action';
  import { api } from '../lib/api.js';
  import { newIngredient, newHeading, validationText } from '../lib/recipeForm.js';
  import { t } from '../lib/i18n.svelte.js';
  import { flipDurationMs, dropTargetStyle, isShadow } from '../lib/dnd.js';
  import DragHandle from './DragHandle.svelte';

  // items: Kärtchen (Zutaten und Überschriften), errors: Fehler je Zeilen-ID.
  // $bindable erlaubt der Seite, mit bind:items die Liste zu teilen, die
  // diese Komponente verändert.
  let { items = $bindable(), errors = {} } = $props();

  // Die Bibliothek meldet beim Ziehen laufend die neue Reihenfolge
  // (consider) und am Ende die endgültige (finalize). Beide Male wird die
  // Liste einfach übernommen.
  function sort(event) {
    items = event.detail.items;
  }

  function remove(index) {
    items.splice(index, 1);
  }

  /**
   * Nach dem Ändern eines Zutatennamens: Steht die Zutat im Vorrat, wird
   * sie nicht zum Einkaufen vorgemerkt, sonst schon. Danach lässt sich das
   * Häkchen von Hand umstellen. Schlägt die Prüfung fehl, bleibt es, wie es
   * ist – das Häkchen ist nur ein Vorschlag.
   */
  async function suggestShopping(item) {
    const name = item.name.trim();
    if (name === '') return;
    try {
      const [inPantry] = await api.matchPantry([name]);
      // Inzwischen weitergetippt? Dann gilt die neuere Prüfung.
      if (item.name.trim() === name) item.shopping_relevant = !inPantry;
    } catch {
      // bewusst ignoriert, siehe oben
    }
  }

  function label(item) {
    return item.kind === 'heading'
      ? t('editor.ingredients.heading_label', { name: item.name || t('editor.ingredients.unnamed') })
      : t('editor.ingredients.ingredient_label', { name: item.name || t('editor.ingredients.empty') });
  }
</script>

<!-- type trennt die Listen: Zutaten lassen sich nicht zu den Schritten ziehen. -->
<div
  class="list"
  aria-label={t('editor.sections.ingredients')}
  use:dragHandleZone={{ items, flipDurationMs, dropTargetStyle, type: 'ingredients' }}
  onconsider={sort}
  onfinalize={sort}
>
  {#each items as item, i (item.id)}
    <!-- animate:flip lässt die übrigen Kärtchen sanft Platz machen. -->
    <div class="slot" class:shadow={isShadow(item)} aria-label={label(item)} animate:flip={{ duration: flipDurationMs }}>
      {#if item.kind === 'heading'}
        <div class="card heading">
          <DragHandle label={t('editor.ingredients.move', { label: label(item) })} />
          <input
            class="heading-input"
            placeholder={t('editor.ingredients.heading_placeholder')}
            aria-label={t('editor.ingredients.heading_aria')}
            bind:value={item.name}
          />
          <button type="button" class="remove" aria-label={t('editor.ingredients.remove_heading')} onclick={() => remove(i)}>×</button>
        </div>
      {:else}
        <div class="card ingredient" class:invalid={errors[item.id]}>
          <span class="grip"><DragHandle label={t('editor.ingredients.move', { label: label(item) })} /></span>
          <input class="quantity" placeholder={t('editor.ingredients.quantity')} aria-label={t('editor.ingredients.quantity')} inputmode="decimal" bind:value={item.quantity} />
          <input class="unit" placeholder={t('editor.ingredients.unit')} aria-label={t('editor.ingredients.unit')} bind:value={item.unit} />
          <!-- change feuert beim Verlassen des Feldes, wenn sich der Text geändert hat. -->
          <input class="name" placeholder={t('editor.ingredients.name')} aria-label={t('editor.ingredients.name')} bind:value={item.name} onchange={() => suggestShopping(item)} />
          <input class="note" placeholder={t('editor.ingredients.note_placeholder')} aria-label={t('editor.ingredients.note')} bind:value={item.note} />
          <label class="shopping" title={t('editor.ingredients.shopping_title')}>
            <input type="checkbox" bind:checked={item.shopping_relevant} />
            {t('editor.ingredients.shopping')}
          </label>
          <button type="button" class="remove" aria-label={t('editor.ingredients.remove')} onclick={() => remove(i)}>×</button>
          {#if errors[item.id]}
            <p class="error">{validationText(errors[item.id])}</p>
          {/if}
        </div>
      {/if}
    </div>
  {/each}
</div>

<div class="add">
  <button type="button" onclick={() => items.push(newIngredient())}>+ {t('editor.ingredients.add')}</button>
  <button type="button" onclick={() => items.push(newHeading())}>+ {t('editor.ingredients.add_heading')}</button>
</div>

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    border-radius: var(--radius);
  }

  /* Platzhalter an der Stelle, an der das gezogene Kärtchen landen wird */
  .slot.shadow {
    opacity: 0.35;
  }

  .card {
    display: grid;
    gap: 0.4rem;
    padding: 0.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
  }

  .card.invalid {
    border-color: var(--danger);
  }

  /* Handy: Griff links über alle Zeilen; daneben Menge, Einheit, Zutat /
     Hinweis / einkaufen, × */
  .ingredient {
    grid-template-columns: auto 3.5rem 4rem 1fr;
    grid-template-areas:
      'grip quantity unit name'
      'grip note note note'
      'grip shopping shopping remove';
  }

  .grip {
    grid-area: grip;
    align-self: center;
  }
  .quantity {
    grid-area: quantity;
  }
  .unit {
    grid-area: unit;
  }
  .name {
    grid-area: name;
  }
  .note {
    grid-area: note;
  }
  .shopping {
    grid-area: shopping;
  }
  .ingredient .remove {
    grid-area: remove;
    justify-self: end;
  }

  .error {
    grid-column: 2 / -1;
    margin: 0;
    font-size: 0.85rem;
    color: var(--danger);
  }

  @media (min-width: 45rem) {
    .ingredient {
      grid-template-columns: auto 4.5rem 5.5rem 1fr 1fr auto auto;
      grid-template-areas: 'grip quantity unit name note shopping remove';
      align-items: center;
    }
  }

  .heading {
    grid-template-columns: auto 1fr auto;
    align-items: center;
    background: var(--accent-soft);
    border-color: transparent;
  }

  .heading-input {
    font-weight: 600;
  }

  input:not([type='checkbox']) {
    width: 100%;
    min-width: 0;
    padding: 0.4rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
  }

  input:focus {
    outline: 2px solid var(--accent);
    outline-offset: -1px;
  }

  .shopping {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    font-size: 0.85rem;
    color: var(--text-muted);
    white-space: nowrap;
  }

  .remove {
    width: 2rem;
    height: 2rem;
    padding: 0;
    border: none;
    border-radius: 999px;
    background: none;
    color: var(--text-muted);
    font-size: 1.3rem;
    line-height: 1;
    cursor: pointer;
  }

  .remove:hover {
    background: var(--surface-hover);
    color: var(--danger);
  }

  .add {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.75rem;
  }

  .add button {
    padding: 0.4rem 0.8rem;
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    background: none;
    color: var(--accent);
    cursor: pointer;
  }

  .add button:hover {
    background: var(--surface-hover);
  }
</style>
