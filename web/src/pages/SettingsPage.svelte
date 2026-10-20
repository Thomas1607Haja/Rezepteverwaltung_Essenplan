<script>
  import { api } from '../lib/api.js';

  // Ein Promise direkt im Markup: {#await} zeigt je nach Zustand
  // "lädt", das Ergebnis oder den Fehler an.
  const health = api.health();
</script>

<h1>Einstellungen</h1>
<p class="placeholder">Die Einstellungen folgen in einer späteren Phase.</p>

<section class="card">
  <h2>Server</h2>
  {#await health}
    <p>Verbindung wird geprüft …</p>
  {:then}
    <p class="ok">Verbunden</p>
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

  .ok {
    color: var(--accent);
  }

  .error {
    color: var(--danger);
  }
</style>
