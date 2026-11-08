<script>
  import { api } from '../lib/api.js';
  import { recipeFilter, toggleTag, resetFilter, hasActiveFilter, assignPreset } from '../lib/recipeFilter.svelte.js';
  import Icon from '../components/Icon.svelte';
  import RecipeTile from '../components/RecipeTile.svelte';
  import TagFilter from '../components/TagFilter.svelte';
  import CourseTabs from '../components/CourseTabs.svelte';
  import { courseTabs } from '../lib/courseTabs.js';
  import { t, errorText } from '../lib/i18n.svelte.js';

  let recipes = $state(null); // null = noch nicht geladen
  let error = $state('');
  let tagGroups = $state([]);
  let showTags = $state(recipeFilter.tagIds.length > 0);
  const tabs = $derived(courseTabs(tagGroups, true));
  const courseGroup = $derived(tagGroups.find((g) => g.role === 'course'));
  const courseIds = $derived(new Set(courseGroup?.tags.map((tag) => tag.id) ?? []));
  const tagName = $derived(new Map(tagGroups.flatMap((g) => g.tags.map((tag) => [tag.id, tag.name]))));
  /** Namen der Gänge eines Rezepts für das Etikett auf der Kachel. */
  const coursesOf = (recipe) => (recipe.tag_ids ?? []).filter((id) => courseIds.has(id)).map((id) => tagName.get(id));
  // Anzahl für den Reiter „Ohne Gang (214)“ – mit Suche und übrigen Filtern.
  let noneCount = $state(null);

  // Die Kategorien ändern sich selten; einmal beim Öffnen laden genügt.
  api.listTagGroups().then((groups) => (tagGroups = groups)).catch(() => {});

  // Zählt die Anfragen mit. Kommt eine ältere Antwort erst nach einer
  // neueren an (langsames Netz), wird sie verworfen.
  let requestCount = 0;
  let lastQuery = recipeFilter.query;

  async function load(filter) {
    const id = ++requestCount;
    try {
      const list = await api.listRecipes(filter);
      if (id !== requestCount) return;
      recipes = list;
      error = '';
      // „Ohne Gang“ zählen: aus der Liste, wenn sie das hergibt, sonst eigene Anfrage.
      if (filter.withoutCourse) noneCount = list.length;
      else if (filter.courseIds.length === 0) noneCount = list.filter((r) => coursesOf(r).length === 0).length;
      else {
        const none = await api.listRecipes({ ...filter, courseIds: [], withoutCourse: true });
        if (id === requestCount) noneCount = none.length;
      }
    } catch (e) {
      if (id !== requestCount) return;
      error = errorText(e);
    }
  }

  // $effect läuft beim Start und jedes Mal, wenn sich ein darin gelesener
  // Wert ändert – hier also bei jeder Änderung am Filter.
  $effect(() => {
    const filter = {
      query: recipeFilter.query,
      tagIds: [...recipeFilter.tagIds],
      favoritesOnly: recipeFilter.favoritesOnly,
      illustrationsOnly: recipeFilter.illustrationsOnly,
      neverPlanned: recipeFilter.neverPlanned,
      courseIds: tabs.find((tab) => tab.key === recipeFilter.course)?.ids ?? [],
      withoutCourse: recipeFilter.course === 'none',
      sort: recipeFilter.sort,
    };
    lastFilter = filter;
    // Beim Tippen kurz warten, damit nicht jeder Buchstabe eine Anfrage
    // auslöst; Klicks auf Filter wirken sofort.
    const delay = filter.query !== lastQuery ? 250 : 0;
    lastQuery = filter.query;
    const timer = setTimeout(() => load(filter), delay);
    // Die zurückgegebene Funktion räumt auf, bevor der Effekt erneut läuft:
    // Ein noch wartender Aufruf entfällt.
    return () => clearTimeout(timer);
  });

  const activeTagCount = $derived(recipeFilter.tagIds.length);

  // Zuweisungsmodus: Kacheln auswählen und ihnen gesammelt einen Gang (oder
  // eine Kategorie) geben bzw. nehmen. Der Gang kommt hinzu; vorhandene
  // Gänge bleiben (Entscheidung 2026-10-24). Die Liste bleibt stabil, bis
  // „Übernehmen“ sie neu lädt.
  let lastFilter = null;
  let assigning = $state(false);
  let target = $state(''); // 'add:12', 'remove:12' oder 'remove:courses'
  let selected = $state(new Set());
  let applied = $state(false); // schon etwas übernommen → „Fertig“ statt „Abbrechen“
  let busy = $state(false);
  // { key, params, undo: { add, remove } | null } – der Text wird erst beim
  // Anzeigen übersetzt, damit ein Sprachwechsel ihn mitnimmt.
  let notice = $state(null);
  let noticeTimer;

  // Vorauswahl aus Einstellungen → Gänge („Vorschläge prüfen“).
  if (assignPreset.tagId) {
    assigning = true;
    target = `add:${assignPreset.tagId}`;
    selected = new Set(assignPreset.ids);
    assignPreset.tagId = 0;
    assignPreset.ids = [];
  }

  const [action, targetPart] = $derived(target ? target.split(':') : ['', '']);
  const targetIds = $derived(
    targetPart === 'courses' ? [...courseIds] : targetPart ? [Number(targetPart)] : [],
  );
  // Name der Ziel-Kategorie (Daten des Nutzers, nicht übersetzt).
  const targetName = $derived(targetPart && targetPart !== 'courses' ? (tagName.get(Number(targetPart)) ?? '') : '');
  const targetIsCourse = $derived(targetIds.length > 0 && targetIds.every((id) => courseIds.has(id)));

  /** Die geladenen Rezepte der Auswahl (für Hinweise und Änderungen). */
  const selectedRecipes = $derived((recipes ?? []).filter((r) => selected.has(r.id)));

  // Hinweis vor dem Übernehmen: Wer hat schon einen anderen Gang (bleibt)?
  const otherCourse = $derived(
    action === 'add' && targetIsCourse
      ? selectedRecipes.filter((r) => (r.tag_ids ?? []).some((id) => courseIds.has(id) && id !== targetIds[0])).length
      : 0,
  );

  function startAssign() {
    assigning = true;
    applied = false;
    if (!target && courseGroup?.tags.length) target = `add:${courseGroup.tags[0].id}`;
  }

  function stopAssign() {
    assigning = false;
    selected = new Set();
  }

  function toggleSelect(recipe) {
    const next = new Set(selected);
    if (next.has(recipe.id)) next.delete(recipe.id);
    else next.add(recipe.id);
    selected = next;
  }

  function selectVisible() {
    selected = new Set([...selected, ...(recipes ?? []).map((r) => r.id)]);
  }

  function showNotice(key, params = {}, undo = null) {
    clearTimeout(noticeTimer);
    notice = { key, params, undo };
    noticeTimer = setTimeout(() => (notice = null), 8000);
  }

  async function apply() {
    // Auch ausgewählte Rezepte, die gerade nicht sichtbar sind (anderer Filter).
    const ids = [...selected];
    const pairs = ids.flatMap((rid) => targetIds.map((tid) => ({ recipe_id: rid, tag_id: tid })));
    if (pairs.length === 0) return;
    busy = true;
    try {
      const result = action === 'add' ? await api.bulkTags(pairs) : await api.bulkTags([], pairs);
      const n = result.recipes;
      if (n === 0) showNotice('recipes.notice.unchanged');
      else {
        const key =
          action === 'add'
            ? 'recipes.notice.added'
            : targetPart === 'courses'
              ? 'recipes.notice.removed_all_courses'
              : 'recipes.notice.removed';
        showNotice(key, { name: targetName, count: n }, { add: result.removed, remove: result.added });
      }
      selected = new Set();
      applied = true;
      if (lastFilter) await load(lastFilter);
    } catch (e) {
      showNotice('recipes.notice.not_saved', { message: errorText(e) });
    } finally {
      busy = false;
    }
  }

  async function undo() {
    const { add, remove } = notice.undo;
    busy = true;
    try {
      await api.bulkTags(add, remove);
      showNotice('recipes.notice.undone');
      if (lastFilter) await load(lastFilter);
    } catch (e) {
      showNotice('recipes.notice.undo_failed', { message: errorText(e) });
    } finally {
      busy = false;
    }
  }
</script>

<div class="page-head">
  <h1>{t('recipes.title')}</h1>
  <div class="head-actions">
    <a class="secondary" href="#/recipes/import">{t('recipes.import_file')}</a>
    <a class="new" href="#/recipes/new">+ {t('recipes.new_recipe')}</a>
  </div>
</div>

<CourseTabs
  {tabs}
  value={recipeFilter.course}
  counts={noneCount != null ? { none: noneCount } : {}}
  onchange={(key) => (recipeFilter.course = key)}
/>

<div class="toolbar">
  <label class="search">
    <Icon name="search" size={18} />
    <input
      type="search"
      placeholder={t('recipes.search_placeholder')}
      aria-label={t('recipes.search_label')}
      bind:value={recipeFilter.query}
    />
  </label>

  <div class="controls">
    <button
      type="button"
      class="toggle"
      aria-expanded={showTags}
      onclick={() => (showTags = !showTags)}
    >
      <Icon name="filter" size={18} />
      {t('recipes.categories')}{activeTagCount > 0 ? ` (${activeTagCount})` : ''}
    </button>

    <button
      type="button"
      class="toggle"
      aria-pressed={recipeFilter.favoritesOnly}
      title={t('recipes.favorites_only')}
      onclick={() => (recipeFilter.favoritesOnly = !recipeFilter.favoritesOnly)}
    >
      <Icon name="heart" size={18} filled={recipeFilter.favoritesOnly} />
      <span class="label-wide">{t('recipes.favorites')}</span>
    </button>

    <button
      type="button"
      class="toggle"
      aria-pressed={recipeFilter.illustrationsOnly}
      title={t('recipes.illustrations_only')}
      onclick={() => (recipeFilter.illustrationsOnly = !recipeFilter.illustrationsOnly)}
    >
      <Icon name="brush" size={18} />
      <span class="label-wide">{t('recipes.illustrations')}</span>
    </button>

    <button
      type="button"
      class="toggle"
      aria-pressed={recipeFilter.neverPlanned}
      title={t('recipes.never_planned_only')}
      onclick={() => (recipeFilter.neverPlanned = !recipeFilter.neverPlanned)}
    >
      <Icon name="sparkle" size={18} />
      <span class="label-wide">{t('recipes.never_planned')}</span>
    </button>

    {#if !assigning && courseGroup}
      <button type="button" class="toggle" title={t('recipes.assign.hint')} onclick={startAssign}>
        {t('recipes.assign.button')}
      </button>
    {/if}

    <select bind:value={recipeFilter.sort} aria-label={t('recipes.sort.label')}>
      <option value="name">{t('recipes.sort.name')}</option>
      <option value="newest">{t('recipes.sort.newest')}</option>
      <option value="last_planned">{t('recipes.sort.last_planned')}</option>
      <option value="most_planned">{t('recipes.sort.most_planned')}</option>
    </select>
  </div>
</div>

{#if showTags}
  <div class="tag-panel">
    <TagFilter groups={tagGroups} selected={recipeFilter.tagIds} ontoggle={toggleTag} />
  </div>
{/if}

{#if assigning}
  <div class="assign-bar" role="region" aria-label={t('recipes.assign.button')}>
    <div class="assign-row">
      <label>
        <span>{t('recipes.assign.label')}</span>
        <select bind:value={target} aria-label={t('recipes.assign.target')}>
          {#if courseGroup}
            <optgroup label={t('recipes.assign.add_course')}>
              {#each courseGroup.tags as tag (tag.id)}<option value="add:{tag.id}">{t('recipes.assign.add_tag', { name: tag.name })}</option>{/each}
            </optgroup>
            <optgroup label={t('recipes.assign.remove_course')}>
              {#each courseGroup.tags as tag (tag.id)}<option value="remove:{tag.id}">{t('recipes.assign.remove_tag', { name: tag.name })}</option>{/each}
              <option value="remove:courses">{t('recipes.assign.remove_all_courses')}</option>
            </optgroup>
          {/if}
          {#each tagGroups.filter((g) => g.role !== 'course') as g (g.name)}
            <optgroup label={t('recipes.assign.add_category', { group: g.name })}>
              {#each g.tags as tag (tag.id)}<option value="add:{tag.id}">{t('recipes.assign.add_tag', { name: tag.name })}</option>{/each}
            </optgroup>
            <optgroup label={t('recipes.assign.remove_category', { group: g.name })}>
              {#each g.tags as tag (tag.id)}<option value="remove:{tag.id}">{t('recipes.assign.remove_tag', { name: tag.name })}</option>{/each}
            </optgroup>
          {/each}
        </select>
      </label>
      <strong>{t('recipes.assign.selected', { count: selected.size })}</strong>
    </div>
    <div class="assign-row">
      <button type="button" onclick={selectVisible} disabled={!recipes?.length}>{t('recipes.assign.select_visible')}</button>
      <button type="button" onclick={() => (selected = new Set())} disabled={selected.size === 0}>{t('recipes.assign.clear_selection')}</button>
      <button type="button" class="primary" onclick={apply} disabled={busy || selected.size === 0 || !target}>
        {busy ? t('common.saving') : t('common.apply')}
      </button>
      <button type="button" onclick={stopAssign}>{applied ? t('recipes.assign.done') : t('common.cancel')}</button>
    </div>
    {#if otherCourse > 0}
      <p class="assign-hint">
        {t('recipes.assign.other_course', { count: otherCourse, name: targetName })}
      </p>
    {/if}
  </div>
{/if}

{#if notice}
  <div class="notice" role="status">
    <span>{t(notice.key, notice.params)}</span>
    {#if notice.undo}<button type="button" class="link" onclick={undo} disabled={busy}>{t('recipes.notice.undo')}</button>{/if}
  </div>
{/if}

{#if error}
  <p class="message error">{t('recipes.load_error', { message: error })}</p>
{:else if recipes === null}
  <p class="message">{t('common.loading')}</p>
{:else if recipes.length === 0}
  {#if hasActiveFilter()}
    <div class="message">
      <p>{t('recipes.empty.none_found')}</p>
      <button type="button" class="link" onclick={resetFilter}>{t('recipes.empty.reset_filter')}</button>
    </div>
  {:else}
    <p class="message">{t('recipes.empty.no_recipes')}</p>
  {/if}
{:else}
  <p class="count">{t('recipes.count', { count: recipes.length })}</p>
  <div class="grid">
    {#each recipes as recipe (recipe.id)}
      <RecipeTile
        {recipe}
        courses={coursesOf(recipe)}
        onselect={assigning ? toggleSelect : null}
        selected={selected.has(recipe.id)}
      />
    {/each}
  </div>
{/if}

<style>
  .page-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .page-head h1 {
    margin: 0;
  }

  .head-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  .secondary {
    padding: 0.5rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    text-decoration: none;
    white-space: nowrap;
  }

  .new {
    padding: 0.5rem 1rem;
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
    text-decoration: none;
    white-space: nowrap;
  }

  .toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }

  .search {
    flex: 1 1 16rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text-muted);
  }

  .search:focus-within {
    border-color: var(--accent);
  }

  .search input {
    flex: 1;
    min-width: 0;
    padding: 0.55rem 0;
    border: none;
    outline: none;
    background: none;
    color: var(--text);
  }

  .controls {
    display: flex;
    flex-wrap: wrap; /* mit „Gang zuweisen“ auf dem Handy zweizeilig */
    flex: 0 1 auto;
    min-width: 0; /* sonst schrumpft die Gruppe nie unter ihre Inhaltsbreite */
    gap: 0.5rem;
  }

  .toggle,
  select {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.5rem 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    white-space: nowrap;
    cursor: pointer;
  }

  /* Die Sortierung darf auf schmalen Bildschirmen schrumpfen. */
  select {
    min-width: 0;
  }

  .toggle:hover {
    background: var(--surface-hover);
  }

  .toggle[aria-pressed='true'],
  .toggle[aria-expanded='true'] {
    border-color: var(--accent);
    color: var(--accent);
  }

  /* Auf schmalen Bildschirmen nur das Herz zeigen */
  .label-wide {
    display: none;
  }

  @media (min-width: 30rem) {
    .label-wide {
      display: inline;
    }
  }

  .tag-panel {
    margin-bottom: 1rem;
    padding: 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .count {
    margin: 0 0 0.5rem;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
    gap: 0.75rem;
  }

  .assign-bar {
    position: sticky;
    top: 0;
    z-index: 5;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
    padding: 0.75rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: 0 4px 12px rgb(0 0 0 / 0.08);
  }

  .assign-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  .assign-row label {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    flex: 1 1 14rem;
    min-width: 0;
  }

  .assign-row select {
    flex: 1;
    min-width: 0;
    max-width: 100%;
  }

  .assign-row button {
    padding: 0.45rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }

  .assign-row button.primary {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .assign-row button:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .assign-hint {
    margin: 0;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .notice {
    position: fixed;
    left: 50%;
    bottom: 1.5rem;
    z-index: 20;
    display: flex;
    align-items: center;
    gap: 1rem;
    max-width: calc(100vw - 2rem);
    padding: 0.7rem 1rem;
    border-radius: var(--radius);
    background: var(--text);
    color: var(--surface);
    transform: translateX(-50%);
    box-shadow: 0 4px 16px rgb(0 0 0 / 0.2);
  }

  .notice .link {
    color: var(--surface);
    font-weight: 600;
  }

  .message {
    margin: 2rem 0;
    text-align: center;
    color: var(--text-muted);
  }

  .message p {
    margin: 0 0 0.5rem;
  }

  .error {
    color: var(--danger);
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    text-decoration: underline;
    cursor: pointer;
  }
</style>
