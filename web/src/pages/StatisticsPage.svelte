<script>
  import { api, imageUrl } from '../lib/api.js';
  import { daysBetween } from '../lib/format.js';
  import { navigate } from '../lib/router.svelte.js';
  import { recipeFilter, resetFilter, filterByTag } from '../lib/recipeFilter.svelte.js';

  // Statistik (Schritt 8.D). Der Server rechnet bei jedem Aufruf; gezählt
  // werden Planeinträge bis einschließlich heute.
  const periods = [
    ['3m', 'Letzte 3 Monate'],
    ['12m', 'Letztes Jahr'],
    ['all', 'Gesamt'],
  ];
  let period = $state('3m');
  let stats = $state(null);
  let error = $state('');

  $effect(() => {
    const p = period;
    api
      .statistics(p)
      .then((s) => {
        // Inzwischen umgeschaltet? Dann gilt die neuere Antwort.
        if (p === period) stats = s;
      })
      .catch((e) => (error = `Statistik nicht geladen: ${e.message}`));
  });

  const percent = (part, whole) => (whole === 0 ? 0 : Math.round((100 * part) / whole));
  const times = (n) => (n === 1 ? '1×' : `${n}×`);

  function ago(date) {
    if (!date) return 'noch nie';
    const days = daysBetween(date, stats.today);
    if (days === 0) return 'heute';
    if (days < 14) return `vor ${days} Tagen`;
    if (days < 60) return `vor ${Math.round(days / 7)} Wochen`;
    return `vor ${Math.round(days / 30)} Monaten`;
  }

  /** Rezeptübersicht mit einem Filter öffnen. */
  function openRecipes(setFilter) {
    resetFilter();
    setFilter();
    navigate('/recipes');
  }

  // Größte Zahl der Kategorien, damit die Balken vergleichbar sind.
  const maxTagCount = $derived(Math.max(1, ...(stats?.categories ?? []).flatMap((g) => g.tags.map((t) => t.count))));
  const maxTop = $derived(Math.max(1, ...(stats?.top ?? []).map((r) => r.count)));
</script>

<h1>Statistik</h1>

{#if error}<p class="error">{error}</p>{/if}

{#if stats}
  <section class="tiles" aria-label="Kennzahlen">
    <div class="tile">
      <strong>{stats.recipes}</strong>
      <span>Rezepte</span>
    </div>
    <div class="tile">
      <strong>{stats.planned}</strong>
      <span>schon eingeplant ({percent(stats.planned, stats.recipes)} %)</span>
    </div>
    <button type="button" class="tile link" onclick={() => openRecipes(() => (recipeFilter.neverPlanned = true))}>
      <strong>{stats.never_planned}</strong>
      <span>noch nie eingeplant ({percent(stats.never_planned, stats.recipes)} %) →</span>
    </button>
    <button type="button" class="tile link" onclick={() => openRecipes(() => (recipeFilter.illustrationsOnly = true))}>
      <strong>{stats.illustrated}</strong>
      <span>mit Illustration statt Foto →</span>
    </button>
    <button type="button" class="tile link" onclick={() => openRecipes(() => (recipeFilter.favoritesOnly = true))}>
      <strong>{stats.favorites}</strong>
      <span>Favoriten →</span>
    </button>
  </section>

  <section class="card">
    <div class="head">
      <h2>Am häufigsten gegessen</h2>
      <div class="period-switch" role="group" aria-label="Zeitraum">
        {#each periods as [value, label] (value)}
          <button type="button" aria-pressed={period === value} onclick={() => (period = value)}>{label}</button>
        {/each}
      </div>
    </div>
    <p class="muted">
      {stats.entries} {stats.entries === 1 ? 'Gericht' : 'Gerichte'} im Essensplan
      {stats.from ? `seit ${new Date(stats.from).toLocaleDateString('de-DE')}` : 'insgesamt'} (bis heute){stats.menus
        ? `, davon ${stats.menus === 1 ? 'ein Menü' : `${stats.menus} Menüs`}`
        : ''}
    </p>
    {#if stats.top.length === 0}
      <p class="muted">In diesem Zeitraum stand noch nichts im Essensplan.</p>
    {:else}
      <ol class="top">
        {#each stats.top as r, i (r.id)}
          <li>
            <span class="rank">{i + 1}.</span>
            {#if r.thumbnail}<img src={imageUrl(r.thumbnail)} alt="" />{:else}<span class="noimg"></span>{/if}
            <a href="#/recipes/{r.id}">{r.title}</a>
            <span class="bar" style:width="{(100 * r.count) / maxTop}%"></span>
            <span class="count">{times(r.count)}</span>
          </li>
        {/each}
      </ol>
    {/if}
  </section>

  {#if stats.without_recipe.length > 0}
    <section class="card">
      <h2>Ohne Rezept</h2>
      <p class="muted">Einträge wie „Essen gehen“ im gewählten Zeitraum – sie zählen nicht als Rezept.</p>
      <ul class="plain">
        {#each stats.without_recipe as n (n.note)}<li><span class="count">{times(n.count)}</span> {n.note}</li>{/each}
      </ul>
    </section>
  {/if}

  <section class="card">
    <h2>Lieblingsgerichte, die lange nicht dran waren</h2>
    {#if stats.favorites_waiting.length === 0}
      <p class="muted">Alle Favoriten waren in den letzten 8 Wochen dran oder sind schon eingeplant.</p>
    {:else}
      <ul class="plain">
        {#each stats.favorites_waiting as r (r.id)}
          <li><a href="#/recipes/{r.id}">{r.title}</a> <span class="muted">– zuletzt {ago(r.last_planned)}</span></li>
        {/each}
      </ul>
    {/if}
  </section>

  <section class="card">
    <h2>Zuletzt hinzugefügt</h2>
    <ul class="plain">
      {#each stats.recently_added as r (r.id)}
        <li>
          <a href="#/recipes/{r.id}">{r.title}</a>
          <span class="muted">– {new Date(r.created_at).toLocaleDateString('de-DE')}, {r.last_planned ? `zuletzt ${ago(r.last_planned)}` : 'noch nie eingeplant'}</span>
        </li>
      {/each}
    </ul>
  </section>

  <section class="card">
    <h2>Rezepte nach Kategorien</h2>
    {#each stats.categories as g (g.name)}
      <h3>{g.name}</h3>
      <ul class="bars">
        {#each g.tags as t (t.id)}
          <li>
            <button type="button" class="linklike" onclick={() => { filterByTag(t.id); navigate('/recipes'); }}>{t.name}</button>
            <span class="bar" style:width="{(100 * t.count) / maxTagCount}%"></span>
            <span class="count">{t.count}</span>
          </li>
        {/each}
      </ul>
    {/each}
    <p class="muted">{stats.without_category} {stats.without_category === 1 ? 'Rezept' : 'Rezepte'} ohne Kategorie</p>
  </section>
{:else if !error}
  <p>Lädt …</p>
{/if}

<style>
  h1 {
    margin-top: 0;
  }

  h2 {
    margin: 0;
    font-size: 1.1rem;
  }

  h3 {
    margin: 0.75rem 0 0.25rem;
    font-size: 0.95rem;
    color: var(--text-muted);
  }

  p {
    margin: 0.25rem 0;
  }

  .muted {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .error {
    color: var(--danger);
  }

  .tiles {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: 0.75rem;
    margin-bottom: 1rem;
  }

  .tile {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    padding: 0.75rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    text-align: left;
  }

  .tile strong {
    font-size: 1.6rem;
  }

  .tile span {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .tile.link {
    cursor: pointer;
  }

  .tile.link:hover {
    background: var(--surface-hover);
  }

  .card {
    margin-bottom: 1rem;
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .head {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }

  .period-switch {
    display: inline-flex;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .period-switch button {
    padding: 0.35rem 0.75rem;
    border: none;
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
    font: inherit;
    font-size: 0.85rem;
  }

  .period-switch button[aria-pressed='true'] {
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  ol,
  ul {
    margin: 0.5rem 0 0;
    padding: 0;
    list-style: none;
  }

  .top li,
  .bars li {
    display: grid;
    align-items: center;
    gap: 0.5rem;
    padding: 0.2rem 0;
  }

  .top li {
    grid-template-columns: 1.75rem 2.25rem minmax(0, 14rem) 1fr 2.5rem;
  }

  .bars li {
    grid-template-columns: minmax(0, 11rem) 1fr 2rem;
  }

  .top a,
  .bars .linklike {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .rank {
    color: var(--text-muted);
    text-align: right;
  }

  .top img,
  .noimg {
    width: 2.25rem;
    height: 2.25rem;
    object-fit: cover;
    border-radius: 0.3rem;
    background: var(--surface-hover);
  }

  .bar {
    height: 0.6rem;
    min-width: 2px;
    border-radius: 999px;
    background: var(--accent);
    opacity: 0.75;
  }

  .count {
    color: var(--text-muted);
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .plain li {
    padding: 0.2rem 0;
  }

  .plain .count {
    display: inline-block;
    min-width: 2.5rem;
  }

  .linklike {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }

  .linklike:hover {
    text-decoration: underline;
  }

  @media (max-width: 30rem) {
    .top li {
      grid-template-columns: 1.5rem 2.25rem minmax(0, 1fr) 2.5rem;
    }

    .top .bar {
      display: none;
    }
  }
</style>
