<script>
  import { tick } from 'svelte';
  import { api } from '../lib/api.js';
  import { addDays, dayLabel, todayString } from '../lib/format.js';
  import { takePlanFocus } from '../lib/planFocus.svelte.js';
  import Icon from '../components/Icon.svelte';
  import PlanEntry from '../components/PlanEntry.svelte';
  import EntryDialog from '../components/EntryDialog.svelte';

  // Wie viele Tage beim Start und bei jedem Nachladen am Rand.
  const INITIAL_BEFORE = 7;
  const INITIAL_AFTER = 27;
  const CHUNK = 14;

  // Geladener Bereich; days enthält jeden Tag darin als "2026-10-21".
  let from = $state('');
  let to = $state('');
  // Einträge je Tag: { "2026-10-21": [entry, …] }
  let entriesByDate = $state({});
  // Heute und Einkaufszeitraum laut Server (Zeitzone von Home Assistant).
  let today = $state(todayString());
  let period = $state(null);
  let error = $state('');

  let strip; // der waagerecht scrollende Streifen (bind:this)
  let loading = false; // verhindert doppeltes Nachladen beim Scrollen
  let menuEntry = $state(null); // Eintrag, dessen Menü offen ist
  // Tag, den der Plan zeigen soll, z. B. nach dem Einplanen; sonst heute.
  const focusDate = takePlanFocus();

  const days = $derived.by(() => {
    if (!from) return [];
    const list = [];
    for (let d = from; d <= to; d = addDays(d, 1)) list.push(d);
    return list;
  });

  /** Lädt die Einträge von start bis end und sortiert sie nach Tagen ein. */
  async function load(start, end) {
    const result = await api.mealPlan(start, end);
    today = result.today;
    period = result.shopping_period;
    for (let d = start; d <= end; d = addDays(d, 1)) entriesByDate[d] = [];
    for (const entry of result.entries) entriesByDate[entry.date].push(entry);
  }

  async function init() {
    try {
      // Um den gewünschten Tag herum laden (mindestens bis heute).
      const center = focusDate ?? today;
      const start = addDays(center < today ? center : today, -INITIAL_BEFORE);
      const end = addDays(center > today ? center : today, INITIAL_AFTER);
      await load(start, end);
      from = start;
      to = end;
      await tick(); // warten, bis die Spalten im DOM stehen
      scrollToDay(center, 'instant');
    } catch (e) {
      error = e.message;
    }
  }
  init();

  /** Scrollt so, dass date die erste sichtbare Spalte ist. */
  function scrollToDay(date, behavior = 'smooth') {
    const column = strip?.querySelector(`[data-date="${date}"]`);
    if (column) strip.scrollTo({ left: column.offsetLeft - strip.offsetLeft, behavior });
  }

  /** Erste (zumindest halb) sichtbare Spalte. */
  function firstVisibleDay() {
    for (const column of strip.querySelectorAll('[data-date]')) {
      if (column.offsetLeft - strip.offsetLeft + column.offsetWidth / 2 >= strip.scrollLeft) return column.dataset.date;
    }
    return today;
  }

  async function shiftWeek(direction) {
    const target = addDays(firstVisibleDay(), 7 * direction);
    // Vor dem Springen links und rechts vom Ziel mindestens eine Woche
    // laden. Sonst würde das Nachladen am Rand erst während des weichen
    // Scrollens Spalten einfügen, und die Ansicht landete daneben.
    while (addDays(target, -7) < from) await loadEarlier();
    while (addDays(target, 13) > to) await loadLater();
    scrollToDay(target);
  }

  async function loadEarlier() {
    if (loading) return;
    loading = true;
    try {
      const start = addDays(from, -CHUNK);
      await load(start, addDays(from, -1));
      // Neue Spalten links schieben den Inhalt nach rechts; die
      // Scrollposition um ihre Breite korrigieren, damit nichts springt.
      if (!strip) return; // Seite inzwischen verlassen
      const before = strip.scrollWidth;
      from = start;
      await tick();
      strip.scrollLeft += strip.scrollWidth - before;
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function loadLater() {
    if (loading) return;
    loading = true;
    try {
      const end = addDays(to, CHUNK);
      await load(addDays(to, 1), end);
      to = end;
      await tick();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  // Nahe am Rand automatisch weitere Tage laden ("endlos" scrollen).
  function onScroll() {
    if (!strip) return; // Seite wird gerade verlassen
    const margin = strip.clientWidth / 2;
    if (strip.scrollLeft < margin) loadEarlier();
    else if (strip.scrollLeft + strip.clientWidth > strip.scrollWidth - margin) loadLater();
  }

  // Änderungen aus dem Menü in die Anzeige übernehmen, ohne neu zu laden.
  function entrySaved(saved) {
    const list = entriesByDate[saved.date];
    const i = list.findIndex((e) => e.id === saved.id);
    if (i !== -1) list[i] = saved;
    menuEntry = null;
  }

  function entryDeleted(id) {
    const list = entriesByDate[menuEntry.date];
    entriesByDate[menuEntry.date] = list.filter((e) => e.id !== id);
    menuEntry = null;
  }

  const inPeriod = (date) => period && date >= period.from && date <= period.to;
  const lunch = (date) => entriesByDate[date]?.filter((e) => e.slot === 'lunch') ?? [];
  const dinner = (date) => entriesByDate[date]?.filter((e) => e.slot === 'dinner') ?? [];
</script>

<div class="head">
  <h1>Essensplan</h1>
  <div class="controls">
    <button type="button" class="icon-button" title="Eine Woche zurück" onclick={() => shiftWeek(-1)}>
      <Icon name="chevron-left" size={20} />
    </button>
    <button type="button" class="today-button" onclick={() => scrollToDay(today)}>Heute</button>
    <button type="button" class="icon-button" title="Eine Woche vor" onclick={() => shiftWeek(1)}>
      <Icon name="chevron-right" size={20} />
    </button>
  </div>
</div>
{#if period}
  <p class="legend"><span class="swatch"></span> Einkaufszeitraum</p>
{/if}

{#if error}
  <p class="error">Der Plan konnte nicht geladen werden: {error}</p>
{/if}

<div class="strip" bind:this={strip} onscroll={onScroll}>
  {#each days as date (date)}
    {@const label = dayLabel(date)}
    {@const lunchEntries = lunch(date)}
    <section class="day" class:today={date === today} class:period={inPeriod(date)} class:past={date < today} data-date={date}>
      <header class="day-head">
        <span class="weekday">{label.weekday}</span>
        <span class="date">{label.date}</span>
        {#if date === today}<span class="badge">Heute</span>{/if}
      </header>
      <div class="entries">
        {#if lunchEntries.length > 0}
          <span class="slot">Mittag</span>
          {#each lunchEntries as entry (entry.id)}
            <PlanEntry {entry} onmenu={(e) => (menuEntry = e)} />
          {/each}
          <span class="slot">Abend</span>
        {/if}
        {#each dinner(date) as entry (entry.id)}
          <PlanEntry {entry} onmenu={(e) => (menuEntry = e)} />
        {/each}
        <!-- Leerer Tag: großes Plus; belegter Tag: kleines Plus für ein
             weiteres Gericht -->
        <a
          class="add"
          class:small={entriesByDate[date]?.length > 0}
          href="#/plan/add/{date}"
          title="Gericht für {label.weekday} {label.date} planen"
          aria-label="Gericht für {label.weekday} {label.date} planen"
        >+</a>
      </div>
    </section>
  {/each}
</div>

{#if menuEntry}
  <EntryDialog
    entry={menuEntry}
    onsaved={entrySaved}
    ondeleted={entryDeleted}
    onclose={() => (menuEntry = null)}
  />
{/if}

<style>
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
  }

  .head h1 {
    margin: 0;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .icon-button,
  .today-button {
    display: inline-grid;
    place-items: center;
    min-height: 2.25rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
  }

  .icon-button {
    width: 2.25rem;
  }

  .today-button {
    padding: 0 0.9rem;
    font-weight: 600;
  }

  .icon-button:hover,
  .today-button:hover {
    background: var(--surface-hover);
  }

  .legend {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0.5rem 0 0.75rem;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .swatch {
    width: 0.9rem;
    height: 0.9rem;
    border-radius: 0.2rem;
    background: var(--accent-soft);
    border: 1px solid var(--accent);
  }

  .error {
    color: var(--danger);
  }

  /* Waagerechter Streifen: Handy 2 Tage, ab Tablet 4, großer Bildschirm 7.
     scroll-snap lässt nach dem Wischen an einer Spaltenkante einrasten. */
  .strip {
    --gap: 0.5rem;
    --columns: 2;
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: calc((100% - (var(--columns) - 1) * var(--gap)) / var(--columns));
    gap: var(--gap);
    overflow-x: auto;
    scroll-snap-type: x proximity;
    padding-bottom: 0.5rem;
    overscroll-behavior-x: contain;
  }

  @media (min-width: 36rem) {
    .strip {
      --columns: 4;
    }
  }

  @media (min-width: 64rem) {
    .strip {
      --columns: 7;
    }
  }

  .day {
    display: flex;
    flex-direction: column;
    min-height: 14rem;
    padding: 0.4rem;
    border: 1px solid transparent;
    border-radius: var(--radius);
    scroll-snap-align: start;
  }

  .day.period {
    background: var(--accent-soft);
  }

  .day.today {
    border-color: var(--accent);
  }

  .day.past .entries {
    opacity: 0.7;
  }

  .day-head {
    display: flex;
    align-items: baseline;
    gap: 0.35rem;
    margin-bottom: 0.4rem;
    padding: 0 0.15rem;
  }

  .weekday {
    font-weight: 700;
  }

  .date {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .badge {
    margin-left: auto;
    padding: 0 0.4rem;
    border-radius: 999px;
    background: var(--accent);
    color: var(--on-accent);
    font-size: 0.7rem;
    font-weight: 600;
  }

  .entries {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .add {
    display: grid;
    place-items: center;
    min-height: 4rem;
    border: 1px dashed var(--border);
    border-radius: calc(var(--radius) - 0.15rem);
    color: var(--text-muted);
    font-size: 1.6rem;
    text-decoration: none;
  }

  .add.small {
    min-height: 1.9rem;
    font-size: 1.1rem;
  }

  .add:hover {
    border-color: var(--accent);
    color: var(--accent);
  }

  .slot {
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
  }
</style>
