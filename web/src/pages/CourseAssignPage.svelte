<script>
  import { api, imageUrl } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { assignPreset, resetFilter, recipeFilter } from '../lib/recipeFilter.svelte.js';

  // Gänge zuordnen (Schritt 9.1): Der Server schlägt für jedes Rezept ohne
  // Gang einen vor. Häkchen und Auswahl prüfen, dann gesammelt übernehmen.
  let suggestions = $state(null);
  let courses = $state([]); // Gänge der Gang-Gruppe zur Auswahl
  let chosen = $state({}); // recipe_id → tag_id (0 = nicht übernehmen)
  let error = $state('');
  let message = $state('');
  let busy = $state(false);

  async function load() {
    try {
      const [list, groups] = await Promise.all([api.courseSuggestions(), api.listTagGroups()]);
      courses = groups.find((g) => g.role === 'course')?.tags ?? [];
      chosen = Object.fromEntries(list.map((s) => [s.recipe_id, s.tag_id]));
      suggestions = list;
      error = '';
    } catch (e) {
      error = e.message;
    }
  }
  load();

  const count = $derived(Object.values(chosen).filter((id) => id > 0).length);

  // Vorschläge je Gang, für „Vorschläge für … prüfen“.
  const byCourse = $derived(
    courses
      .map((c) => ({ ...c, ids: (suggestions ?? []).filter((s) => s.tag_id === c.id).map((s) => s.recipe_id) }))
      .filter((c) => c.ids.length > 0),
  );

  /** Zuweisungsmodus der Rezeptübersicht mit den Vorschlägen als Auswahl öffnen. */
  function review(course) {
    resetFilter();
    recipeFilter.course = 'none';
    assignPreset.tagId = course.id;
    assignPreset.ids = course.ids;
    navigate('/recipes');
  }

  async function apply() {
    busy = true;
    message = '';
    try {
      const assignments = Object.entries(chosen)
        .filter(([, tagId]) => tagId > 0)
        .map(([recipeId, tagId]) => ({ recipe_id: Number(recipeId), tag_id: Number(tagId) }));
      const result = await api.assignCourses(assignments);
      message = `${result.assigned} ${result.assigned === 1 ? 'Rezept' : 'Rezepte'} zugeordnet.`;
      await load();
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<p class="back"><a href="#/settings">← Einstellungen</a></p>
<h1>Gänge zuordnen</h1>
<p class="hint">
  Vorschläge für alle Rezepte ohne Gang – nach Wörtern im Titel, den Zutaten und vorhandenen Kategorien. Bitte prüfen;
  „nicht zuordnen“ lässt ein Rezept aus. Weitere Gänge (z. B. Salat auch als Beilage) lassen sich im Rezept ergänzen.
</p>

{#if error}<p class="error">{error}</p>{/if}
{#if message}<p class="ok">{message}</p>{/if}

{#if byCourse.length > 0}
  <div class="review">
    <p class="hint">
      Oder in der Rezeptübersicht mit Bildern prüfen – die Vorschläge sind dort schon ausgewählt (Zuweisungsmodus):
    </p>
    {#each byCourse as c (c.id)}
      <button type="button" onclick={() => review(c)}>Vorschläge für {c.name} prüfen ({c.ids.length})</button>
    {/each}
  </div>
{/if}

{#if suggestions === null && !error}
  <p>Lädt …</p>
{:else if suggestions?.length === 0}
  <p>Alle Rezepte haben einen Gang.</p>
{:else if suggestions}
  <ul class="list">
    {#each suggestions as s (s.recipe_id)}
      <li>
        {#if s.thumbnail}<img src={imageUrl(s.thumbnail)} alt="" />{:else}<span class="noimg"></span>{/if}
        <span class="text">
          <a href="#/recipes/{s.recipe_id}">{s.title}</a>
          <small>{s.reason}</small>
        </span>
        <select bind:value={chosen[s.recipe_id]} aria-label="Gang für {s.title}">
          <option value={0}>nicht zuordnen</option>
          {#each courses as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
        </select>
      </li>
    {/each}
  </ul>
  <div class="actions">
    <button type="button" class="primary" disabled={busy || count === 0} onclick={apply}>
      {busy ? 'Speichert …' : `${count} ${count === 1 ? 'Zuordnung' : 'Zuordnungen'} übernehmen`}
    </button>
  </div>
{/if}

<style>
  .review {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .review .hint {
    flex-basis: 100%;
    margin: 0;
  }

  .review button {
    padding: 0.4rem 0.8rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--accent);
    font: inherit;
    cursor: pointer;
  }

  .hint {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .error {
    color: var(--danger);
  }

  .ok {
    color: var(--accent);
  }

  .list {
    margin: 1rem 0;
    padding: 0;
    list-style: none;
    border-top: 1px solid var(--border);
  }

  .list li {
    display: grid;
    grid-template-columns: 2.5rem minmax(0, 1fr) minmax(9rem, 12rem);
    align-items: center;
    gap: 0.6rem;
    padding: 0.4rem 0;
    border-bottom: 1px solid var(--border);
  }

  img,
  .noimg {
    width: 2.5rem;
    height: 2.5rem;
    object-fit: cover;
    border-radius: 0.3rem;
    background: var(--surface-hover);
  }

  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .text a {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .text small {
    color: var(--text-muted);
  }

  select {
    padding: 0.35rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
  }

  .primary {
    padding: 0.5rem 1rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--on-accent);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }

  .primary:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
