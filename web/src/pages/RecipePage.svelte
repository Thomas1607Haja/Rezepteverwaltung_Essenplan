<script>
  import { onMount } from 'svelte';
  import { api, imageUrl } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { filterByTag } from '../lib/recipeFilter.svelte.js';
  import { dayLabel, formatMinutes, formatQuantity, planStatus } from '../lib/format.js';
  import { keepScreenOn } from '../lib/wakeLock.js';
  import { focusPlanOn } from '../lib/planFocus.svelte.js';
  import Icon from '../components/Icon.svelte';
  import AddToPlanDialog from '../components/AddToPlanDialog.svelte';

  // id kommt aus der Adresse #/recipes/:id. Aus dem Essensplan geöffnet
  // (#/recipes/:id/entry/:entryId) kommt die ID des Planeintrags dazu.
  let { id, entryId = null } = $props();
  // App.svelte baut die Seite bei jeder neuen Adresse neu auf ({#key}),
  // id ändert sich also nie. Svelte warnt sonst, dass nur der Anfangswert
  // gelesen wird – genau das ist hier gewollt.
  // svelte-ignore state_referenced_locally
  const recipeId = id;
  // svelte-ignore state_referenced_locally
  const planEntryId = entryId;

  let recipe = $state(null);
  let error = $state('');
  let notFound = $state(false);
  let servings = $state(1);
  let planEntry = $state(null); // Planeintrag, aus dem die Seite geöffnet wurde
  let showAddDialog = $state(false);

  // Rezept und ggf. Planeintrag gleichzeitig laden. Aus dem Plan gelten
  // dessen Portionen, sonst die Grundportionen des Rezepts.
  Promise.all([api.getRecipe(recipeId), planEntryId ? api.getEntry(planEntryId).catch(() => null) : null])
    .then(([r, entry]) => {
      recipe = r;
      planEntry = entry?.recipe_id === r.id ? entry : null;
      servings = planEntry ? planEntry.servings : r.base_servings;
      document.title = `${r.title} – Rezeptplaner`;
    })
    .catch((e) => {
      if (e.status === 404 || e.status === 400) notFound = true;
      else error = e.message;
    });

  /** Nach dem Einplanen den Statushinweis auffrischen. */
  async function refreshPlanInfo() {
    try {
      const r = await api.getRecipe(recipeId);
      recipe.next_planned = r.next_planned;
      recipe.last_planned = r.last_planned;
      recipe.plan_count = r.plan_count;
    } catch {
      // nicht schlimm: Der Hinweis stimmt dann erst nach dem Neuladen
    }
  }

  function backToPlan() {
    focusPlanOn(planEntry.date);
    navigate('/');
  }

  // onMount läuft, sobald die Seite angezeigt wird. Die zurückgegebene
  // Funktion läuft beim Verlassen: Sperre freigeben, Titel zurücksetzen.
  onMount(() => {
    const release = keepScreenOn();
    return () => {
      release();
      document.title = 'Rezeptplaner';
    };
  });

  // Faktor für die Umrechnung, z. B. 6 statt 4 Portionen -> 1,5.
  const factor = $derived(recipe ? servings / recipe.base_servings : 1);

  // Zutaten nach Gruppen: Aufeinanderfolgende Zutaten mit demselben
  // group_name bilden eine Gruppe. Leerer Name = ohne Überschrift.
  const ingredientGroups = $derived.by(() => {
    const groups = [];
    for (const ing of recipe?.ingredients ?? []) {
      const last = groups.at(-1);
      if (last && last.name === ing.group_name) last.items.push(ing);
      else groups.push({ name: ing.group_name, items: [ing] });
    }
    return groups;
  });

  /** Hostname der Quelle für die Anzeige, z. B. "www.chefkoch.de". */
  function sourceHost(url) {
    try {
      return new URL(url).hostname;
    } catch {
      return url;
    }
  }

  // Nur Webadressen als Link: "javascript:…" würde beim Anklicken im
  // Browser ausgeführt – und die Oberfläche läuft in der Sitzung von Home
  // Assistant. Der Server lehnt solche Adressen ohnehin ab.
  function isWebUrl(url) {
    try {
      return ['http:', 'https:'].includes(new URL(url).protocol);
    } catch {
      return false;
    }
  }

  async function toggleFavorite() {
    const next = !recipe.is_favorite;
    recipe.is_favorite = next; // sofort anzeigen, bei Fehler zurücknehmen
    try {
      await api.setFavorite(recipe.id, next);
    } catch (e) {
      recipe.is_favorite = !next;
      alert(`Favorit konnte nicht gespeichert werden: ${e.message}`);
    }
  }

  async function remove() {
    let question = `„${recipe.title}“ wirklich löschen?`;
    if (recipe.next_planned) {
      question += '\n\nEs ist noch im Essensplan eingeplant. Dort bleibt der Titel als Notiz stehen.';
    }
    if (!confirm(question)) return;
    try {
      await api.deleteRecipe(recipe.id);
      navigate('/recipes');
    } catch (e) {
      alert(`Löschen fehlgeschlagen: ${e.message}`);
    }
  }

  function showTag(tagId) {
    filterByTag(tagId);
    navigate('/recipes');
  }
</script>

<p class="back">
  {#if planEntry}
    <button type="button" class="link back-link" onclick={backToPlan}>← Essensplan</button>
  {:else}
    <a href="#/recipes">← Rezepte</a>
  {/if}
</p>

{#if notFound}
  <h1>Rezept nicht gefunden</h1>
  <p>Dieses Rezept gibt es nicht (mehr).</p>
{:else if error}
  <p class="error">Rezept konnte nicht geladen werden: {error}</p>
{:else if !recipe}
  <p class="muted">Lädt …</p>
{:else}
  <article>
    {#if recipe.image}
      <img class="photo" src={imageUrl(recipe.image)} alt={recipe.title} />
    {:else}
      <div class="hero">
        <Icon name="dish" size={64} />
      </div>
    {/if}

    <header class="head">
      <div class="title-row">
        <h1>{recipe.title}</h1>
        <button
          type="button"
          class="icon-button favorite"
          aria-pressed={recipe.is_favorite}
          title={recipe.is_favorite ? 'Favorit entfernen' : 'Als Favorit markieren'}
          onclick={toggleFavorite}
        >
          <Icon name="heart" size={26} filled={recipe.is_favorite} />
        </button>
      </div>

      {#if planEntry}
        {@const l = dayLabel(planEntry.date)}
        <p class="from-plan">
          <Icon name="calendar" size={16} />
          Im Essensplan: {l.weekday} {l.date}, {planEntry.servings}
          {planEntry.servings === 1 ? 'Portion' : 'Portionen'}{planEntry.note ? ` · ${planEntry.note}` : ''}
        </p>
      {/if}

      {#if recipe.description}
        <p class="description">{recipe.description}</p>
      {/if}

      <div class="facts">
        <span class:upcoming={recipe.next_planned}><Icon name="calendar" size={16} /> {planStatus(recipe)}</span>
        {#if recipe.prep_minutes != null}
          <span><Icon name="clock" size={16} /> Vorbereitung {formatMinutes(recipe.prep_minutes)}</span>
        {/if}
        {#if recipe.total_minutes != null}
          <span><Icon name="clock" size={16} /> Gesamt {formatMinutes(recipe.total_minutes)}</span>
        {/if}
        {#if recipe.source_url}
          <span>
            Quelle:
            {#if isWebUrl(recipe.source_url)}
              <a href={recipe.source_url} target="_blank" rel="noopener noreferrer">{sourceHost(recipe.source_url)}</a>
            {:else}
              {recipe.source_url}
            {/if}
          </span>
        {/if}
      </div>

      {#if recipe.tags.length > 0}
        <div class="tags">
          {#each recipe.tags as tag (tag.id)}
            <button type="button" class="tag" title="Rezepte mit „{tag.name}“ zeigen" onclick={() => showTag(tag.id)}>
              {tag.name}
            </button>
          {/each}
        </div>
      {/if}
    </header>

    <div class="columns">
      <section class="ingredients">
        <div class="section-head">
          <h2>Zutaten</h2>
          <div class="servings" role="group" aria-label="Portionen">
            <button
              type="button"
              class="icon-button"
              aria-label="Eine Portion weniger"
              disabled={servings <= 1}
              onclick={() => servings--}>−</button
            >
            <span class="servings-value">
              {servings}
              {servings === 1 ? 'Portion' : 'Portionen'}
            </span>
            <button type="button" class="icon-button" aria-label="Eine Portion mehr" onclick={() => servings++}>+</button>
          </div>
        </div>
        {#if servings !== recipe.base_servings}
          <p class="hint">
            Umgerechnet von {recipe.base_servings}
            {recipe.base_servings === 1 ? 'Portion' : 'Portionen'} ·
            <button type="button" class="link" onclick={() => (servings = recipe.base_servings)}>zurücksetzen</button>
          </p>
        {/if}

        {#if recipe.ingredients.length === 0}
          <p class="muted">Keine Zutaten erfasst.</p>
        {/if}
        {#each ingredientGroups as group, i (i)}
          {#if group.name}
            <h3>{group.name}</h3>
          {/if}
          <ul>
            {#each group.items as ing (ing.id)}
              <li>
                <span class="amount">
                  {#if ing.quantity != null}{formatQuantity(ing.quantity * factor)}{/if}
                  {ing.unit}
                </span>
                <span class="name">
                  {ing.name}{#if ing.note}<span class="note">, {ing.note}</span>{/if}
                </span>
              </li>
            {/each}
          </ul>
        {/each}
      </section>

      <section class="steps">
        <h2>Zubereitung</h2>
        {#if recipe.steps.length === 0}
          <p class="muted">Keine Schritte erfasst.</p>
        {:else}
          <ol>
            {#each recipe.steps as step (step.id)}
              <li>{step.text}</li>
            {/each}
          </ol>
        {/if}

        {#if recipe.notes}
          <h2>Notizen</h2>
          <p class="notes">{recipe.notes}</p>
        {/if}
      </section>
    </div>

    <footer class="actions">
      <button type="button" class="danger" onclick={remove}>Rezept löschen</button>
      <button type="button" class="plan" onclick={() => (showAddDialog = true)}>Zum Essensplan hinzufügen</button>
      <a class="edit" href={`#/recipes/${recipe.id}/edit`}>Bearbeiten</a>
    </footer>
  </article>

  {#if showAddDialog}
    <AddToPlanDialog {recipe} onadded={refreshPlanInfo} onclose={() => (showAddDialog = false)} />
  {/if}
{/if}

<style>
  .back {
    margin: 0 0 0.75rem;
  }

  .back a {
    text-decoration: none;
  }

  .muted {
    color: var(--text-muted);
  }

  .error {
    color: var(--danger);
  }

  .photo {
    display: block;
    width: 100%;
    aspect-ratio: 16 / 9;
    max-height: 26rem;
    margin-bottom: 1rem;
    border-radius: var(--radius);
    object-fit: cover;
    background: var(--accent-soft);
  }

  /* Platzhalter ohne Bild: flach, damit er nicht unnötig Platz einnimmt. */
  .hero {
    display: grid;
    place-items: center;
    height: clamp(6rem, 18vw, 10rem);
    width: 100%;
    margin-bottom: 1rem;
    border-radius: var(--radius);
    background: var(--accent-soft);
    color: var(--accent);
  }

  .title-row {
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .title-row h1 {
    flex: 1;
    margin: 0;
  }

  .description {
    margin: 0.5rem 0 0;
  }

  .facts {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem 1.25rem;
    margin-top: 0.75rem;
    font-size: 0.9rem;
    color: var(--text-muted);
  }

  .facts span {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
  }

  .facts .upcoming {
    color: var(--accent);
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-top: 0.75rem;
  }

  .tag {
    padding: 0.2rem 0.65rem;
    border: none;
    border-radius: 999px;
    background: var(--accent-soft);
    color: var(--accent);
    font-size: 0.85rem;
    cursor: pointer;
  }

  .columns {
    display: grid;
    gap: 1.5rem;
    margin-top: 1.5rem;
  }

  @media (min-width: 60rem) {
    .columns {
      grid-template-columns: minmax(16rem, 1fr) 2fr;
      align-items: start;
    }

    /* Zutaten bleiben beim Scrollen durch lange Schritte sichtbar. */
    .ingredients {
      position: sticky;
      top: 1rem;
    }
  }

  section {
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  h2 {
    margin: 0 0 0.5rem;
    font-size: 1.15rem;
  }

  .steps h2:not(:first-child) {
    margin-top: 1.5rem;
  }

  h3 {
    margin: 1rem 0 0.25rem;
    font-size: 0.95rem;
    color: var(--text-muted);
  }

  .section-head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }

  .section-head h2 {
    margin: 0;
  }

  .servings {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
  }

  .servings-value {
    min-width: 6.5rem;
    text-align: center;
    font-variant-numeric: tabular-nums;
  }

  .hint {
    margin: 0.4rem 0 0;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  ul {
    margin: 0.5rem 0 0;
    padding: 0;
    list-style: none;
  }

  ul li {
    display: grid;
    grid-template-columns: 5.5rem 1fr;
    gap: 0.75rem;
    padding: 0.35rem 0;
    border-bottom: 1px solid var(--border);
  }

  ul li:last-child {
    border-bottom: none;
  }

  .amount {
    text-align: right;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }

  .note {
    color: var(--text-muted);
  }

  ol {
    margin: 0;
    padding-left: 1.5rem;
  }

  ol li {
    padding: 0.4rem 0 0.4rem 0.25rem;
    line-height: 1.6;
  }

  ol li::marker {
    font-weight: 700;
    color: var(--accent);
  }

  .notes {
    margin: 0;
    white-space: pre-line; /* Zeilenumbrüche aus der Eingabe erhalten */
  }

  .icon-button {
    display: inline-grid;
    place-items: center;
    width: 2.25rem;
    height: 2.25rem;
    padding: 0;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    font-size: 1.2rem;
    line-height: 1;
    cursor: pointer;
  }

  .icon-button:hover:not(:disabled) {
    background: var(--surface-hover);
  }

  .icon-button:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .favorite {
    width: 2.75rem;
    height: 2.75rem;
    border: none;
    background: none;
    color: var(--accent);
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    text-decoration: underline;
    font-size: inherit;
    cursor: pointer;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 2rem;
  }

  .plan {
    padding: 0.5rem 1.2rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--accent);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }

  .from-plan {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0.5rem 0 0;
    padding: 0.4rem 0.7rem;
    border-radius: var(--radius);
    background: var(--accent-soft);
    font-size: 0.9rem;
  }

  .back-link {
    text-decoration: none;
  }

  .edit {
    padding: 0.5rem 1.2rem;
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
    text-decoration: none;
  }

  .danger {
    padding: 0.5rem 1rem;
    border: 1px solid var(--danger);
    border-radius: var(--radius);
    background: none;
    color: var(--danger);
    cursor: pointer;
  }
</style>
