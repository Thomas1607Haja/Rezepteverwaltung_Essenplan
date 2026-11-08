<script>
  import { api, imageUrl } from '../lib/api.js';
  import { daysBetween } from '../lib/format.js';
  import { t, errorText, currentLocale } from '../lib/i18n.svelte.js';
  import { navigate } from '../lib/router.svelte.js';
  import { recipeFilter, resetFilter, filterByTag } from '../lib/recipeFilter.svelte.js';

  // Statistik (Schritt 8.D). Der Server rechnet bei jedem Aufruf; gezählt
  // werden Planeinträge bis einschließlich heute.
  // Texte stehen unter stats.periods.<wert>.
  const periods = ['3m', '12m', 'all'];
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
      .catch((e) => (error = t('stats.load_failed', { message: errorText(e) })));
  });

  const percent = (part, whole) => (whole === 0 ? 0 : Math.round((100 * part) / whole));
  const times = (n) => (n === 1 ? '1×' : `${n}×`);

  /** „zuletzt vor 3 Wochen“ bzw. „noch nie eingeplant“. */
  function lastText(date) {
    if (!date) return t('stats.last.never');
    const days = daysBetween(date, stats.today);
    if (days === 0) return t('stats.last.today');
    if (days < 14) return t('stats.last.days', { count: days });
    if (days < 60) return t('stats.last.weeks', { count: Math.round(days / 7) });
    return t('stats.last.months', { count: Math.round(days / 30) });
  }

  /** Datum kurz in der Schreibweise der Sprache; reine Tage ("2026-10-21") ohne Zeitzonen-Versatz. */
  function shortDate(value) {
    const dateOnly = /^\d{4}-\d{2}-\d{2}$/.test(value);
    return new Intl.DateTimeFormat(currentLocale(), dateOnly ? { timeZone: 'UTC' } : {}).format(new Date(value));
  }

  /** Zeile unter „Am häufigsten gegessen“: Anzahl, Zeitraum, Menüs. */
  function entriesSummary(s) {
    const text = s.from
      ? t('stats.entries_since', { count: s.entries, date: shortDate(s.from) })
      : t('stats.entries_total', { count: s.entries });
    return s.menus ? text + t('stats.menus', { count: s.menus }) : text;
  }

  /** Rezeptübersicht mit einem Filter öffnen. */
  function openRecipes(setFilter) {
    resetFilter();
    setFilter();
    navigate('/recipes');
  }

  // Größte Zahl der Kategorien, damit die Balken vergleichbar sind.
  const maxTagCount = $derived(Math.max(1, ...(stats?.categories ?? []).flatMap((g) => g.tags.map((tag) => tag.count))));
  const maxTop = $derived(Math.max(1, ...(stats?.top ?? []).map((r) => r.count)));
</script>

<h1>{t('stats.title')}</h1>

{#if error}<p class="error">{error}</p>{/if}

{#if stats}
  <section class="tiles" aria-label={t('stats.key_figures')}>
    <div class="tile">
      <strong>{stats.recipes}</strong>
      <span>{t('stats.recipes', { count: stats.recipes })}</span>
    </div>
    <div class="tile">
      <strong>{stats.planned}</strong>
      <span>{t('stats.planned', { percent: percent(stats.planned, stats.recipes) })}</span>
    </div>
    <button type="button" class="tile link" onclick={() => openRecipes(() => (recipeFilter.neverPlanned = true))}>
      <strong>{stats.never_planned}</strong>
      <span>{t('stats.never_planned', { percent: percent(stats.never_planned, stats.recipes) })} →</span>
    </button>
    <button type="button" class="tile link" onclick={() => openRecipes(() => (recipeFilter.illustrationsOnly = true))}>
      <strong>{stats.illustrated}</strong>
      <span>{t('stats.illustrated')} →</span>
    </button>
    <button type="button" class="tile link" onclick={() => openRecipes(() => (recipeFilter.favoritesOnly = true))}>
      <strong>{stats.favorites}</strong>
      <span>{t('stats.favorites')} →</span>
    </button>
  </section>

  <section class="card">
    <div class="head">
      <h2>{t('stats.top_title')}</h2>
      <div class="period-switch" role="group" aria-label={t('stats.period_label')}>
        {#each periods as value (value)}
          <button type="button" aria-pressed={period === value} onclick={() => (period = value)}>
            {t(`stats.periods.${value}`)}
          </button>
        {/each}
      </div>
    </div>
    <p class="muted">{entriesSummary(stats)}</p>
    {#if stats.top.length === 0}
      <p class="muted">{t('stats.top_empty')}</p>
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
      <h2>{t('stats.without_recipe_title')}</h2>
      <p class="muted">{t('stats.without_recipe_hint', { example: t('quick.eating_out') })}</p>
      <ul class="plain">
        {#each stats.without_recipe as n (n.note)}<li><span class="count">{times(n.count)}</span> {n.note}</li>{/each}
      </ul>
    </section>
  {/if}

  <section class="card">
    <h2>{t('stats.favorites_waiting_title')}</h2>
    {#if stats.favorites_waiting.length === 0}
      <p class="muted">{t('stats.favorites_waiting_empty')}</p>
    {:else}
      <ul class="plain">
        {#each stats.favorites_waiting as r (r.id)}
          <li><a href="#/recipes/{r.id}">{r.title}</a> <span class="muted">– {lastText(r.last_planned)}</span></li>
        {/each}
      </ul>
    {/if}
  </section>

  <section class="card">
    <h2>{t('stats.recent_title')}</h2>
    <ul class="plain">
      {#each stats.recently_added as r (r.id)}
        <li>
          <a href="#/recipes/{r.id}">{r.title}</a>
          <span class="muted">– {shortDate(r.created_at)}, {lastText(r.last_planned)}</span>
        </li>
      {/each}
    </ul>
  </section>

  <section class="card">
    <h2>{t('stats.categories_title')}</h2>
    {#each stats.categories as g (g.name)}
      <h3>{g.name}</h3>
      <ul class="bars">
        {#each g.tags as tag (tag.id)}
          <li>
            <button type="button" class="linklike" onclick={() => { filterByTag(tag.id); navigate('/recipes'); }}>{tag.name}</button>
            <span class="bar" style:width="{(100 * tag.count) / maxTagCount}%"></span>
            <span class="count">{tag.count}</span>
          </li>
        {/each}
      </ul>
    {/each}
    <p class="muted">{t('stats.without_category', { count: stats.without_category })}</p>
  </section>
{:else if !error}
  <p>{t('common.loading')}</p>
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
