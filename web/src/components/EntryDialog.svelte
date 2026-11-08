<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { dayLabel } from '../lib/format.js';
  import Icon from './Icon.svelte';
  import PlanPreview from './PlanPreview.svelte';
  import ResetTransferDialog from './ResetTransferDialog.svelte';
  import { slotLabel } from '../lib/slots.js';
  import { t, errorText } from '../lib/i18n.svelte.js';

  // entry: der Planeintrag. onsaved(entry) meldet den neuen Stand,
  // ondeleted(id) das Entfernen, onmove(entry, date) den Wunsch, den
  // Eintrag auf einen anderen Tag zu verschieben (ohne Ziehen), onclose das
  // Schließen ohne Änderung.
  let { entry, onsaved, ondeleted, onmove, onclose, ontomenu = null } = $props();

  // Bearbeitet wird eine Kopie; übernommen wird erst beim Speichern.
  // svelte-ignore state_referenced_locally
  let servings = $state(entry.servings);
  // svelte-ignore state_referenced_locally
  let note = $state(entry.note);
  // svelte-ignore state_referenced_locally
  let haveIngredients = $state(entry.status === 'have_ingredients');
  // svelte-ignore state_referenced_locally
  let isFavorite = $state(entry.recipe?.is_favorite ?? false);
  // svelte-ignore state_referenced_locally
  let moveDate = $state(entry.date);
  let error = $state('');
  let busy = $state(false);
  let showPreview = $state(false);
  let showReset = $state(false);
  // Nach dem Zurücksetzen: Beim Schließen den neuen Stand melden.
  let wasReset = false;

  async function resetClosed() {
    showReset = false;
    if (!wasReset) return;
    try {
      onsaved(await api.getEntry(entry.id));
    } catch (e) {
      error = errorText(e);
    }
  }

  let dialog;
  // showModal öffnet den Dialog über der Seite; Esc schließt ihn.
  onMount(() => dialog.showModal());

  const label = $derived(dayLabel(entry.date));
  const title = $derived(entry.recipe ? entry.recipe.title : entry.note);
  const transferred = $derived(entry.status === 'transferred');

  async function save(event) {
    event.preventDefault();
    if (!entry.recipe_id && note.trim() === '') {
      error = t('plan.entry.note_required');
      return;
    }
    busy = true;
    error = '';
    try {
      const status = transferred ? 'transferred' : haveIngredients ? 'have_ingredients' : 'planned';
      const saved = await api.updateEntry(entry.id, {
        recipe_id: entry.recipe_id,
        servings: Number(servings),
        note: note.trim(),
        status,
      });
      onsaved(saved);
    } catch (e) {
      error = errorText(e);
      busy = false;
    }
  }

  /** Abwahl aus der Kurzvorschau speichern (Portionen und Notiz gleich mit). */
  async function saveChoice(choice) {
    busy = true;
    error = '';
    try {
      let status = transferred ? 'transferred' : choice.allAvailable ? 'have_ingredients' : 'planned';
      const saved = await api.updateEntry(entry.id, {
        recipe_id: entry.recipe_id,
        servings: choice.servings,
        note: note.trim(),
        status,
        excluded_ingredients: choice.excluded,
      });
      showPreview = false;
      onsaved(saved);
    } catch (e) {
      error = errorText(e);
      busy = false;
    }
  }

  async function toggleFavorite() {
    const next = !isFavorite;
    isFavorite = next;
    try {
      await api.setFavorite(entry.recipe_id, next);
      // Auch im Plan sofort zeigen, ohne die übrigen Änderungen zu speichern.
      entry.recipe.is_favorite = next;
    } catch (e) {
      isFavorite = !next;
      error = t('plan.entry.favorite_failed', { message: errorText(e) });
    }
  }

  async function remove() {
    if (!confirm(t('plan.entry.remove_confirm', { title }))) return;
    busy = true;
    try {
      await api.deleteEntry(entry.id);
      ondeleted(entry.id);
    } catch (e) {
      error = errorText(e);
      busy = false;
    }
  }
</script>

<!-- oncancel: Esc-Taste; Klick auf den abgedunkelten Rand schließt ebenfalls. -->
<dialog
  bind:this={dialog}
  oncancel={onclose}
  onclick={(e) => e.target === dialog && onclose()}
  aria-labelledby="entry-title"
>
  <form onsubmit={save}>
    <header>
      <div>
        <p class="when">{label.weekday} {label.date} · {slotLabel(entry.slot)}</p>
        <h2 id="entry-title">{title}</h2>
      </div>
      <button type="button" class="icon-button" title={t('common.close')} onclick={onclose}>✕</button>
    </header>

    {#if entry.recipe_id}
      <div class="row">
        <span>{entry.recipe?.yield_unit || t('plan.entry.servings')}</span>
        <div class="stepper">
          <button type="button" aria-label={t('plan.entry.fewer')} disabled={servings <= 1} onclick={() => servings--}>−</button>
          <output>{servings}</output>
          <button type="button" aria-label={t('plan.entry.more')} disabled={servings >= 50} onclick={() => servings++}>+</button>
        </div>
      </div>
    {/if}

    <label class="field">
      <span>{entry.recipe_id ? t('plan.entry.note_optional') : t('plan.entry.note')}</span>
      <input bind:value={note} maxlength="200" placeholder={entry.recipe_id ? t('plan.entry.note_placeholder_recipe') : t('plan.entry.note_placeholder')} />
    </label>

    {#if entry.recipe_id}
      {#if transferred}
        <p class="hint">
          <Icon name="cart" size={16} /> {t('plan.entry.sent_to_bring')}
          <button type="button" class="link" onclick={() => (showReset = true)}>{t('plan.entry.undo_sending')}</button>
        </p>
      {:else}
        <label class="check">
          <input type="checkbox" bind:checked={haveIngredients} />
          {t('plan.entry.have_ingredients')}
        </label>
      {/if}
      <p class="hint">
        {#if entry.excluded_ingredients?.length}
          {t('plan.entry.excluded_count', { count: entry.excluded_ingredients.length })}
        {/if}
        <button type="button" class="link" onclick={() => (showPreview = true)}>{t('plan.entry.have_ingredients_open')}</button>
        {#if transferred}<small>{t('plan.entry.already_sent')}</small>{/if}
      </p>
    {/if}

    <div class="move">
      <label for="move-date">{t('plan.entry.move_to')}</label>
      <input id="move-date" type="date" bind:value={moveDate} />
      <button
        type="button"
        class="secondary"
        disabled={!moveDate || moveDate === entry.date || busy}
        onclick={() => onmove(entry, moveDate)}
      >
        {t('plan.entry.move')}
      </button>
    </div>

    {#if error}<p class="error">{error}</p>{/if}

    <div class="links">
      {#if entry.recipe_id}
        <a href="#/recipes/{entry.recipe_id}/entry/{entry.id}" onclick={onclose}>{t('plan.entry.open_cooking_view')}</a>
        <button type="button" class="link" onclick={toggleFavorite}>
          <Icon name="heart" size={16} filled={isFavorite} />
          {isFavorite ? t('plan.entry.favorite') : t('plan.entry.make_favorite')}
        </button>
      {/if}
      <button type="button" class="link" onclick={() => navigate(`/plan/replace/${entry.id}`)}>
        {entry.recipe_id ? t('plan.entry.replace_recipe') : t('plan.entry.choose_recipe')}
      </button>
      {#if ontomenu && !entry.menu_id}
        <!-- Menü (Schritt 9.4): weitere Gänge dazu, z. B. Suppe und Nachtisch -->
        <button type="button" class="link" onclick={() => ontomenu(entry)}>{t('plan.entry.to_menu')}</button>
      {/if}
    </div>

    <footer>
      <button type="button" class="danger" disabled={busy} onclick={remove}>{t('common.remove')}</button>
      <button type="submit" class="primary" disabled={busy}>{t('common.save')}</button>
    </footer>
  </form>
</dialog>

{#if showPreview}
  <PlanPreview
    recipeId={entry.recipe_id}
    {servings}
    excluded={entry.excluded_ingredients ?? []}
    confirmLabel={t('common.apply')}
    {busy}
    onconfirm={saveChoice}
    onclose={() => (showPreview = false)}
  />
{/if}

{#if showReset}
  <ResetTransferDialog entryId={entry.id} ondone={() => (wasReset = true)} onclose={resetClosed} />
{/if}

<style>
  dialog {
    width: min(26rem, calc(100vw - 2rem));
    padding: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
  }

  dialog::backdrop {
    background: rgb(0 0 0 / 0.4);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    padding: 1rem;
  }

  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 0.5rem;
  }

  .when {
    margin: 0;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
    overflow-wrap: break-word;
  }

  .icon-button {
    flex: none;
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: 50%;
    background: none;
    color: var(--text-muted);
    cursor: pointer;
  }

  .icon-button:hover {
    background: var(--surface-hover);
  }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .row > span,
  .field span {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .stepper {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .stepper button {
    width: 2.25rem;
    height: 2.25rem;
    border: 1px solid var(--border);
    border-radius: 50%;
    background: var(--surface);
    color: var(--text);
    font-size: 1.2rem;
    cursor: pointer;
  }

  .stepper button:disabled {
    opacity: 0.4;
    cursor: default;
  }

  output {
    min-width: 2ch;
    font-size: 1.1rem;
    font-weight: 600;
    text-align: center;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .field input {
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .check {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .check input {
    width: 1.1rem;
    height: 1.1rem;
    accent-color: var(--accent);
  }

  .hint {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0;
    color: var(--text-muted);
  }

  .error {
    margin: 0;
    color: var(--danger);
  }

  .links {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem 1.25rem;
    font-size: 0.9rem;
  }

  .link {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    font: inherit;
    cursor: pointer;
  }

  .move {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  .move label {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .move input {
    padding: 0.4rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .secondary {
    padding: 0.4rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }

  .secondary:disabled {
    opacity: 0.5;
    cursor: default;
  }

  footer {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
    padding-top: 0.25rem;
  }

  .primary,
  .danger {
    padding: 0.5rem 1.1rem;
    border-radius: var(--radius);
    cursor: pointer;
    font: inherit;
  }

  .primary {
    border: 1px solid var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .danger {
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--danger);
  }

  .primary:disabled,
  .danger:disabled {
    opacity: 0.6;
  }
</style>
