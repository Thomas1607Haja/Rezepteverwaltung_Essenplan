<script>
  import Icon from './Icon.svelte';
  import { imageUrl } from '../lib/api.js';

  // entry ist ein Eintrag aus GET /meal-plan; onmenu(entry) öffnet das
  // Menü (Portionen, Notiz, Ersetzen, Entfernen). dropTarget hebt den
  // Eintrag hervor, wenn ein anderer zum Tauschen darüber gezogen wird.
  let { entry, onmenu, dropTarget = false } = $props();

  const title = $derived(entry.recipe ? entry.recipe.title : entry.note);
  // Bei einem Rezept ist die Notiz eine Zusatzangabe (z. B. „mit Salat“).
  const extra = $derived(entry.recipe ? entry.note : '');

  const statusInfo = {
    transferred: { icon: 'cart', label: 'An Bring übertragen' },
    have_ingredients: { icon: 'check', label: 'Zutaten vorhanden' },
  };
  const status = $derived(statusInfo[entry.status]);
</script>

<!-- Einträge mit Rezept öffnen die Kochansicht (mit den Portionen des
     Eintrags), Einträge ohne Rezept das Menü. Der Menüknopf liegt neben
     dem Link, denn ein Knopf darf nicht in einem Link stecken. -->
<!-- data-*: Angaben für das Ziehen (lib/planDrag.js) -->
<div
  class="entry"
  class:no-recipe={!entry.recipe_id}
  class:drop-target={dropTarget}
  data-entry-id={entry.id}
  data-date={entry.date}
  data-slot={entry.slot}
>
  {#snippet content()}
    {#if entry.recipe}
      <span class="thumb">
        {#if entry.recipe.thumbnail}
          <img src={imageUrl(entry.recipe.thumbnail)} alt="" loading="lazy" draggable="false" />
        {:else}
          <Icon name="dish" size={22} />
        {/if}
      </span>
    {/if}
    <span class="body">
      <span class="title">{title}</span>
      {#if extra}<span class="extra">{extra}</span>{/if}
      <span class="meta">
        {#if entry.recipe}<span>{entry.servings} P.</span>{/if}
        {#if entry.recipe?.is_favorite}
          <span class="favorite" title="Favorit"><Icon name="heart" size={13} filled /></span>
        {/if}
        {#if status}
          <span class="status" title={status.label}><Icon name={status.icon} size={14} /></span>
        {/if}
      </span>
    </span>
  {/snippet}
  {#if entry.recipe_id}
    <a class="main" href="#/recipes/{entry.recipe_id}/entry/{entry.id}" draggable="false">{@render content()}</a>
  {:else}
    <button type="button" class="main" onclick={() => onmenu(entry)}>{@render content()}</button>
  {/if}
  <button type="button" class="menu" title="Menü" aria-label="Menü für {title}" onclick={() => onmenu(entry)}>
    <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <circle cx="12" cy="5" r="1.8" /><circle cx="12" cy="12" r="1.8" /><circle cx="12" cy="19" r="1.8" />
    </svg>
  </button>
</div>

<style>
  .entry {
    position: relative;
    overflow: hidden;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.15rem);
  }

  .entry:hover {
    border-color: var(--accent);
  }

  /* Beim langen Drücken auf dem Handy keinen Text markieren und kein
     Link-Menü öffnen */
  .entry {
    user-select: none;
    -webkit-user-select: none;
    -webkit-touch-callout: none;
  }

  .drop-target {
    outline: 3px solid var(--accent);
    outline-offset: 1px;
  }

  /* Original, solange eine Kopie davon gezogen wird (Klasse aus planDrag.js) */
  :global(.entry.drag-source) {
    opacity: 0.35;
  }

  .main {
    display: flex;
    flex-direction: column;
    width: 100%;
    padding: 0;
    border: none;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    text-decoration: none;
    cursor: pointer;
  }

  /* Rechts oben, auf dem Bild bzw. neben dem Titel */
  .menu {
    position: absolute;
    top: 0.2rem;
    right: 0.2rem;
    display: grid;
    place-items: center;
    width: 1.8rem;
    height: 1.8rem;
    border: none;
    border-radius: 50%;
    background: color-mix(in srgb, var(--surface) 85%, transparent);
    color: var(--text);
    cursor: pointer;
  }

  .menu:hover {
    background: var(--surface-hover);
  }

  .no-recipe {
    border-style: dashed;
    background: transparent;
  }

  .thumb {
    display: grid;
    place-items: center;
    aspect-ratio: 16 / 9;
    overflow: hidden;
    background: var(--accent-soft);
    color: var(--accent);
  }

  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .body {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    padding: 0.35rem 0.5rem 0.4rem;
  }

  /* Ohne Bild steht der Menüknopf neben dem Titel */
  .no-recipe .body {
    padding-right: 2rem;
  }

  .title {
    font-size: 0.85rem;
    font-weight: 600;
    line-height: 1.25;
    overflow-wrap: anywhere;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .extra {
    font-size: 0.78rem;
    color: var(--text-muted);
  }

  .meta {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .favorite,
  .status {
    display: inline-flex;
    color: var(--accent);
  }
</style>
