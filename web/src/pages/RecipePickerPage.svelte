<script>
  import { api, imageUrl } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { focusPlanOn } from '../lib/planFocus.svelte.js';
  import { dayLabel, planStatus } from '../lib/format.js';
  import Icon from '../components/Icon.svelte';
  import TagFilter from '../components/TagFilter.svelte';
  import PlanPreview from '../components/PlanPreview.svelte';
  import CourseTabs from '../components/CourseTabs.svelte';
  import { courseTabs } from '../lib/courseTabs.js';
  import { slotChoices } from '../lib/slots.js';
  import { previewEnabled, createWithChoice, replaceWithChoice } from '../lib/planPreview.js';

  // Zwei Arten, die Seite zu öffnen:
  //   #/plan/add/2026-10-24     neues Gericht für diesen Tag
  //   #/plan/replace/17         Rezept von Eintrag 17 ersetzen
  //   #/plan/menu/5             Gang zu Menü 5 hinzufügen (Schritt 9.4)
  let { date = null, entryId = null, menuId = null } = $props();
  // Die Seite wird bei jeder neuen Adresse neu aufgebaut; nur der
  // Anfangswert zählt.
  // svelte-ignore state_referenced_locally
  const replaceId = entryId;
  // svelte-ignore state_referenced_locally
  let targetDate = $state(date);

  let entry = $state(null); // der zu ersetzende Eintrag
  let slot = $state('dinner');
  let error = $state('');
  let saving = $state(false);

  // Eigener Filter, unabhängig von der Rezeptübersicht
  let query = $state('');
  let favoritesOnly = $state(false);
  let tagIds = $state([]);
  let sort = $state('name');
  let showTags = $state(false);
  let tagGroups = $state([]);
  let course = $state('all');
  let coffeeSlot = $state(true);
  api.getSettings().then((s) => (coffeeSlot = s.coffee_slot)).catch(() => {});

  /** Slot wählen; beim Kaffee gleich den Reiter „Backen“ zeigen. */
  function chooseSlot(value) {
    slot = value;
    if (value === 'coffee' && tabs.some((t) => t.key === 'baking')) course = 'baking';
  }
  const tabs = $derived(courseTabs(tagGroups));
  let recipes = $state(null);
  let note = $state('');
  // Rezept in der Kurzvorschau (Schritt 8.E), null = keine offen
  let previewId = $state(null);
  // Menü-Modus: das Menü, der Gang für das nächste Gericht, zuletzt Hinzugefügtes
  let menu = $state(null);
  let courseTag = $state(0);
  let added = $state('');
  // svelte-ignore state_referenced_locally
  if (menuId) {
    api
      .getMenu(menuId)
      .then((m) => {
        menu = m;
        targetDate = m.date;
        slot = m.slot;
      })
      .catch((e) => (error = e.status === 404 ? 'Dieses Menü gibt es nicht mehr.' : e.message));
  }
  const courseTags = $derived(tagGroups.find((g) => g.role === 'course')?.tags ?? []);
  // Reiter und Gang passen zusammen: „Nachspeisen“ → Nachspeise usw.
  $effect(() => {
    const tab = tabs.find((t) => t.key === course);
    if (menuId && tab && tab.ids.length === 1) courseTag = tab.ids[0];
  });

  if (replaceId) {
    api
      .getEntry(replaceId)
      .then((e) => {
        entry = e;
        targetDate = e.date;
        slot = e.slot;
      })
      .catch((e) => (error = e.status === 404 ? 'Diesen Eintrag gibt es nicht mehr.' : e.message));
  }
  api.listTagGroups().then((g) => (tagGroups = g)).catch(() => {});

  // Liste neu laden, wenn sich der Filter ändert (beim Tippen kurz warten).
  let requestCount = 0;
  $effect(() => {
    const filter = {
      query,
      favoritesOnly,
      tagIds: [...tagIds],
      sort,
      courseIds: tabs.find((t) => t.key === course)?.ids ?? [],
    };
    const id = ++requestCount;
    const timer = setTimeout(async () => {
      try {
        const list = await api.listRecipes(filter);
        if (id === requestCount) recipes = list;
      } catch (e) {
        if (id === requestCount) error = e.message;
      }
    }, filter.query ? 250 : 0);
    return () => clearTimeout(timer);
  });

  function toggleTag(id) {
    const i = tagIds.indexOf(id);
    if (i === -1) tagIds.push(id);
    else tagIds.splice(i, 1);
  }

  const heading = $derived.by(() => {
    if (!targetDate) return '';
    const l = dayLabel(targetDate);
    return `${l.weekday} ${l.date}`;
  });

  /** Rezept gewählt: erst die Kurzvorschau (falls eingeschaltet), dann eintragen. */
  async function pick(recipeId) {
    if (saving) return;
    if (menuId) {
      addCourse(recipeId, '');
      return;
    }
    if (await previewEnabled()) previewId = recipeId;
    else choose(recipeId);
  }

  /**
   * Trägt ein Rezept (recipeId) bzw. eine Notiz ein und geht zurück zum
   * Plan. choice: Ergebnis der Kurzvorschau ({ servings, excluded,
   * allAvailable }) oder null.
   */
  async function choose(recipeId, noteText = '', choice = null) {
    if (saving) return;
    saving = true;
    error = '';
    try {
      if (replaceId && recipeId === null) {
        await api.updateEntry(replaceId, {
          recipe_id: null,
          servings: entry.servings,
          note: noteText,
          status: entry.status === 'have_ingredients' ? 'planned' : entry.status,
        });
      } else if (replaceId) {
        await replaceWithChoice(entry, recipeId, choice);
      } else if (recipeId === null) {
        await api.createEntry({ date: targetDate, slot, recipe_id: null, note: noteText });
      } else {
        await createWithChoice(targetDate, slot, recipeId, choice);
      }
      focusPlanOn(targetDate);
      navigate('/');
    } catch (e) {
      error = e.message;
      saving = false;
    }
  }

  function chooseNote(event) {
    event.preventDefault();
    if (note.trim() === '') return;
    if (menuId) addCourse(null, note.trim());
    else choose(null, note.trim());
  }

  /**
   * Menü-Modus: Gericht (oder Text) als Gang anlegen und auf der Seite
   * bleiben, damit gleich der nächste Gang folgen kann. Portionen kommen
   * vom Menü; Zutaten abwählen geht danach im Menü für alle Gänge.
   */
  async function addCourse(recipeId, noteText) {
    saving = true;
    error = '';
    try {
      const entry = await api.createEntry({ menu_id: Number(menuId), recipe_id: recipeId, note: noteText, course_tag_id: Number(courseTag) || null });
      const name = courseTags.find((t) => t.id === Number(courseTag))?.name;
      added = `„${entry.recipe?.title ?? entry.note}“${name ? ` als ${name}` : ''} hinzugefügt.`;
      note = '';
      menu = await api.getMenu(menuId);
    } catch (e) {
      error = e.message;
    } finally {
      saving = false;
    }
  }

  /** Plus → „Menü zusammenstellen“: leeres Menü anlegen, dann Gänge wählen. */
  async function startMenu() {
    saving = true;
    try {
      const m = await api.createMenu({ date: targetDate, slot });
      navigate(`/plan/menu/${m.id}`);
    } catch (e) {
      error = e.message;
      saving = false;
    }
  }

  function back() {
    focusPlanOn(targetDate);
    navigate('/');
  }
</script>

<p class="back"><button type="button" class="link" onclick={back}>← Essensplan</button></p>

<h1>
  {#if menuId}
    Gang für Menü{#if menu}: {menu.display_title}{/if}
  {:else if replaceId}
    Ersetzen{#if entry}: {entry.recipe?.title ?? entry.note}{/if}
  {:else}
    Gericht für {heading}
  {/if}
</h1>

{#if error}<p class="error">{error}</p>{/if}

{#if menuId}
  <div class="menu-bar">
    <label>
      Gang
      <select bind:value={courseTag}>
        <option value={0}>– keiner –</option>
        {#each courseTags as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
      </select>
    </label>
    {#if menu}<span class="muted">{menu.entries.length === 1 ? '1 Gang' : `${menu.entries.length} Gänge`} im Menü</span>{/if}
    <button type="button" class="primary" onclick={back}>Fertig</button>
  </div>
  {#if added}<p class="ok">{added} Weiteren Gang wählen oder „Fertig“.</p>{/if}
{:else if !replaceId}
  <div class="slot-switch" role="group" aria-label="Mahlzeit">
    {#each slotChoices(coffeeSlot, slot) as s (s.value)}
      <button type="button" aria-pressed={slot === s.value} onclick={() => chooseSlot(s.value)}>{s.label}</button>
    {/each}
  </div>
{/if}

{#if !menuId && !replaceId}
  <button type="button" class="secondary menu-start" disabled={saving} onclick={startMenu}>Menü zusammenstellen …</button>
{/if}

<form class="note-form" onsubmit={chooseNote}>
  <input placeholder="Ohne Rezept, z. B. Essen gehen" aria-label="Eintrag ohne Rezept" bind:value={note} maxlength="200" />
  <button type="submit" class="secondary" disabled={note.trim() === '' || saving}>Eintragen</button>
</form>
<div class="quick">
  {#each ['Essen gehen', 'Reste', 'Pizza bestellen'] as text (text)}
    <button type="button" class="chip" disabled={saving} onclick={() => (menuId ? addCourse(null, text) : choose(null, text))}>{text}</button>
  {/each}
</div>

<!-- Gang-Reiter und Suche haften beim Scrollen oben (die ganze Seite
     scrollt, keine eigene Scrollbox für die Rezepte). -->
<div class="finder">
  <CourseTabs {tabs} value={course} onchange={(key) => (course = key)} />
  <label class="search">
    <Icon name="search" size={18} />
    <input type="search" placeholder="Rezept suchen" aria-label="Rezepte durchsuchen" bind:value={query} />
  </label>
</div>

<div class="toolbar">
  <button type="button" class="toggle" aria-expanded={showTags} onclick={() => (showTags = !showTags)}>
    <Icon name="filter" size={18} />{tagIds.length > 0 ? ` ${tagIds.length}` : ''}
  </button>
  <button type="button" class="toggle" aria-pressed={favoritesOnly} title="Nur Favoriten" onclick={() => (favoritesOnly = !favoritesOnly)}>
    <Icon name="heart" size={18} filled={favoritesOnly} />
  </button>
  <select bind:value={sort} aria-label="Sortierung">
    <option value="name">Name</option>
    <option value="last_planned">Zuletzt eingeplant</option>
    <option value="most_planned">Am häufigsten eingeplant</option>
    <option value="newest">Neu hinzugefügt</option>
  </select>
</div>
{#if showTags}
  <div class="tags"><TagFilter groups={tagGroups} selected={tagIds} ontoggle={toggleTag} /></div>
{/if}

{#if recipes === null}
  <p class="muted">Lädt …</p>
{:else if recipes.length === 0}
  <p class="muted">Keine passenden Rezepte.</p>
{:else}
  <ul class="list">
    {#each recipes as recipe (recipe.id)}
      <li>
        <button type="button" class="row" disabled={saving} onclick={() => pick(recipe.id)}>
          <span class="thumb">
            {#if recipe.thumbnail}
              <img src={imageUrl(recipe.thumbnail)} alt="" loading="lazy" />
            {:else}
              <Icon name="dish" size={22} />
            {/if}
          </span>
          <span class="text">
            <span class="title">{recipe.title}{#if recipe.parent_recipe_id != null}<small class="variant-badge">Variante</small>{/if}</span>
            <span class="status" class:upcoming={recipe.next_planned}>{planStatus(recipe)}</span>
          </span>
          {#if recipe.is_favorite}<span class="fav"><Icon name="heart" size={16} filled /></span>{/if}
        </button>
      </li>
    {/each}
  </ul>
{/if}

{#if previewId}
  <PlanPreview
    recipeId={previewId}
    servings={replaceId ? entry?.servings : null}
    confirmLabel={replaceId ? 'Ersetzen' : 'Hinzufügen'}
    busy={saving}
    onconfirm={(choice) => choose(previewId, '', choice)}
    onclose={() => (previewId = null)}
  />
{/if}

<style>
  .menu-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.75rem;
  }

  .menu-bar select {
    margin-left: 0.4rem;
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .menu-bar .primary {
    margin-left: auto;
    padding: 0.4rem 1rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--on-accent);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }

  .muted {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .ok {
    color: var(--accent);
  }

  .menu-start {
    margin-bottom: 0.5rem;
  }

  .variant-badge {
    margin-left: 0.4rem;
    padding: 0 0.4rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    font-size: 0.72rem;
    font-weight: normal;
    color: var(--text-muted);
  }

  .back {
    margin: 0 0 0.75rem;
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    cursor: pointer;
  }

  h1 {
    overflow-wrap: break-word;
  }

  .error {
    color: var(--danger);
  }

  .muted {
    color: var(--text-muted);
  }

  .slot-switch {
    display: inline-flex;
    margin-bottom: 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .slot-switch button {
    padding: 0.4rem 0.9rem;
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

  .note-form {
    display: flex;
    gap: 0.5rem;
  }

  .note-form input,
  .search input,
  select {
    min-width: 0;
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .note-form input {
    flex: 1;
  }

  .secondary,
  .chip,
  .toggle {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
  }

  .secondary {
    padding: 0 0.9rem;
  }

  .quick {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin: 0.5rem 0 1.25rem;
  }

  .chip {
    padding: 0.25rem 0.75rem;
    border-radius: 999px;
    font-size: 0.85rem;
  }

  .finder {
    margin: 0 -0.25rem 0.5rem;
    padding: 0 0.25rem 0.5rem;
    background: var(--bg);
  }

  /* Nur haften, wenn darunter genug Platz für Rezepte bleibt. */
  @media (min-height: 34rem) {
    .finder {
      position: sticky;
      top: 0;
      z-index: 5;
    }
  }

  .toolbar {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }

  .toolbar select {
    flex: 1;
  }

  .search {
    position: relative;
    display: flex;
    align-items: center;
    color: var(--text-muted);
  }

  .search :global(svg) {
    position: absolute;
    left: 0.6rem;
  }

  .search input {
    width: 100%;
    padding-left: 2.1rem;
  }

  .toggle {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0 0.6rem;
  }

  .toggle[aria-pressed='true'],
  .toggle[aria-expanded='true'] {
    border-color: var(--accent);
    color: var(--accent);
  }

  .tags {
    margin-bottom: 0.75rem;
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.4rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: inherit;
    text-align: left;
    font: inherit;
    cursor: pointer;
  }

  .row:hover {
    border-color: var(--accent);
  }

  .thumb {
    display: grid;
    flex: none;
    place-items: center;
    width: 4rem;
    aspect-ratio: 4 / 3;
    overflow: hidden;
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--accent-soft);
    color: var(--accent);
  }

  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .text {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
  }

  .title {
    font-weight: 600;
    overflow-wrap: break-word;
    /* Lange Titel nach zwei Zeilen mit „…“ abschneiden */
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .status {
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .status.upcoming {
    color: var(--accent);
  }

  .fav {
    display: inline-flex;
    color: var(--accent);
  }
</style>
