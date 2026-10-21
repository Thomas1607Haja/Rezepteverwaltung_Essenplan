<script>
  import Icon from './Icon.svelte';
  import { imageUrl } from '../lib/api.js';

  // entry ist ein Eintrag aus GET /meal-plan.
  let { entry } = $props();

  const title = $derived(entry.recipe ? entry.recipe.title : entry.note);
  // Bei einem Rezept ist die Notiz eine Zusatzangabe (z. B. „mit Salat“).
  const extra = $derived(entry.recipe ? entry.note : '');

  const statusInfo = {
    transferred: { icon: 'cart', label: 'An Bring übertragen' },
    have_ingredients: { icon: 'check', label: 'Zutaten vorhanden' },
  };
  const status = $derived(statusInfo[entry.status]);
</script>

<!-- Einträge mit Rezept öffnen das Rezept; ohne Rezept gibt es nichts zu öffnen. -->
<svelte:element
  this={entry.recipe_id ? 'a' : 'div'}
  class="entry"
  class:no-recipe={!entry.recipe_id}
  href={entry.recipe_id ? `#/recipes/${entry.recipe_id}` : undefined}
>
  {#if entry.recipe}
    <div class="thumb">
      {#if entry.recipe.thumbnail}
        <img src={imageUrl(entry.recipe.thumbnail)} alt="" loading="lazy" />
      {:else}
        <Icon name="dish" size={22} />
      {/if}
    </div>
  {/if}
  <div class="body">
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
  </div>
</svelte:element>

<style>
  .entry {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.15rem);
    color: inherit;
    text-decoration: none;
  }

  a.entry:hover {
    border-color: var(--accent);
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

  .meta:empty {
    display: none;
  }

  .favorite,
  .status {
    display: inline-flex;
    color: var(--accent);
  }
</style>
