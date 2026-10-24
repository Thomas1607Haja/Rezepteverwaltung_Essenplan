<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { amountText, formatQuantity } from '../lib/format.js';

  // Kurzvorschau für ein ganzes Menü (Schritt 9.4): ein Abschnitt je Gang
  // mit Rezept, Häkchen wie in PlanPreview. Gespeichert wird je Gang
  // (abgewählte Zutaten gehören zum Planeintrag).
  //   entries     Gänge des Menüs
  //   courseName  (id) => Name des Gangs
  //   onsaved()   nach dem Speichern
  //   onclose()
  let { entries, courseName, onsaved, onclose } = $props();

  let dialog;
  onMount(() => dialog.showModal());

  let recipes = $state({}); // recipe_id → Rezept
  let off = $state({}); // entry id → Set abgewählter Schlüssel
  let error = $state('');
  let busy = $state(false);

  // svelte-ignore state_referenced_locally
  const courses = entries.filter((e) => e.recipe_id);
  for (const e of courses) off[e.id] = new Set(e.excluded_ingredients ?? []);
  Promise.all(courses.map((e) => api.getRecipe(e.recipe_id)))
    .then((list) => {
      for (const r of list) recipes[r.id] = r;
    })
    .catch((e) => (error = e.message));

  const loaded = $derived(courses.every((e) => recipes[e.recipe_id]));

  function amount(ing, e) {
    const r = recipes[e.recipe_id];
    if (ing.quantity == null) return ing.unit;
    return `${formatQuantity((ing.quantity * e.servings) / r.base_servings)} ${ing.unit}`.trim();
  }

  function toggle(entryId, key) {
    const next = new Set(off[entryId]);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    off[entryId] = next;
  }

  async function save(allAvailable) {
    busy = true;
    error = '';
    try {
      for (const e of courses) {
        const keys = [...off[e.id]].filter((k) => recipes[e.recipe_id].ingredients.some((i) => i.item_key === k));
        let status = e.status;
        if (status !== 'transferred') status = allAvailable ? 'have_ingredients' : 'planned';
        await api.updateEntry(e.id, {
          recipe_id: e.recipe_id,
          servings: e.servings,
          note: e.note,
          status,
          excluded_ingredients: allAvailable ? [] : keys.sort(),
        });
      }
      onsaved();
    } catch (err) {
      error = err.message;
      busy = false;
    }
  }
</script>

<dialog bind:this={dialog} oncancel={onclose} onclick={(e) => e.target === dialog && onclose()} aria-labelledby="menu-preview-title">
  <div class="content">
    <h2 id="menu-preview-title">Zutaten vorhanden – ganzes Menü</h2>
    <p class="hint">Häkchen entfernen, was du noch hast. Gilt nur für dieses Menü.</p>
    {#if error}<p class="error">{error}</p>{/if}
    {#if !loaded}
      <p>Lädt …</p>
    {:else}
      {#each courses as e (e.id)}
        {@const r = recipes[e.recipe_id]}
        <section>
          <h3>
            {#if e.course_tag_id && courseName(e.course_tag_id)}<small>{courseName(e.course_tag_id)}</small>{/if}
            {r.title} <span class="servings">{amountText(e.servings, r.yield_unit)}</span>
          </h3>
          {#if e.status === 'transferred'}<p class="hint">Schon an Bring gesendet – Änderungen erscheinen im Einkauf als Differenz.</p>{/if}
          <ul>
            {#each r.ingredients.filter((i) => i.shopping_relevant) as ing (ing.id)}
              <li>
                <label class:off={off[e.id].has(ing.item_key)}>
                  <input type="checkbox" checked={!off[e.id].has(ing.item_key)} onchange={() => toggle(e.id, ing.item_key)} />
                  <span class="amount">{amount(ing, e)}</span>
                  <span>{ing.name}</span>
                </label>
              </li>
            {:else}
              <li class="hint">Keine Zutaten zum Einkaufen.</li>
            {/each}
          </ul>
        </section>
      {:else}
        <p class="hint">Das Menü hat noch keine Gänge mit Rezept.</p>
      {/each}
    {/if}
    <footer>
      <button type="button" onclick={onclose} disabled={busy}>Abbrechen</button>
      <button type="button" onclick={() => save(true)} disabled={busy || !loaded}>Alles vorhanden</button>
      <button type="button" class="primary" onclick={() => save(false)} disabled={busy || !loaded}>
        {busy ? 'Speichert …' : 'Übernehmen'}
      </button>
    </footer>
  </div>
</dialog>

<style>
  dialog {
    width: min(32rem, calc(100vw - 2rem));
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

  h3 {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 0.4rem;
    margin: 0 0 0.25rem;
    font-size: 1rem;
  }

  h3 small {
    flex-basis: 100%;
    color: var(--accent);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .servings {
    color: var(--text-muted);
    font-size: 0.85rem;
    font-weight: normal;
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

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    border-bottom: 1px solid var(--border);
  }

  label {
    display: grid;
    grid-template-columns: 1.5rem 5.5rem 1fr;
    align-items: center;
    gap: 0.4rem;
    padding: 0.3rem 0;
    cursor: pointer;
  }

  label.off span {
    color: var(--text-muted);
    text-decoration: line-through;
  }

  input[type='checkbox'] {
    width: 1.1rem;
    height: 1.1rem;
    accent-color: var(--accent);
  }

  .amount {
    font-weight: 600;
    text-align: right;
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
