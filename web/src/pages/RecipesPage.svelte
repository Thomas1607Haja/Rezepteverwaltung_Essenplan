<script>
  import { api } from '../lib/api.js';
  import { recipeFilter, toggleTag, resetFilter, hasActiveFilter } from '../lib/recipeFilter.svelte.js';
  import Icon from '../components/Icon.svelte';
  import RecipeTile from '../components/RecipeTile.svelte';
  import TagFilter from '../components/TagFilter.svelte';

  let recipes = $state(null); // null = noch nicht geladen
  let error = $state('');
  let tagGroups = $state([]);
  let showTags = $state(recipeFilter.tagIds.length > 0);

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
    } catch (e) {
      if (id !== requestCount) return;
      error = e.message;
    }
  }

  // $effect läuft beim Start und jedes Mal, wenn sich ein darin gelesener
  // Wert ändert – hier also bei jeder Änderung am Filter.
  $effect(() => {
    const filter = {
      query: recipeFilter.query,
      tagIds: [...recipeFilter.tagIds],
      favoritesOnly: recipeFilter.favoritesOnly,
      sort: recipeFilter.sort,
    };
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
</script>

<div class="page-head">
  <h1>Rezepte</h1>
  <a class="new" href="#/recipes/new">+ Neues Rezept</a>
</div>

<div class="toolbar">
  <label class="search">
    <Icon name="search" size={18} />
    <input
      type="search"
      placeholder="Suchen nach Titel oder Zutat"
      aria-label="Rezepte durchsuchen"
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
      Kategorien{activeTagCount > 0 ? ` (${activeTagCount})` : ''}
    </button>

    <button
      type="button"
      class="toggle"
      aria-pressed={recipeFilter.favoritesOnly}
      title="Nur Favoriten"
      onclick={() => (recipeFilter.favoritesOnly = !recipeFilter.favoritesOnly)}
    >
      <Icon name="heart" size={18} filled={recipeFilter.favoritesOnly} />
      <span class="label-wide">Favoriten</span>
    </button>

    <select bind:value={recipeFilter.sort} aria-label="Sortierung">
      <option value="name">Name</option>
      <option value="newest">Neu hinzugefügt</option>
      <option value="last_planned">Zuletzt eingeplant</option>
      <option value="most_planned">Am häufigsten eingeplant</option>
    </select>
  </div>
</div>

{#if showTags}
  <div class="tag-panel">
    <TagFilter groups={tagGroups} selected={recipeFilter.tagIds} ontoggle={toggleTag} />
  </div>
{/if}

{#if error}
  <p class="message error">Rezepte konnten nicht geladen werden: {error}</p>
{:else if recipes === null}
  <p class="message">Lädt …</p>
{:else if recipes.length === 0}
  {#if hasActiveFilter()}
    <div class="message">
      <p>Keine Rezepte gefunden.</p>
      <button type="button" class="link" onclick={resetFilter}>Filter zurücksetzen</button>
    </div>
  {:else}
    <p class="message">Noch keine Rezepte vorhanden.</p>
  {/if}
{:else}
  <p class="count">{recipes.length} {recipes.length === 1 ? 'Rezept' : 'Rezepte'}</p>
  <div class="grid">
    {#each recipes as recipe (recipe.id)}
      <RecipeTile {recipe} />
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
