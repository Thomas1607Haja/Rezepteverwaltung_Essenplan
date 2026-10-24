<script>
  import { api } from '../lib/api.js';
  import { formatBytes, formatUptime } from '../lib/format.js';
  import PantryEditor from '../components/PantryEditor.svelte';
  import ApiKeysEditor from '../components/ApiKeysEditor.svelte';
  import TagManager from '../components/TagManager.svelte';

  // Ein Promise direkt im Markup: {#await} zeigt je nach Zustand
  // "lädt", das Ergebnis oder den Fehler an. Neu zuweisen lädt neu.
  let system = $state(api.system());

  function reload() {
    system = api.system();
  }

  // Haushalt: Familiengröße und Einkaufstag
  const weekdays = [
    ['monday', 'Montag'],
    ['tuesday', 'Dienstag'],
    ['wednesday', 'Mittwoch'],
    ['thursday', 'Donnerstag'],
    ['friday', 'Freitag'],
    ['saturday', 'Samstag'],
    ['sunday', 'Sonntag'],
  ];
  let household = $state(null);
  let saved = ''; // Stand beim Laden bzw. letzten Speichern
  let message = $state('');
  let messageIsError = $state(false);
  let saving = $state(false);

  api
    .getSettings()
    .then((s) => {
      household = s;
      saved = JSON.stringify(s);
    })
    .catch((e) => showMessage(`Einstellungen nicht geladen: ${e.message}`, true));

  const changed = $derived(household !== null && JSON.stringify(household) !== saved);

  // To-do-Listen aus Home Assistant für die Auswahl der Bring-Liste
  let ha = $state(null);
  api
    .homeAssistantStatus()
    .then((s) => (ha = s))
    .catch((e) => (ha = { mode: '', lists: [], error: e.message }));

  // Eine gespeicherte Liste, die Home Assistant nicht (mehr) kennt, bleibt
  // auswählbar, damit sie nicht stillschweigend verloren geht.
  const missingEntity = $derived(
    household?.bring_entity && ha && !ha.lists.some((l) => l.entity_id === household.bring_entity)
      ? household.bring_entity
      : '',
  );

  function showMessage(text, isError = false) {
    message = text;
    messageIsError = isError;
  }

  async function saveHousehold(event) {
    event.preventDefault();
    saving = true;
    try {
      // Nur die Felder dieses Formulars: Die Adresse für die App wird unter
      // App-Zugang gespeichert und bliebe hier sonst auf altem Stand.
      household = await api.updateSettings({
        family_size: Number(household.family_size),
        shopping_day: household.shopping_day,
        shopping_time: household.shopping_time,
        plan_preview: household.plan_preview,
        coffee_slot: household.coffee_slot,
        lunch_time: household.lunch_time,
        coffee_time: household.coffee_time,
        dinner_time: household.dinner_time,
        bring_entity: household.bring_entity,
      });
      saved = JSON.stringify(household);
      showMessage('Gespeichert.');
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      saving = false;
    }
  }
</script>

<h1>Einstellungen</h1>

<section class="card">
  <h2>Haushalt</h2>
  {#if household}
    <form class="household" onsubmit={saveHousehold}>
      <label>
        <span>Familiengröße</span>
        <input type="number" min="1" max="50" bind:value={household.family_size} oninput={() => (message = '')} />
        <small>Standard-Portionen für neue Einträge im Essensplan</small>
      </label>
      <label>
        <span>Einkaufstag</span>
        <select bind:value={household.shopping_day} onchange={() => (message = '')}>
          {#each weekdays as [value, name] (value)}
            <option {value}>{name}</option>
          {/each}
        </select>
        <small>Mit ihm beginnt die eingekaufte Woche</small>
      </label>
      <label>
        <span>Uhrzeit des Einkaufs</span>
        <input type="time" step="60" bind:value={household.shopping_time} oninput={() => (message = '')} />
        <small>Ab dieser Uhrzeit gilt am Einkaufstag eingekauft; vorher ist die neue Woche der „nächste Einkauf“</small>
      </label>
      <label>
        <span>Bring-Liste</span>
        <select bind:value={household.bring_entity} onchange={() => (message = '')} disabled={!ha}>
          <option value="">– keine gewählt –</option>
          {#each ha?.lists ?? [] as list (list.entity_id)}
            <option value={list.entity_id}>{list.name}</option>
          {/each}
          {#if missingEntity}<option value={missingEntity}>{missingEntity} (nicht gefunden)</option>{/if}
        </select>
        {#if !ha}
          <small>Listen werden geladen …</small>
        {:else if ha.error}
          <small class="error">Fehler: {ha.error}</small>
        {:else if ha.mode === 'test'}
          <small>Testmodus ohne Home Assistant: Artikel landen nur in einer Testliste.</small>
        {:else if ha.lists.length === 0}
          <small>Keine To-do-Liste gefunden. Bitte die Bring-Integration in Home Assistant einrichten.</small>
        {:else}
          <small>To-do-Liste in Home Assistant, an die die Einkaufsliste geht</small>
        {/if}
      </label>
      <label class="toggle">
        <input type="checkbox" bind:checked={household.plan_preview} onchange={() => (message = '')} />
        <span>Kurzvorschau beim Einplanen</span>
        <small>Beim Hinzufügen zum Essensplan Portionen wählen und Zutaten abwählen, die noch im Haus sind</small>
      </label>
      <label class="toggle">
        <input type="checkbox" bind:checked={household.coffee_slot} onchange={() => (message = '')} />
        <span>Slot „Kaffee & Kuchen“</span>
        <small>Zwischen Mittag- und Abendessen, z. B. für Kuchen am Wochenende</small>
      </label>
      <fieldset class="times">
        <legend>Uhrzeiten im Home-Assistant-Kalender</legend>
        <label><span>Mittag</span><input type="time" step="60" bind:value={household.lunch_time} oninput={() => (message = '')} /></label>
        {#if household.coffee_slot}
          <label><span>Kaffee</span><input type="time" step="60" bind:value={household.coffee_time} oninput={() => (message = '')} /></label>
        {/if}
        <label><span>Abend</span><input type="time" step="60" bind:value={household.dinner_time} oninput={() => (message = '')} /></label>
      </fieldset>
      <div class="save">
        <button type="submit" class="primary" disabled={!changed || saving}>{saving ? 'Speichert …' : 'Speichern'}</button>
        {#if message}<span class:error={messageIsError} class:ok={!messageIsError}>{message}</span>{/if}
      </div>
    </form>
  {:else if message}
    <p class="error">{message}</p>
  {:else}
    <p>Lädt …</p>
  {/if}
</section>

<PantryEditor />

<TagManager />

<ApiKeysEditor />

<section class="card">
  <h2>System</h2>
  {#await system}
    <p>Verbindung wird geprüft …</p>
  {:then info}
    <dl>
      <dt>Version</dt>
      <dd>{info.version}</dd>
      <dt>Läuft seit</dt>
      <dd>{formatUptime(info.uptime_seconds)}</dd>
      <dt>Arbeitsspeicher</dt>
      <dd>{formatBytes(info.memory_bytes)}</dd>
      <dt>Spitze seit Start</dt>
      <dd>{formatBytes(info.memory_peak_bytes)}</dd>
      <dt>Datenbank</dt>
      <dd>{formatBytes(info.database_bytes)}</dd>
      <dt>Bilder</dt>
      <dd>{formatBytes(info.images_bytes)} ({info.images_count} Dateien)</dd>
    </dl>
    <button type="button" onclick={reload}>Aktualisieren</button>
  {:catch error}
    <p class="error">Nicht verbunden: {error.message}</p>
  {/await}
</section>

<style>
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

  .card p {
    margin: 0;
  }

  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.35rem 1.25rem;
    margin: 0 0 0.75rem;
  }

  dt {
    color: var(--text-muted);
  }

  dd {
    margin: 0;
    font-variant-numeric: tabular-nums;
  }

  button {
    padding: 0.4rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    cursor: pointer;
  }

  button:hover {
    background: var(--surface-hover);
  }

  .error {
    color: var(--danger);
  }

  .ok {
    color: var(--accent);
  }

  .household {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: 0.75rem 1.25rem;
  }

  .household label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .household label span {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .times {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    margin: 0;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .times legend {
    padding: 0 0.25rem;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .household label.toggle {
    flex-direction: row;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  .household label.toggle input {
    width: 1.1rem;
    height: 1.1rem;
    padding: 0;
  }

  .household label.toggle small {
    flex-basis: 100%;
  }

  .household small {
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .household input,
  .household select {
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .save {
    grid-column: 1 / -1;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .primary {
    border: 1px solid var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .primary:hover {
    background: var(--accent);
    opacity: 0.9;
  }

  .primary:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
