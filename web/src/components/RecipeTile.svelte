<script>
  import Icon from './Icon.svelte';
  import { formatMinutes, planStatus } from '../lib/format.js';
  import { imageUrl } from '../lib/api.js';

  // recipe ist eine Kurzfassung aus der Rezeptliste.
  //   courses   Namen der Gänge des Rezepts (Etikett auf der Kachel)
  //   onselect  gesetzt = Zuweisungsmodus: Klick wählt aus statt zu öffnen
  //   selected  im Zuweisungsmodus ausgewählt
  let { recipe, courses = [], onselect = null, selected = false } = $props();

  const minutes = $derived(recipe.total_minutes ?? recipe.prep_minutes);
</script>

{#snippet content()}
  <div class="image">
    {#if recipe.thumbnail}
      <!-- loading="lazy": Bilder weiter unten lädt der Browser erst beim Scrollen. -->
      <img src={imageUrl(recipe.thumbnail)} alt="" loading="lazy" />
      {#if recipe.image_kind === 'illustration'}
        <span class="drawn" title="Gezeichnetes Bild – durch ein Foto ersetzen">Illustration</span>
      {/if}
    {:else}
      <Icon name="dish" size={40} />
    {/if}
  </div>
  <div class="body">
    {#if courses.length > 0}
      <span class="courses">{courses.join(' · ')}</span>
    {:else if onselect}
      <span class="courses none">ohne Gang</span>
    {/if}
    <h2 class="title">{recipe.title}</h2>
    {#if recipe.parent_recipe_id != null}<span class="variant-badge">Variante</span>{/if}
    <p class="status" class:upcoming={recipe.next_planned}>{planStatus(recipe)}</p>
    <div class="meta">
      {#if minutes != null}
        <span class="time"><Icon name="clock" size={15} /> {formatMinutes(minutes)}</span>
      {/if}
      {#if recipe.is_favorite}
        <span class="favorite" title="Favorit"><Icon name="heart" size={16} filled /></span>
      {/if}
    </div>
  </div>
{/snippet}

{#if onselect}
  <!-- div statt button: Überschrift und Blöcke sind in einem button nicht erlaubt. -->
  <div
    class="tile selectable"
    class:selected
    role="checkbox"
    aria-checked={selected}
    tabindex="0"
    onclick={() => onselect(recipe)}
    onkeydown={(e) => {
      if (e.key === ' ' || e.key === 'Enter') {
        e.preventDefault();
        onselect(recipe);
      }
    }}
  >
    <span class="check" aria-hidden="true">{selected ? '✓' : ''}</span>
    {@render content()}
  </div>
{:else}
  <a class="tile" href={'#/recipes/' + recipe.id}>{@render content()}</a>
{/if}

<style>
  .tile {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: inherit;
    text-decoration: none;
    transition: transform 0.1s, box-shadow 0.1s;
  }

  .tile.selectable {
    position: relative;
    cursor: pointer;
    user-select: none;
  }

  .tile.selected {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--accent);
  }

  .check {
    position: absolute;
    z-index: 1;
    top: 0.5rem;
    right: 0.5rem;
    display: grid;
    place-items: center;
    width: 1.6rem;
    height: 1.6rem;
    border: 2px solid var(--accent);
    border-radius: 0.35rem;
    background: var(--surface);
    color: var(--accent);
    font-weight: 700;
  }

  .tile.selected .check {
    background: var(--accent);
    color: var(--on-accent);
  }

  .courses {
    align-self: flex-start;
    padding: 0.05rem 0.45rem;
    border-radius: 999px;
    background: var(--accent-soft);
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 600;
  }

  .courses.none {
    background: var(--surface-hover);
    color: var(--text-muted);
    font-weight: 400;
  }

  .tile:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgb(0 0 0 / 0.08);
  }

  .image {
    position: relative; /* für das Kennzeichen „Illustration“ */
    display: grid;
    place-items: center;
    aspect-ratio: 4 / 3;
    overflow: hidden;
    background: var(--accent-soft);
    color: var(--accent);
  }

  /* object-fit: cover füllt die Fläche und schneidet überstehende Ränder ab. */
  .image img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .body {
    display: flex;
    flex-direction: column;
    flex: 1;
    gap: 0.35rem;
    padding: 0.6rem 0.75rem 0.7rem;
  }

  .title {
    margin: 0;
    font-size: 0.95rem;
    font-weight: 600;
    line-height: 1.3;
    /* Lange Titel nach zwei Zeilen abschneiden */
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .status {
    margin: 0;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .status.upcoming {
    color: var(--accent);
  }

  .meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: auto;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .time {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
  }

  .variant-badge {
    align-self: flex-start;
    margin-bottom: 0.25rem;
    padding: 0.05rem 0.45rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .drawn {
    position: absolute;
    left: 0.5rem;
    bottom: 0.5rem;
    padding: 0.05rem 0.45rem;
    border-radius: 999px;
    background: rgb(255 255 255 / 0.85);
    font-size: 0.72rem;
    color: #444;
  }

  .favorite {
    display: inline-flex;
    margin-left: auto;
    color: var(--accent);
  }
</style>
