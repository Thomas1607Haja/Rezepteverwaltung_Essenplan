<script>
  import { api, imageUrl } from '../lib/api.js';
  import { dayLabel, amountText } from '../lib/format.js';
  import { slotLabel } from '../lib/slots.js';
  import { focusPlanOn } from '../lib/planFocus.svelte.js';
  import { navigate } from '../lib/router.svelte.js';

  // Kochansicht eines Menüs (Schritt 9.4): Notiz, Personen und alle Gänge
  // nacheinander; jeder Gang öffnet seine Kochansicht mit den Portionen des
  // Eintrags.
  let { id } = $props();

  let menu = $state(null);
  let courses = $state([]);
  let error = $state('');

  // svelte-ignore state_referenced_locally
  Promise.all([api.getMenu(id), api.listTagGroups()])
    .then(([m, groups]) => {
      menu = m;
      courses = groups.find((g) => g.role === 'course')?.tags ?? [];
      document.title = `Menü: ${m.display_title} – Rezeptplaner`;
    })
    .catch((e) => (error = e.status === 404 ? 'Dieses Menü gibt es nicht mehr.' : e.message));

  const courseName = (cid) => courses.find((c) => c.id === cid)?.name ?? '';

  function back() {
    if (menu) focusPlanOn(menu.date);
    navigate('/');
  }
</script>

<p class="back"><button type="button" class="link" onclick={back}>← Essensplan</button></p>

{#if error}
  <p class="error">{error}</p>
{:else if !menu}
  <p>Lädt …</p>
{:else}
  {@const l = dayLabel(menu.date)}
  <p class="badge">Menü</p>
  <h1>{menu.display_title}</h1>
  <p class="muted">{l.weekday} {l.date} · {slotLabel(menu.slot)} · {menu.servings === 1 ? '1 Person' : `${menu.servings} Personen`}</p>
  {#if menu.note}<p class="note">{menu.note}</p>{/if}

  <ol class="courses">
    {#each menu.entries as e (e.id)}
      <li>
        {#if e.recipe?.thumbnail}<img src={imageUrl(e.recipe.thumbnail)} alt="" />{:else}<span class="noimg"></span>{/if}
        <span class="text">
          {#if e.course_tag_id && courseName(e.course_tag_id)}<small>{courseName(e.course_tag_id)}</small>{/if}
          {#if e.recipe_id}
            <a href="#/recipes/{e.recipe_id}/entry/{e.id}">{e.recipe?.title ?? e.note}</a>
            <span class="muted">{amountText(e.servings, e.recipe?.yield_unit)}{e.note ? ` · ${e.note}` : ''}</span>
          {:else}
            <strong>{e.note}</strong>
          {/if}
        </span>
      </li>
    {:else}
      <li class="muted">Noch keine Gänge – im Essensplan über das Menü hinzufügen.</li>
    {/each}
  </ol>
{/if}

<style>
  .badge {
    margin: 0;
    color: var(--accent);
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  h1 {
    margin: 0.1rem 0 0.25rem;
  }

  .muted {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .note {
    margin: 0.75rem 0;
    padding: 0.6rem 0.8rem;
    border-left: 3px solid var(--accent);
    background: var(--accent-soft);
    border-radius: 0 var(--radius) var(--radius) 0;
    white-space: pre-line;
  }

  .error {
    color: var(--danger);
  }

  .courses {
    margin: 1rem 0;
    padding: 0;
    list-style: none;
  }

  .courses li {
    display: grid;
    grid-template-columns: 4rem minmax(0, 1fr);
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 0;
    border-bottom: 1px solid var(--border);
  }

  img,
  .noimg {
    width: 4rem;
    height: 4rem;
    object-fit: cover;
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface-hover);
  }

  .text {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .text small {
    color: var(--accent);
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .text a {
    font-weight: 600;
    font-size: 1.05rem;
  }
</style>
