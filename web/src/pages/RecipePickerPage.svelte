<script>
  import { api, imageUrl } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { focusPlanOn } from '../lib/planFocus.svelte.js';
  import { dayLabel, planStatus } from '../lib/format.js';
  import Icon from '../components/Icon.svelte';
  import TagFilter from '../components/TagFilter.svelte';

  // Zwei Arten, die Seite zu öffnen:
  //   #/plan/add/2026-10-24     neues Gericht für diesen Tag
  //   #/plan/replace/17         Rezept von Eintrag 17 ersetzen
  let { date = null, entryId = null } = $props();
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
  let recipes = $state(null);
  let note = $state('');

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
    const filter = { query, favoritesOnly, tagIds: [...tagIds], sort };
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

  /** Trägt ein Rezept (recipeId) bzw. eine Notiz ein und geht zurück zum Plan. */
  async function choose(recipeId, noteText = '') {
    if (saving) return;
    saving = true;
    error = '';
    try {
      if (replaceId) {
        await api.updateEntry(replaceId, {
          recipe_id: recipeId,
          servings: entry.servings,
          // Ohne bisheriges Rezept war die Notiz der Titel – sie entfällt
          // dann. Eine Zusatznotiz zum Rezept („mit Salat“) bleibt.
          note: recipeId === null ? noteText : entry.recipe_id ? entry.note : '',
          // „Zutaten vorhanden“ galt für das alte Gericht.
          status: entry.status === 'have_ingredients' ? 'planned' : entry.status,
        });
      } else {
        await api.createEntry({ date: targetDate, slot, recipe_id: recipeId, note: noteText });
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
    choose(null, note.trim());
  }

  function back() {
    focusPlanOn(targetDate);
    navigate('/');
  }
</script>

<p class="back"><button type="button" class="link" onclick={back}>← Essensplan</button></p>

<h1>
  {#if replaceId}
    Ersetzen{#if entry}: {entry.recipe?.title ?? entry.note}{/if}
  {:else}
    Gericht für {heading}
  {/if}
</h1>

{#if error}<p class="error">{error}</p>{/if}

{#if !replaceId}
  <div class="slot-switch" role="group" aria-label="Mahlzeit">
    <button type="button" aria-pressed={slot === 'dinner'} onclick={() => (slot = 'dinner')}>Abendessen</button>
    <button type="button" aria-pressed={slot === 'lunch'} onclick={() => (slot = 'lunch')}>Mittagessen</button>
  </div>
{/if}

<form class="note-form" onsubmit={chooseNote}>
  <input placeholder="Ohne Rezept, z. B. Essen gehen" aria-label="Eintrag ohne Rezept" bind:value={note} maxlength="200" />
  <button type="submit" class="secondary" disabled={note.trim() === '' || saving}>Eintragen</button>
</form>
<div class="quick">
  {#each ['Essen gehen', 'Reste', 'Pizza bestellen'] as text (text)}
    <button type="button" class="chip" disabled={saving} onclick={() => choose(null, text)}>{text}</button>
  {/each}
</div>

<div class="toolbar">
  <label class="search">
    <Icon name="search" size={18} />
    <input type="search" placeholder="Rezept suchen" aria-label="Rezepte durchsuchen" bind:value={query} />
  </label>
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
        <button type="button" class="row" disabled={saving} onclick={() => choose(recipe.id)}>
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

<style>
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
    overflow-wrap: anywhere;
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

  .toolbar {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }

  .search {
    position: relative;
    flex: 1;
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
    overflow-wrap: anywhere;
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
