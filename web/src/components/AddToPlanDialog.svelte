<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { focusPlanOn } from '../lib/planFocus.svelte.js';
  import { addDays, dayLabel, todayString } from '../lib/format.js';
  import PlanPreview from './PlanPreview.svelte';
  import { t, errorText } from '../lib/i18n.svelte.js';
  import { slotChoices } from '../lib/slots.js';
  import { previewEnabled, createWithChoice, replaceWithChoice } from '../lib/planPreview.js';

  // recipe: das Rezept der Kochansicht. onadded() meldet, dass es
  // eingeplant wurde (die Kochansicht aktualisiert dann ihren Hinweis).
  let { recipe, onadded, onclose } = $props();

  const DAYS = 14;

  let slot = $state('dinner');
  let days = $state([]); // [{ date, entries }]
  let error = $state('');
  let busy = $state(false);
  let done = $state(null); // Tag, an dem eingeplant wurde
  // Offene Kurzvorschau: { date } zum Hinzufügen bzw. { entry } zum Ersetzen
  let pending = $state(null);
  let coffeeSlot = $state(true);
  api.getSettings().then((s) => (coffeeSlot = s.coffee_slot)).catch(() => {});

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
      error = errorText(e);
    }
  }
  load();

  const entriesIn = (day) => day.entries.filter((e) => e.slot === slot);
  const label = (e) => e.recipe?.title ?? e.note;

  /** Als (weiteres) Gericht an date einplanen – mit Kurzvorschau, falls eingeschaltet. */
  async function add(date) {
    if (await previewEnabled()) pending = { date };
    else await run(date, () => createWithChoice(date, slot, recipe.id, null));
  }

  /** Das Gericht entry durch dieses Rezept ersetzen. */
  async function replace(entry) {
    if (await previewEnabled()) {
      pending = { entry };
      return;
    }
    if (!confirm(t('plan.add.replace_confirm', { old: label(entry), title: recipe.title }))) return;
    await run(entry.date, () => replaceWithChoice(entry, recipe.id, null));
  }

  /** Ergebnis der Kurzvorschau übernehmen. */
  async function confirmPreview(choice) {
    const p = pending;
    if (p.entry) await run(p.entry.date, () => replaceWithChoice(p.entry, recipe.id, choice));
    else await run(p.date, () => createWithChoice(p.date, slot, recipe.id, choice));
    pending = null;
  }

  async function run(date, action) {
    busy = true;
    error = '';
    try {
      await action();
      done = date;
      onadded();
    } catch (e) {
      error = errorText(e);
    } finally {
      busy = false;
    }
  }

  // Satz „… ist für <Tag> eingeplant.“ in Teilen: der Tag wird fett gesetzt.
  const doneParts = $derived(done ? t('plan.add.done', { title: recipe.title, day: '{day}' }).split('{day}') : []);

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
      <h2 id="add-title">{t('plan.add.title')}</h2>
      <button type="button" class="close" title={t('common.close')} onclick={onclose}>✕</button>
    </header>

    {#if done}
      {@const l = dayLabel(done)}
      <p>{doneParts[0]}<strong>{l.weekday} {l.date}</strong>{doneParts.slice(1).join('')}</p>
      <footer>
        <button type="button" class="secondary" onclick={onclose}>{t('common.close')}</button>
        <button type="button" class="primary" onclick={showPlan}>{t('plan.add.show_plan')}</button>
      </footer>
    {:else}
      <div class="slot-switch" role="group" aria-label={t('plan.add.meal')}>
        {#each slotChoices(coffeeSlot, slot) as s (s.value)}
          <button type="button" aria-pressed={slot === s.value} onclick={() => (slot = s.value)}>{s.label}</button>
        {/each}
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
                  <button type="button" class="link" disabled={busy} onclick={() => replace(entry)}>{t('plan.add.replace')}</button>
                </span>
              {:else}
                <span class="free">{t('plan.add.free')}</span>
              {/each}
            </span>
            <button type="button" class="add" disabled={busy} onclick={() => add(day.date)}>
              {taken.length > 0 ? t('plan.add.add_more') : t('plan.add.add')}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</dialog>

{#if pending}
  <PlanPreview
    recipeId={recipe.id}
    servings={pending.entry ? pending.entry.servings : null}
    confirmLabel={pending.entry ? t('plan.add.replace_button') : t('common.add')}
    {busy}
    onconfirm={confirmPreview}
    onclose={() => (pending = null)}
  />
{/if}

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
    overflow-wrap: break-word;
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
