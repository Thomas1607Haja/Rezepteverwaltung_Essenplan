<script>
  import { api } from '../lib/api.js';
  import { formatBytes, formatUptime } from '../lib/format.js';
  import PantryEditor from '../components/PantryEditor.svelte';

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

  function showMessage(text, isError = false) {
    message = text;
    messageIsError = isError;
  }

  async function saveHousehold(event) {
    event.preventDefault();
    saving = true;
    try {
      household = await api.updateSettings({ ...household, family_size: Number(household.family_size) });
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
        <small>Mit ihm beginnt der Einkaufszeitraum (eine Woche)</small>
      </label>
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
