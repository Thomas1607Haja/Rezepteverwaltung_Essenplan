<script>
  import { tick } from 'svelte';
  import { api } from '../lib/api.js';
  import { addDays, dayLabel, todayString } from '../lib/format.js';
  import { takePlanFocus } from '../lib/planFocus.svelte.js';
  import Icon from '../components/Icon.svelte';
  import PlanEntry from '../components/PlanEntry.svelte';
  import MenuCard from '../components/MenuCard.svelte';
  import MenuDialog from '../components/MenuDialog.svelte';
  import EntryDialog from '../components/EntryDialog.svelte';
  import MoveWarningDialog from '../components/MoveWarningDialog.svelte';
  import { planDrag } from '../lib/planDrag.js';

  // Wie viele Tage beim Start und bei jedem Nachladen am Rand.
  const INITIAL_BEFORE = 7;
  const INITIAL_AFTER = 27;
  const CHUNK = 14;

  // Geladener Bereich; days enthält jeden Tag darin als "2026-10-21".
  let from = $state('');
  let to = $state('');
  // Einträge je Tag: { "2026-10-21": [entry, …] }
  let entriesByDate = $state({});
  // Menüs je Tag (Schritt 9.4); ihre Gänge stehen in entriesByDate.
  let menusByDate = $state({});
  // Gänge der Gang-Gruppe für Namen und Auswahl im Menü
  let courses = $state([]);
  api
    .listTagGroups()
    .then((groups) => (courses = groups.find((g) => g.role === 'course')?.tags ?? []))
    .catch(() => {});
  let openMenu = $state(null); // ID des Menüs, dessen Dialog offen ist
  // Heute, eingekaufte Woche und nächster Einkauf laut Server (Zeitzone
  // von Home Assistant, Einkaufstag und -uhrzeit aus den Einstellungen).
  let today = $state(todayString());
  let period = $state(null);
  let nextPeriod = $state(null);
  let error = $state('');

  let strip; // der waagerecht scrollende Streifen (bind:this)
  let loading = false; // verhindert doppeltes Nachladen beim Scrollen
  let menuEntry = $state(null); // Eintrag, dessen Menü offen ist
  let dropTarget = $state(null); // Ziel beim Ziehen, zum Hervorheben
  let pendingMove = $state(null); // { source, fromDate, move, warnings }, wartet auf Bestätigung
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
    nextPeriod = result.next_shopping_period ?? null;
    for (let d = start; d <= end; d = addDays(d, 1)) {
      entriesByDate[d] = [];
      menusByDate[d] = [];
    }
    for (const entry of result.entries) entriesByDate[entry.date].push(entry);
    for (const menu of result.menus ?? []) menusByDate[menu.date]?.push(menu);
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
    await ensureLoaded(target);
    scrollToDay(target);
  }

  // Vor dem Springen links und rechts von date mindestens eine Woche
  // laden. Sonst würde das Nachladen am Rand erst während des weichen
  // Scrollens Spalten einfügen, und die Ansicht landete daneben.
  async function ensureLoaded(date) {
    while (addDays(date, -7) < from) await loadEarlier();
    while (addDays(date, 13) > to) await loadLater();
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

  /** Sucht einen geladenen Eintrag anhand seiner ID. */
  function findEntry(id) {
    for (const list of Object.values(entriesByDate)) {
      const entry = list.find((e) => e.id === id);
      if (entry) return entry;
    }
    return null;
  }

  /** Sucht ein geladenes Menü anhand seiner ID. */
  function findMenu(id) {
    for (const list of Object.values(menusByDate)) {
      const menu = list.find((m) => m.id === id);
      if (menu) return menu;
    }
    return null;
  }

  // Abgelegt: auf einem Gericht oder Menü = tauschen, auf freier Fläche =
  // anhängen. Menüs (und Gänge eines Menüs) wandern immer als Block.
  function handleDrop(source, target) {
    const from = source.kind === 'menu' ? findMenu(source.id) : findEntry(source.id);
    if (!from) return;
    runMove(source, from.date, {
      date: target.date,
      slot: target.slot,
      swap_with: target.type === 'swap' && target.kind === 'entry' ? target.id : null,
      swap_with_menu: target.type === 'swap' && target.kind === 'menu' ? target.id : null,
    });
  }

  /** Schickt das Verschieben an den Server; bei Warnungen erst fragen. */
  async function runMove(source, fromDate, move) {
    try {
      const result = source.kind === 'menu' ? await api.moveMenu(source.id, move) : await api.moveEntry(source.id, move);
      if (!result.applied) {
        pendingMove = { source, fromDate, move, warnings: result.warnings };
        return;
      }
      await reloadDates([fromDate, move.date, ...result.entries.map((e) => e.date)]);
    } catch (e) {
      alert(`Verschieben fehlgeschlagen: ${e.message}`);
      await reloadDates([fromDate, move.date]);
    }
  }

  async function moveChoice(choice) {
    const { source, fromDate, move, warnings } = pendingMove;
    pendingMove = null;
    if (choice === 'cancel') return;
    const intoPeriod = warnings.filter((w) => w.type === 'planned_into_period').map((w) => w.entry_id);
    const haveIngredients = choice === 'have_ingredients' ? intoPeriod : [];
    await runMove(source, fromDate, { ...move, confirm: true, have_ingredients: haveIngredients });
    if (choice === 'send') await sendToBring(intoPeriod);
  }

  // Einträge und Menüs eines Slots in Anzeigereihenfolge; Gänge eines
  // Menüs stecken in seiner Karte.
  function items(date, slot) {
    const list = [
      ...(menusByDate[date] ?? []).filter((m) => m.slot === slot).map((m) => ({ kind: 'menu', key: `m${m.id}`, position: m.position, menu: m })),
      ...(entriesByDate[date] ?? [])
        .filter((e) => e.slot === slot && !e.menu_id)
        .map((e) => ({ kind: 'entry', key: `e${e.id}`, position: e.position, entry: e })),
    ];
    return list.sort((a, b) => a.position - b.position);
  }

  const menuEntries = (menu) => (entriesByDate[menu.date] ?? []).filter((e) => e.menu_id === menu.id).sort((a, b) => a.position - b.position);
  const courseName = (id) => courses.find((c) => c.id === id)?.name ?? '';

  // Eintrag zu einem Menü erweitern und gleich das Menü öffnen.
  async function toMenu(entry) {
    menuEntry = null;
    try {
      // Gang des Rezepts (z. B. Kuchen & Torten), sonst Hauptgericht.
      let course = 0;
      if (entry.recipe_id) {
        const recipe = await api.getRecipe(entry.recipe_id);
        const own = recipe.tags.find((t) => courses.some((c) => c.id === t.id));
        course = own?.id ?? courses.find((c) => c.name.toLowerCase() === 'hauptgericht')?.id ?? 0;
      }
      const menu = await api.toMenu(entry.id, { course_tag_id: course });
      await reloadDates([entry.date]);
      openMenu = menu.id;
    } catch (e) {
      alert(`Menü nicht angelegt: ${e.message}`);
    }
  }

  // Menü über seinen Dialog verschieben.
  async function moveMenuTo(menu, date) {
    openMenu = null;
    await runMove({ kind: 'menu', id: menu.id }, menu.date, { date, slot: menu.slot });
    await ensureLoaded(date);
    await tick();
    scrollToDay(date);
  }

  /** Sendet die Zutaten der Gerichte ids an Bring (nach dem Verschieben). */
  async function sendToBring(ids) {
    try {
      const result = await api.sendShopping(false, [], { entries: ids });
      const failed = result.failed.length > 0 ? ` ${result.failed.length} Artikel kamen nicht an (siehe Einkauf).` : '';
      alert(`${result.sent.length} Artikel an Bring gesendet.${failed}`);
    } catch (e) {
      alert(`Senden an Bring fehlgeschlagen: ${e.message}`);
    }
    await reloadDates(ids.map((id) => findEntry(id)?.date).filter(Boolean));
  }

  /** Lädt die geladenen unter den Tagen dates neu (z. B. nach einem Tausch). */
  async function reloadDates(dates) {
    for (const date of new Set(dates)) {
      if (date >= from && date <= to) await load(date, date);
    }
  }

  // Verschieben über das Menü (ohne Ziehen), z. B. per Tastatur.
  async function moveFromMenu(entry, date) {
    menuEntry = null;
    await runMove({ kind: 'entry', id: entry.id }, entry.date, { date, slot: entry.slot });
    await ensureLoaded(date);
    await tick();
    scrollToDay(date);
  }

  const isDropTarget = (date, slot) =>
    dropTarget?.type === 'append' && dropTarget.date === date && dropTarget.slot === slot;

  const inPeriod = (date) => period && date >= period.from && date <= period.to;
  const inNext = (date) => nextPeriod && date >= nextPeriod.from && date <= nextPeriod.to;
  const range = (p) => `${dayLabel(p.from).weekday} ${dayLabel(p.from).date}–${dayLabel(p.to).weekday} ${dayLabel(p.to).date}`;

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
  <p class="legend">
    <span class="item"><span class="swatch"></span> Eingekauft: {range(period)}</span>
    {#if nextPeriod}<span class="item"><span class="swatch next"></span> Nächster Einkauf: {range(nextPeriod)}</span>{/if}
  </p>
{/if}

{#if error}
  <p class="error">Der Plan konnte nicht geladen werden: {error}</p>
{/if}

{#snippet planItem(item)}
  {#if item.kind === 'menu'}
    <MenuCard
      menu={item.menu}
      entries={menuEntries(item.menu)}
      {courseName}
      onopen={(m) => (openMenu = m.id)}
      dropTarget={dropTarget?.kind === 'menu' && dropTarget?.id === item.menu.id}
    />
  {:else}
    <PlanEntry
      entry={item.entry}
      onmenu={(e) => (menuEntry = e)}
      dropTarget={dropTarget?.kind === 'entry' && dropTarget?.id === item.entry.id}
    />
  {/if}
{/snippet}

<div
  class="strip"
  bind:this={strip}
  onscroll={onScroll}
  use:planDrag={{ ondrop: handleDrop, ontarget: (t) => (dropTarget = t) }}
>
  {#each days as date (date)}
    {@const label = dayLabel(date)}
    {@const lunchEntries = items(date, 'lunch')}
    {@const coffeeEntries = items(date, 'coffee')}
    <section class="day" class:today={date === today} class:period={inPeriod(date)} class:next={inNext(date)} class:past={date < today} data-date={date}>
      <header class="day-head">
        <span class="weekday">{label.weekday}</span>
        <span class="date">{label.date}</span>
        {#if date === today}<span class="badge">Heute</span>{/if}
      </header>
      <!-- data-slot-zone: Ablagefläche beim Ziehen (lib/planDrag.js) -->
      <div class="entries">
        {#if lunchEntries.length > 0}
          <div class="zone" data-slot-zone="lunch" class:drop-target={isDropTarget(date, 'lunch')}>
            <span class="slot">Mittag</span>
            {#each lunchEntries as item (item.key)}{@render planItem(item)}{/each}
          </div>
        {/if}
        {#if coffeeEntries.length > 0}
          <div class="zone" data-slot-zone="coffee" class:drop-target={isDropTarget(date, 'coffee')}>
            <span class="slot">Kaffee</span>
            {#each coffeeEntries as item (item.key)}{@render planItem(item)}{/each}
          </div>
        {/if}
        <div class="zone dinner" data-slot-zone="dinner" class:drop-target={isDropTarget(date, 'dinner')}>
          {#if lunchEntries.length > 0 || coffeeEntries.length > 0}<span class="slot">Abend</span>{/if}
          {#each items(date, 'dinner') as item (item.key)}{@render planItem(item)}{/each}
          <!-- Leerer Tag: großes Plus; belegter Tag: kleines Plus für ein
               weiteres Gericht -->
          <a
            class="add"
            class:small={entriesByDate[date]?.length > 0}
            href="#/plan/add/{date}"
            draggable="false"
            title="Gericht für {label.weekday} {label.date} planen"
            aria-label="Gericht für {label.weekday} {label.date} planen"
          >+</a>
        </div>
      </div>
    </section>
  {/each}
</div>

{#if menuEntry}
  <EntryDialog
    entry={menuEntry}
    onsaved={entrySaved}
    ondeleted={entryDeleted}
    onmove={moveFromMenu}
    ontomenu={toMenu}
    onclose={() => (menuEntry = null)}
  />
{/if}

{#if openMenu}
  <MenuDialog
    menuId={openMenu}
    {courses}
    onchanged={(dates) => reloadDates(dates)}
    onmove={moveMenuTo}
    onclose={() => (openMenu = null)}
  />
{/if}

{#if pendingMove}
  <MoveWarningDialog warnings={pendingMove.warnings} onchoice={moveChoice} />
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
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem 1rem;
    margin: 0.5rem 0 0.75rem;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .legend .item {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
  }

  .swatch {
    width: 0.9rem;
    height: 0.9rem;
    border-radius: 0.2rem;
    background: var(--accent-soft);
    border: 1px solid var(--accent);
  }

  .swatch.next {
    background: var(--next-soft);
    border-color: var(--next);
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

  .day.next {
    background: var(--next-soft);
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
    flex: 1;
    flex-direction: column;
    gap: 0.4rem;
  }

  .zone {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    border-radius: calc(var(--radius) - 0.15rem);
  }

  /* Der Abend füllt den Rest der Spalte: auch die freie Fläche unter den
     Gerichten nimmt beim Ziehen etwas an. */
  .zone.dinner {
    flex: 1;
  }

  .zone.drop-target {
    outline: 2px dashed var(--accent);
    outline-offset: 3px;
  }

  /* Während des Ziehens keinen Text markieren (Klasse aus planDrag.js) */
  :global(body.plan-dragging) {
    user-select: none;
    -webkit-user-select: none;
    cursor: grabbing;
  }

  /* Einrasten an Spaltenkanten würde das Mitscrollen am Rand beim Ziehen
     sofort wieder zurücksetzen. */
  :global(body.plan-dragging) .strip {
    scroll-snap-type: none;
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
