<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { dayLabel } from '../lib/format.js';
  import { slotLabel } from '../lib/slots.js';
  import MenuPreview from './MenuPreview.svelte';

  // Menü eines Menüs (Schritt 9.4): Titel, Notiz, Portionen, Gänge
  // (Gang ändern, lösen, hinzufügen), Zutaten vorhanden, verschieben,
  // auflösen, löschen.
  //   menuId     das Menü
  //   courses    Gänge der Gang-Gruppe [{ id, name }]
  //   onchanged(dates)  etwas wurde gespeichert (Tage zum Neuladen)
  //   onmove(menu, date) verschieben auf einen anderen Tag
  //   onclose()
  let { menuId, courses, onchanged, onmove, onclose } = $props();

  let dialog;
  onMount(() => dialog.showModal());

  let menu = $state(null);
  let title = $state('');
  let note = $state('');
  let servings = $state(4);
  let applyServings = $state(true);
  let moveDate = $state('');
  let error = $state('');
  let busy = $state(false);
  let showPreview = $state(false);

  async function load() {
    try {
      menu = await api.getMenu(menuId);
      title = menu.title;
      note = menu.note;
      servings = menu.servings;
      moveDate = menu.date;
    } catch (e) {
      error = e.message;
    }
  }
  load();

  const courseName = (id) => courses.find((c) => c.id === id)?.name ?? '';
  const changed = $derived(menu && (title !== menu.title || note !== menu.note || servings !== menu.servings));

  async function run(action, close = false) {
    busy = true;
    error = '';
    try {
      await action();
      onchanged([menu.date]);
      if (close) onclose();
      else await load();
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }

  function save(event) {
    event.preventDefault();
    run(() => api.updateMenu(menu.id, { title, note, servings: Number(servings), apply_servings: applyServings }));
  }

  function setCourse(entry, value) {
    run(() =>
      api.updateEntry(entry.id, {
        recipe_id: entry.recipe_id,
        servings: entry.servings,
        note: entry.note,
        status: entry.status,
        course_tag_id: Number(value),
      }),
    );
  }

  function detach(entry) {
    run(() => api.detachEntry(entry.id));
  }

  function dissolve() {
    if (!confirm('Menü auflösen? Die Gänge bleiben als einzelne Gerichte im Plan.')) return;
    run(() => api.deleteMenu(menu.id), true);
  }

  function remove() {
    if (!confirm(`Menü „${menu.display_title}“ mit allen ${menu.entries.length} Gängen löschen?`)) return;
    run(() => api.deleteMenu(menu.id, true), true);
  }

  const label = $derived(menu ? dayLabel(menu.date) : null);
</script>

<dialog bind:this={dialog} oncancel={onclose} onclick={(e) => e.target === dialog && onclose()} aria-labelledby="menu-title">
  <div class="content">
    {#if !menu}
      {#if error}<p class="error">{error}</p>{:else}<p>Lädt …</p>{/if}
    {:else}
      <header>
        <div>
          <p class="when">{label.weekday} {label.date} · {slotLabel(menu.slot)}</p>
          <h2 id="menu-title">Menü: {menu.display_title}</h2>
        </div>
        <button type="button" class="icon-button" title="Schließen" onclick={onclose}>✕</button>
      </header>

      <form class="fields" onsubmit={save}>
        <label>
          <span>Titel</span>
          <input bind:value={title} maxlength="200" placeholder="{menu.display_title} (automatisch)" />
        </label>
        <label>
          <span>Notiz</span>
          <textarea bind:value={note} rows="2" maxlength="2000" placeholder="z. B. Suppe vorher kochen, Nachtisch am Vortag"></textarea>
        </label>
        <div class="row">
          <span>Personen</span>
          <div class="stepper">
            <button type="button" aria-label="Weniger" disabled={servings <= 1} onclick={() => servings--}>−</button>
            <output>{servings}</output>
            <button type="button" aria-label="Mehr" disabled={servings >= 50} onclick={() => servings++}>+</button>
          </div>
          {#if servings !== menu.servings}
            <label class="check"><input type="checkbox" bind:checked={applyServings} /> auf alle Gänge übertragen</label>
          {/if}
        </div>
        {#if changed}<button type="submit" class="primary" disabled={busy}>Speichern</button>{/if}
      </form>

      <h3>Gänge</h3>
      <ul class="courses">
        {#each menu.entries as e (e.id)}
          <li>
            <select aria-label="Gang" value={e.course_tag_id ?? 0} onchange={(ev) => setCourse(e, ev.currentTarget.value)} disabled={busy}>
              <option value={0}>– Gang –</option>
              {#each courses as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
            </select>
            <span class="dish">
              {#if e.recipe_id}
                <a href="#/recipes/{e.recipe_id}/entry/{e.id}" onclick={onclose}>{e.recipe?.title ?? e.note}</a>
              {:else}
                {e.note}
              {/if}
              <small>
                {e.recipe_id ? `${e.servings} P.` : 'ohne Rezept'}
                {#if e.status === 'transferred'} · an Bring{:else if e.status === 'have_ingredients'} · Zutaten vorhanden{/if}
                {#if e.excluded_ingredients?.length} · {e.excluded_ingredients.length} vorhanden{/if}
              </small>
            </span>
            <button type="button" class="link" onclick={() => detach(e)} disabled={busy} title="Als einzelnes Gericht im Plan lassen">lösen</button>
          </li>
        {:else}
          <li class="hint">Noch keine Gänge.</li>
        {/each}
      </ul>
      <div class="actions">
        <button type="button" class="primary" onclick={() => { onclose(); navigate(`/plan/menu/${menu.id}`); }}>+ Gang hinzufügen</button>
        {#if menu.entries.some((e) => e.recipe_id)}
          <button type="button" onclick={() => (showPreview = true)} disabled={busy}>Zutaten vorhanden …</button>
        {/if}
        <a class="button" href="#/menus/{menu.id}" onclick={onclose}>Kochansicht</a>
      </div>

      <div class="move">
        <label for="menu-move-date">Verschieben auf</label>
        <input id="menu-move-date" type="date" bind:value={moveDate} />
        <button type="button" disabled={!moveDate || moveDate === menu.date || busy} onclick={() => onmove(menu, moveDate)}>Verschieben</button>
      </div>

      {#if error}<p class="error">{error}</p>{/if}

      <footer>
        <button type="button" onclick={dissolve} disabled={busy}>Menü auflösen</button>
        <button type="button" class="danger" onclick={remove} disabled={busy}>Menü löschen</button>
      </footer>
    {/if}
  </div>
</dialog>

{#if showPreview && menu}
  <MenuPreview
    entries={menu.entries}
    {courseName}
    onsaved={() => {
      showPreview = false;
      onchanged([menu.date]);
      load();
    }}
    onclose={() => (showPreview = false)}
  />
{/if}

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

  header {
    display: flex;
    justify-content: space-between;
    gap: 0.5rem;
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  h3 {
    margin: 0;
    font-size: 1rem;
  }

  p {
    margin: 0;
  }

  .when,
  .hint {
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  .error {
    color: var(--danger);
  }

  .fields {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .fields label {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .fields label > span,
  .row > span {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  input,
  textarea,
  select {
    padding: 0.4rem 0.55rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
  }

  .stepper {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .stepper button {
    width: 2rem;
    height: 2rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    cursor: pointer;
  }

  .check {
    flex-direction: row !important;
    align-items: center;
    font-size: 0.85rem;
  }

  .courses {
    margin: 0;
    padding: 0;
    list-style: none;
    border-top: 1px solid var(--border);
  }

  .courses li {
    display: grid;
    grid-template-columns: 8.5rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.5rem;
    padding: 0.35rem 0;
    border-bottom: 1px solid var(--border);
  }

  .dish {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .dish small {
    color: var(--text-muted);
  }

  .actions,
  .move,
  footer {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  footer {
    justify-content: space-between;
  }

  button,
  .button {
    padding: 0.45rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    text-decoration: none;
    cursor: pointer;
  }

  .primary {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .danger {
    color: var(--danger);
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
  }

  .icon-button {
    width: 2rem;
    height: 2rem;
    padding: 0;
    border: none;
    background: none;
  }
</style>
