<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { focusPlanOn } from '../lib/planFocus.svelte.js';
  import { addDays, dayLabel, todayString } from '../lib/format.js';

  // recipe: das Rezept der Kochansicht. onadded() meldet, dass es
  // eingeplant wurde (die Kochansicht aktualisiert dann ihren Hinweis).
  let { recipe, onadded, onclose } = $props();

  const DAYS = 14;

  let slot = $state('dinner');
  let days = $state([]); // [{ date, entries }]
  let error = $state('');
  let busy = $state(false);
  let done = $state(null); // Tag, an dem eingeplant wurde

  let dialog;
  onMount(() => dialog.showModal());

  async function load() {
    try {
      const today = todayString();
      const result = await api.mealPlan(today, addDays(today, DAYS - 1));
      const list = [];
      for (let i = 0; i < DAYS; i++) {
        const date = addDays(result.today, i);
        list.push({ date, entries: result.entries.filter((e) => e.date === date) });
      }
      days = list;
    } catch (e) {
      error = e.message;
    }
  }
  load();

  const entriesIn = (day) => day.entries.filter((e) => e.slot === slot);
  const label = (e) => e.recipe?.title ?? e.note;

  /** Als (weiteres) Gericht an date einplanen. */
  async function add(date) {
    await run(date, () => api.createEntry({ date, slot, recipe_id: recipe.id }));
  }

  /** Das Gericht entry durch dieses Rezept ersetzen. */
  async function replace(entry) {
    if (!confirm(`„${label(entry)}“ durch „${recipe.title}“ ersetzen?`)) return;
    await run(entry.date, () =>
      api.updateEntry(entry.id, {
        recipe_id: recipe.id,
        servings: entry.servings,
        note: entry.recipe_id ? entry.note : '',
        status: entry.status === 'have_ingredients' ? 'planned' : entry.status,
      }),
    );
  }

  async function run(date, action) {
    busy = true;
    error = '';
    try {
      await action();
      done = date;
      onadded();
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }

  function showPlan() {
    focusPlanOn(done);
    navigate('/');
  }
</script>

<dialog
  bind:this={dialog}
  oncancel={onclose}
  onclick={(e) => e.target === dialog && onclose()}
  aria-labelledby="add-title"
>
  <div class="content">
    <header>
      <h2 id="add-title">Zum Essensplan hinzufügen</h2>
      <button type="button" class="close" title="Schließen" onclick={onclose}>✕</button>
    </header>

    {#if done}
      {@const l = dayLabel(done)}
      <p>„{recipe.title}“ ist für <strong>{l.weekday} {l.date}</strong> eingeplant.</p>
      <footer>
        <button type="button" class="secondary" onclick={onclose}>Schließen</button>
        <button type="button" class="primary" onclick={showPlan}>Zum Essensplan</button>
      </footer>
    {:else}
      <div class="slot-switch" role="group" aria-label="Mahlzeit">
        <button type="button" aria-pressed={slot === 'dinner'} onclick={() => (slot = 'dinner')}>Abendessen</button>
        <button type="button" aria-pressed={slot === 'lunch'} onclick={() => (slot = 'lunch')}>Mittagessen</button>
      </div>

      {#if error}<p class="error">{error}</p>{/if}

      <ul class="days">
        {#each days as day (day.date)}
          {@const l = dayLabel(day.date)}
          {@const taken = entriesIn(day)}
          <li>
            <span class="day">{l.weekday} <small>{l.date}</small></span>
            <span class="taken">
              {#each taken as entry (entry.id)}
                <span class="dish">
                  {label(entry)}
                  <button type="button" class="link" disabled={busy} onclick={() => replace(entry)}>ersetzen</button>
                </span>
              {:else}
                <span class="free">frei</span>
              {/each}
            </span>
            <button type="button" class="add" disabled={busy} onclick={() => add(day.date)}>
              {taken.length > 0 ? '+ dazu' : 'Einplanen'}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
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

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  p {
    margin: 0;
  }

  .close {
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: 50%;
    background: none;
    color: var(--text-muted);
    cursor: pointer;
  }

  .slot-switch {
    display: inline-flex;
    align-self: flex-start;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .slot-switch button {
    padding: 0.35rem 0.8rem;
    border: none;
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
  }

  .slot-switch button[aria-pressed='true'] {
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .error {
    color: var(--danger);
  }

  .days {
    display: flex;
    flex-direction: column;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .days li {
    display: grid;
    grid-template-columns: 4.5rem 1fr auto;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0;
    border-top: 1px solid var(--border);
  }

  .day {
    font-weight: 600;
  }

  .day small {
    font-weight: 400;
    color: var(--text-muted);
  }

  .taken {
    display: flex;
    flex-direction: column;
    min-width: 0;
    font-size: 0.85rem;
  }

  .dish {
    overflow-wrap: anywhere;
  }

  .free {
    color: var(--text-muted);
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    font-size: 0.8rem;
    text-decoration: underline;
    cursor: pointer;
  }

  .add,
  .primary,
  .secondary {
    padding: 0.35rem 0.75rem;
    border-radius: var(--radius);
    font: inherit;
    cursor: pointer;
  }

  .add {
    border: 1px solid var(--accent);
    background: var(--surface);
    color: var(--accent);
    font-size: 0.85rem;
    white-space: nowrap;
  }

  footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  .primary {
    border: 1px solid var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .secondary {
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
  }
</style>
