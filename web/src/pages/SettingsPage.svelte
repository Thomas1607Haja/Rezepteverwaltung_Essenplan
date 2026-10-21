<script>
  import { api } from '../lib/api.js';
  import { formatBytes, formatUptime } from '../lib/format.js';

  // Ein Promise direkt im Markup: {#await} zeigt je nach Zustand
  // "lädt", das Ergebnis oder den Fehler an. Neu zuweisen lädt neu.
  let system = $state(api.system());

  function reload() {
    system = api.system();
  }
</script>

<h1>Einstellungen</h1>
<p class="placeholder">Die Einstellungen folgen in einer späteren Phase.</p>

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
  .placeholder {
    color: var(--text-muted);
  }

  .card {
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
</style>
