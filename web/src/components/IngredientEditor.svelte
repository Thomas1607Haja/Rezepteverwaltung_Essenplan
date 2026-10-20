<script>
  import { newIngredient, newHeading } from '../lib/recipeForm.js';

  // items: Kärtchen (Zutaten und Überschriften), errors: Fehler je Schlüssel.
  // $bindable erlaubt der Seite, mit bind:items die Liste zu teilen, die
  // diese Komponente verändert.
  let { items = $bindable(), errors = {} } = $props();

  function add(item) {
    items.push(item);
  }

  function remove(index) {
    items.splice(index, 1);
  }
</script>

<div class="list">
  {#each items as item, i (item.key)}
    {#if item.kind === 'heading'}
      <div class="card heading">
        <input
          class="heading-input"
          placeholder="Überschrift, z. B. Für die Soße"
          aria-label="Gruppenüberschrift"
          bind:value={item.name}
        />
        <button type="button" class="remove" aria-label="Überschrift entfernen" onclick={() => remove(i)}>×</button>
      </div>
    {:else}
      <div class="card ingredient" class:invalid={errors[item.key]}>
        <input class="quantity" placeholder="Menge" aria-label="Menge" inputmode="decimal" bind:value={item.quantity} />
        <input class="unit" placeholder="Einheit" aria-label="Einheit" bind:value={item.unit} />
        <input class="name" placeholder="Zutat" aria-label="Zutat" bind:value={item.name} />
        <input class="note" placeholder="Hinweis, z. B. gehackt" aria-label="Hinweis" bind:value={item.note} />
        <label class="shopping" title="Beim Einkaufen berücksichtigen">
          <input type="checkbox" bind:checked={item.shopping_relevant} />
          einkaufen
        </label>
        <button type="button" class="remove" aria-label="Zutat entfernen" onclick={() => remove(i)}>×</button>
        {#if errors[item.key]}
          <p class="error">{errors[item.key]}</p>
        {/if}
      </div>
    {/if}
  {/each}
</div>

<div class="add">
  <button type="button" onclick={() => add(newIngredient())}>+ Zutat</button>
  <button type="button" onclick={() => add(newHeading())}>+ Überschrift</button>
</div>

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
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

  /* Handy: zwei Zeilen – Menge, Einheit, Zutat / Hinweis, einkaufen, × */
  .ingredient {
    grid-template-columns: 4.5rem 5rem 1fr;
    grid-template-areas:
      'quantity unit name'
      'note note note'
      'shopping shopping remove';
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
    grid-column: 1 / -1;
    margin: 0;
    font-size: 0.85rem;
    color: var(--danger);
  }

  @media (min-width: 45rem) {
    .ingredient {
      grid-template-columns: 4.5rem 5.5rem 1fr 1fr auto auto;
      grid-template-areas: 'quantity unit name note shopping remove';
      align-items: center;
    }
  }

  .heading {
    grid-template-columns: 1fr auto;
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
