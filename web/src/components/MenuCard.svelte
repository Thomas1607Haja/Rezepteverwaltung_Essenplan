<script>
  import Icon from './Icon.svelte';
  import { amountText } from '../lib/format.js';

  // Ein Menü im Plan (Schritt 9.4): eine kompakte Karte statt vieler
  // Einträge, ausklappbar. menu kommt aus GET /meal-plan, entries sind seine
  // Gänge in Reihenfolge, courseName(id) liefert den Namen eines Gangs.
  // onopen(menu) öffnet das Menü des Menüs. dropTarget hebt die Karte beim
  // Ziehen hervor.
  let { menu, entries, courseName, onopen, dropTarget = false } = $props();

  let open = $state(false);
  const transferred = $derived(entries.filter((e) => e.status === 'transferred').length);
  const have = $derived(entries.filter((e) => e.status === 'have_ingredients').length);
  const withRecipe = $derived(entries.filter((e) => e.recipe_id).length);
</script>

<!-- data-*: zieht als ganzer Block (lib/planDrag.js) -->
<div class="menu-card" class:drop-target={dropTarget} data-menu-id={menu.id} data-date={menu.date} data-slot={menu.slot}>
  <button type="button" class="main" onclick={() => onopen(menu)}>
    <span class="badge">Menü</span>
    <span class="title">{menu.display_title}</span>
    <span class="meta">
      <span>{entries.length === 1 ? '1 Gang' : `${entries.length} Gänge`}</span>
      <span>{amountText(menu.servings, '')}</span>
      {#if menu.note}<span title={menu.note}><Icon name="clock" size={13} /> Notiz</span>{/if}
      {#if withRecipe > 0 && transferred === withRecipe}
        <span class="status" title="An Bring übertragen"><Icon name="cart" size={14} /></span>
      {:else if transferred > 0}
        <span>{transferred} von {withRecipe} an Bring</span>
      {:else if withRecipe > 0 && have === withRecipe}
        <span class="status" title="Zutaten vorhanden"><Icon name="check" size={14} /></span>
      {/if}
    </span>
  </button>
  {#if entries.length > 0}
    <button type="button" class="toggle" aria-expanded={open} onclick={() => (open = !open)}>
      {open ? 'Gänge ausblenden' : 'Gänge zeigen'}
    </button>
  {/if}
  {#if open}
    <ul class="courses">
      {#each entries as e (e.id)}
        <li>
          {#if e.course_tag_id && courseName(e.course_tag_id)}<small>{courseName(e.course_tag_id)}</small>{/if}
          {#if e.recipe_id}
            <a href="#/recipes/{e.recipe_id}/entry/{e.id}" draggable="false">{e.recipe?.title ?? e.note}</a>
          {:else}
            <span>{e.note}</span>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .menu-card {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--surface);
    overflow: hidden;
    touch-action: manipulation;
  }

  .menu-card.drop-target {
    outline: 2px dashed var(--accent);
    outline-offset: 2px;
  }

  .main {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.15rem;
    padding: 0.5rem 0.6rem;
    border: none;
    background: var(--accent-soft);
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }

  .badge {
    font-size: 0.65rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--accent);
  }

  .title {
    font-weight: 600;
    line-height: 1.25;
  }

  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .status {
    display: inline-flex;
    color: var(--accent);
  }

  .toggle {
    padding: 0.2rem 0.6rem;
    border: none;
    border-top: 1px solid var(--border);
    background: var(--surface);
    color: var(--text-muted);
    font: inherit;
    font-size: 0.75rem;
    text-align: left;
    cursor: pointer;
  }

  .courses {
    margin: 0;
    padding: 0.3rem 0.6rem 0.5rem;
    list-style: none;
    font-size: 0.85rem;
  }

  .courses li {
    display: flex;
    flex-direction: column;
    padding: 0.15rem 0;
  }

  .courses small {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
</style>
