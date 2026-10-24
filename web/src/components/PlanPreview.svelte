<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { formatQuantity } from '../lib/format.js';

  // Kurzvorschau beim Einplanen (Schritt 8.E): Portionen und Zutaten, die
  // man noch hat, abwählen. Gilt nur für diesen Planeintrag.
  //   recipeId     Rezept, das eingeplant wird
  //   servings     Portionen zu Beginn (null = Familiengröße)
  //   excluded     schon abgewählte Zutatenschlüssel (item_key)
  //   confirmLabel Text des Hauptknopfs
  //   onconfirm({ servings, excluded, allAvailable })
  //   onclose()
  let { recipeId, servings = null, excluded = [], confirmLabel = 'Hinzufügen', busy = false, onconfirm, onclose } =
    $props();

  let recipe = $state(null);
  let error = $state('');
  // svelte-ignore state_referenced_locally
  let portions = $state(servings);
  // svelte-ignore state_referenced_locally
  let off = $state(new Set(excluded));

  let dialog;
  onMount(() => dialog.showModal());

  // Einmal beim Öffnen laden; spätere Änderungen der Werte zählen nicht.
  // svelte-ignore state_referenced_locally
  Promise.all([api.getRecipe(recipeId), portions == null ? api.getSettings() : null])
    .then(([r, settings]) => {
      recipe = r;
      if (portions == null) portions = settings.family_size;
    })
    .catch((e) => (error = e.message));

  const toBuy = $derived(recipe?.ingredients.filter((i) => i.shopping_relevant) ?? []);
  const pantry = $derived(recipe?.ingredients.filter((i) => !i.shopping_relevant) ?? []);
  // Gezählt je Schlüssel: Zwei Zeilen „Zwiebel“ sind ein Artikel.
  const offCount = $derived(new Set(toBuy.map((i) => i.item_key).filter((k) => off.has(k))).size);

  function amount(ing) {
    if (ing.quantity == null) return ing.unit;
    return `${formatQuantity((ing.quantity * portions) / recipe.base_servings)} ${ing.unit}`.trim();
  }

  function toggle(key) {
    const next = new Set(off);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    off = next;
  }

  function confirm(allAvailable) {
    // Nur Schlüssel, die es im Rezept noch gibt.
    const keys = [...off].filter((k) => toBuy.some((i) => i.item_key === k));
    onconfirm({ servings: Number(portions), excluded: allAvailable ? [] : keys, allAvailable });
  }
</script>

<dialog
  bind:this={dialog}
  oncancel={onclose}
  onclick={(e) => e.target === dialog && onclose()}
  aria-labelledby="preview-title"
>
  <div class="content">
    {#if error}
      <p class="error">{error}</p>
      <footer><button type="button" onclick={onclose}>Schließen</button></footer>
    {:else if !recipe}
      <p>Lädt …</p>
    {:else}
      <h2 id="preview-title">{recipe.title}</h2>
      <div class="portions">
        <button type="button" aria-label="Weniger Portionen" disabled={portions <= 1} onclick={() => portions--}>−</button>
        <span>{portions} {portions === 1 ? 'Portion' : 'Portionen'}</span>
        <button type="button" aria-label="Mehr Portionen" disabled={portions >= 50} onclick={() => portions++}>+</button>
      </div>

      {#if toBuy.length > 0}
        <p class="hint">Häkchen entfernen, was du noch hast – gilt nur für diesen Eintrag.</p>
        <ul class="ingredients">
          {#each toBuy as ing (ing.id)}
            <li>
              <label class:off={off.has(ing.item_key)}>
                <input type="checkbox" checked={!off.has(ing.item_key)} onchange={() => toggle(ing.item_key)} />
                <span class="amount">{amount(ing)}</span>
                <span>{ing.name}{ing.note ? `, ${ing.note}` : ''}</span>
              </label>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="hint">Keine Zutaten zum Einkaufen.</p>
      {/if}

      {#if pantry.length > 0}
        <details>
          <summary>Aus dem Vorrat ({pantry.length}) – wird nicht eingekauft</summary>
          <ul class="ingredients pantry">
            {#each pantry as ing (ing.id)}
              <li><span class="amount">{amount(ing)}</span> {ing.name}</li>
            {/each}
          </ul>
        </details>
      {/if}

      <footer>
        <button type="button" onclick={onclose} disabled={busy}>Abbrechen</button>
        <button type="button" onclick={() => confirm(true)} disabled={busy} title="Alle Zutaten sind im Haus">
          Alles vorhanden
        </button>
        <button type="button" class="primary" onclick={() => confirm(false)} disabled={busy}>
          {busy ? 'Speichert …' : confirmLabel}{offCount > 0 ? ` (${offCount} vorhanden)` : ''}
        </button>
      </footer>
    {/if}
  </div>
</dialog>

<style>
  dialog {
    width: min(30rem, calc(100vw - 2rem));
    max-height: calc(100dvh - 2rem);
    padding: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
  }

  dialog::backdrop {
    background: rgb(0 0 0 / 0.4);
  }

  .content {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 1rem;
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  p {
    margin: 0;
  }

  .hint {
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  .error {
    color: var(--danger);
  }

  .portions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .portions button {
    width: 2.2rem;
    height: 2.2rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    font-size: 1.1rem;
    cursor: pointer;
  }

  .ingredients {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .ingredients li {
    border-bottom: 1px solid var(--border);
  }

  .ingredients label {
    display: grid;
    grid-template-columns: 1.5rem 5.5rem 1fr;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0;
    cursor: pointer;
  }

  input[type='checkbox'] {
    width: 1.1rem;
    height: 1.1rem;
    accent-color: var(--accent);
  }

  .ingredients label.off span {
    color: var(--text-muted);
    text-decoration: line-through;
  }

  .amount {
    font-weight: 600;
    text-align: right;
  }

  .pantry li {
    display: grid;
    grid-template-columns: 5.5rem 1fr;
    gap: 0.4rem;
    padding: 0.25rem 0 0.25rem 1.9rem;
    color: var(--text-muted);
  }

  details summary {
    color: var(--text-muted);
    font-size: 0.9rem;
    cursor: pointer;
  }

  footer {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  footer button {
    padding: 0.5rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }

  footer .primary {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }
</style>
