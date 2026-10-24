<script>
  import { SvelteSet } from 'svelte/reactivity';
  import { api } from '../lib/api.js';
  import { dayLabel } from '../lib/format.js';

  let next = $state(false); // false = eingekaufte Woche, true = nächster Einkauf
  // Beim Öffnen: Ist für die eingekaufte Woche nichts mehr zu senden,
  // gleich den nächsten Einkauf zeigen – den plant man dann gerade.
  let firstLoad = true;
  let preview = $state(null);
  let error = $state('');
  // Abgewählte Artikel (über ihren key). SvelteSet ist eine Menge, deren
  // Änderungen Svelte bemerkt – ein normales Set würde die Anzeige nicht
  // aktualisieren.
  const deselected = new SvelteSet();
  // Abgewählte Artikel beim Nachsenden der Differenz
  const deselectedMissing = new SvelteSet();

  async function load() {
    error = '';
    try {
      preview = await api.shoppingPreview(next);
      deselected.clear();
      deselectedMissing.clear();
      if (firstLoad && !next && preview.items.length === 0 && preview.missing.length === 0) {
        firstLoad = false;
        next = true;
        preview = null;
        await load();
        return;
      }
      firstLoad = false;
    } catch (e) {
      error = `Einkaufsliste nicht geladen: ${e.message}`;
    }
  }

  function switchPeriod(value) {
    if (next === value) return;
    next = value;
    preview = null;
    sendResult = null;
    sendError = '';
    load();
  }

  // Ergebnis bzw. Fehler der letzten Übertragung
  let sending = $state(false);
  let sendResult = $state(null);
  let sendError = $state('');
  let needsList = $state(false); // keine Bring-Liste gewählt

  /** Sendet die Liste bzw. mit missing = true die Differenz. */
  async function send(missing = false) {
    sending = true;
    sendError = '';
    needsList = false;
    try {
      sendResult = missing
        ? await api.sendShopping(next, [...deselectedMissing], { missing: true })
        : await api.sendShopping(next, [...deselected]);
      await load();
    } catch (e) {
      sendError = e.message;
      needsList = e.field === 'bring_entity';
    } finally {
      sending = false;
    }
  }

  /** Zusammenfassung wie "12 Artikel gesendet, davon 2 ergänzt". */
  function sentSummary(r) {
    const n = r.sent.length;
    const merged = r.sent.filter((s) => s.merged).length;
    let text = n === 1 ? 'Ein Artikel an Bring gesendet' : `${n} Artikel an Bring gesendet`;
    if (merged > 0) text += `, davon ${merged} bei schon vorhandenen ergänzt`;
    return text + '.';
  }

  load();

  const selectedCount = $derived(preview ? preview.items.length - deselected.size : 0);

  function toggle(key, set = deselected) {
    if (set.has(key)) set.delete(key);
    else set.add(key);
  }

  function selectAll(all) {
    deselected.clear();
    if (!all) for (const it of preview.items) deselected.add(it.key);
  }

  /** Kurzer Tag mit Datum, z. B. "Mi 21.10." */
  function day(date) {
    const d = dayLabel(date);
    return `${d.weekday} ${d.date}`;
  }

  /** Warum ein Gericht nicht in die Liste eingeht. */
  function reason(entry) {
    if (entry.recipe_id == null) return 'ohne Rezept';
    if (entry.status === 'transferred') return 'schon an Bring gesendet';
    if (entry.status === 'have_ingredients') return 'Zutaten vorhanden';
    if (entry.date < preview.today) return 'vorbei';
    return '';
  }
</script>

<h1>Einkauf</h1>

<div class="period-switch" role="group" aria-label="Zeitraum">
  <button type="button" aria-pressed={!next} onclick={() => switchPeriod(false)}>Eingekaufte Woche</button>
  <button type="button" aria-pressed={next} onclick={() => switchPeriod(true)}>Nächster Einkauf</button>
</div>

{#if error}
  <p class="error">{error}</p>
{:else if !preview}
  <p class="muted">Lädt …</p>
{:else}
  <p class="muted period">{day(preview.period.from)} bis {day(preview.period.to)}</p>

  <section class="card">
    <h2>Gerichte</h2>
    {#if preview.entries.length === 0}
      <p class="muted">Für diesen Zeitraum ist nichts geplant. <a href="#/">Zum Essensplan</a></p>
    {:else}
      <ul class="entries">
        {#each preview.entries as entry (entry.id)}
          <li class:excluded={!entry.included}>
            <span class="day">
              {day(entry.date)}
              {#if entry.slot === 'lunch'}<small>mittags</small>{:else if entry.slot === 'coffee'}<small>nachmittags</small>{/if}
            </span>
            <span class="title">
              {#if entry.recipe_id != null}
                <a href="#/recipes/{entry.recipe_id}/entry/{entry.id}">{entry.title}</a>
              {:else}
                {entry.title}
              {/if}
            </span>
            <span class="servings">{entry.recipe_id != null ? `${entry.servings} ${entry.yield_unit || 'Port.'}` : ''}</span>
            {#if !entry.included}<span class="reason">{reason(entry)}</span>{/if}
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <section class="card">
    <div class="list-head">
      <h2>Einkaufsliste</h2>
      {#if preview.items.length > 0}
        <span class="muted">{selectedCount} von {preview.items.length} ausgewählt</span>
        <span class="bulk">
          <button type="button" class="link" onclick={() => selectAll(true)}>Alle</button>
          <button type="button" class="link" onclick={() => selectAll(false)}>Keine</button>
        </span>
      {/if}
    </div>

    {#if sendResult}
      <div class="result" role="status">
        <p>{sentSummary(sendResult)}</p>
        {#if sendResult.failed.length > 0}
          <p class="error">Nicht angekommen – bitte später nachsenden oder von Hand eintragen:</p>
          <ul class="failed">
            {#each sendResult.failed as f (f.name)}
              <li>{f.name}{f.spec ? ` (${f.spec})` : ''}</li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}

    {#if preview.items.length === 0}
      <p class="muted">Nichts einzukaufen.</p>
    {:else}
      <ul class="items">
        {#each preview.items as item (item.key)}
          <li>
            <label class:off={deselected.has(item.key)}>
              <input type="checkbox" checked={!deselected.has(item.key)} onchange={() => toggle(item.key)} />
              <span class="name">{item.name}</span>
              <span class="spec">{item.spec}</span>
              <small class="from">{item.recipes.join(', ')}</small>
            </label>
          </li>
        {/each}
      </ul>

      <div class="send">
        <button type="button" class="primary" onclick={() => send()} disabled={sending}>
          {sending ? 'Sendet …' : `An Bring senden (${selectedCount})`}
        </button>
        {#if sendError}
          <span class="error">
            {sendError}{#if needsList}: <a href="#/settings">Zu den Einstellungen</a>{/if}
          </span>
        {:else}
          <small class="muted">Abgewählte Artikel gelten danach als erledigt.</small>
        {/if}
      </div>
    {/if}
  </section>

  {#if preview.missing.length > 0 || preview.surplus.length > 0}
    <section class="card">
      <h2>Nach dem Senden geändert</h2>
      <p class="muted hint">Gerichte wurden nach dem Senden geändert, verschoben oder entfernt.</p>

      {#if preview.missing.length > 0}
        <h3>Fehlt noch auf der Bring-Liste</h3>
        <ul class="items">
          {#each preview.missing as item (item.key)}
            <li>
              <label class:off={deselectedMissing.has(item.key)}>
                <input
                  type="checkbox"
                  checked={!deselectedMissing.has(item.key)}
                  onchange={() => toggle(item.key, deselectedMissing)}
                />
                <span class="name">{item.name}</span>
                <span class="spec">{item.spec}</span>
                <small class="from">{item.recipes.join(', ')}</small>
              </label>
            </li>
          {/each}
        </ul>
        <div class="send">
          <button type="button" class="primary" onclick={() => send(true)} disabled={sending}>
            {sending ? 'Sendet …' : `Differenz nachsenden (${preview.missing.length - deselectedMissing.size})`}
          </button>
        </div>
      {/if}

      {#if preview.surplus.length > 0}
        <h3>Zu viel gesendet</h3>
        <p class="muted hint">Wird nicht aus Bring gelöscht – vielleicht schon gekauft. Am besten für ein anderes Gericht einplanen.</p>
        <ul class="surplus">
          {#each preview.surplus as item (item.key)}
            <li><span class="name">{item.name}</span> <span class="spec">{item.spec}</span></li>
          {/each}
        </ul>
      {/if}
    </section>
  {/if}
{/if}

<style>
  .muted {
    color: var(--text-muted);
  }

  .error {
    color: var(--danger);
  }

  p {
    margin: 0;
  }

  .period {
    margin: 0.5rem 0 1rem;
  }

  .period-switch {
    display: inline-flex;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .period-switch button {
    padding: 0.4rem 0.9rem;
    border: none;
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
  }

  .period-switch button[aria-pressed='true'] {
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .card {
    margin-bottom: 1rem;
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  h2 {
    margin: 0 0 0.5rem;
    font-size: 1.1rem;
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  /* Gerichte: Tag | Titel | Portionen, darunter ggf. der Grund */
  .entries li {
    display: grid;
    grid-template-columns: 6.5rem 1fr auto;
    column-gap: 0.75rem;
    padding: 0.35rem 0;
    border-top: 1px solid var(--border);
  }

  .entries li:first-child {
    border-top: none;
  }

  .day,
  .servings {
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .day small {
    display: block;
    font-size: 0.8rem;
  }

  .title {
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .title a {
    color: inherit;
    text-decoration: none;
  }

  .title a:hover {
    text-decoration: underline;
  }

  .reason {
    grid-column: 2 / -1;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .excluded .title,
  .excluded .day,
  .excluded .servings {
    opacity: 0.55;
  }

  .list-head {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    column-gap: 0.75rem;
    margin-bottom: 0.25rem;
  }

  .list-head h2 {
    margin: 0;
  }

  .bulk {
    margin-left: auto;
    display: flex;
    gap: 0.75rem;
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    cursor: pointer;
  }

  /* Artikel: Häkchen | Name, Menge | Rezepte */
  .items label {
    display: grid;
    grid-template-columns: auto 1fr auto;
    grid-template-areas:
      'check name spec'
      'check from from';
    column-gap: 0.6rem;
    align-items: baseline;
    padding: 0.45rem 0;
    border-top: 1px solid var(--border);
    cursor: pointer;
  }

  .items li:first-child label {
    border-top: none;
  }

  .items input {
    grid-area: check;
    width: 1.15rem;
    height: 1.15rem;
    accent-color: var(--accent);
  }

  .name {
    grid-area: name;
    font-weight: 600;
  }

  .spec {
    grid-area: spec;
    font-variant-numeric: tabular-nums;
    text-align: right;
  }

  .from {
    grid-area: from;
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .off .name,
  .off .spec {
    color: var(--text-muted);
    text-decoration: line-through;
  }

  h3 {
    margin: 1rem 0 0.25rem;
    font-size: 1rem;
  }

  .hint {
    font-size: 0.85rem;
  }

  .surplus li {
    display: flex;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.35rem 0;
    border-top: 1px solid var(--border);
  }

  .surplus li:first-child {
    border-top: none;
  }

  .surplus .name {
    font-weight: 400;
  }

  .result {
    margin: 0.25rem 0 0.75rem;
    padding: 0.6rem 0.75rem;
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--accent-soft);
  }

  .failed {
    margin-top: 0.25rem;
    padding-left: 1.25rem;
    list-style: disc;
  }

  .send {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin-top: 1rem;
  }

  .primary {
    padding: 0.55rem 1.1rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
    cursor: pointer;
  }

  .primary:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
