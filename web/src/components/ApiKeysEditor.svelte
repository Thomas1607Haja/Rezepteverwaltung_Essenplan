<script>
  import { api } from '../lib/api.js';

  // API-Schlüssel für den App-Zugang (Android-App): einer pro Gerät. Der
  // Schlüssel selbst ist nur direkt nach dem Anlegen sichtbar; der Server
  // speichert nur seinen Hash.
  let keys = $state(null); // null = lädt noch
  let loadError = $state('');
  let message = $state('');
  let messageIsError = $state(false);
  let name = $state('');
  let busy = $state(false);
  // Gerade angelegter Schlüssel: { name, key } bis zum Schließen des Hinweises
  let created = $state(null);
  let copied = $state(false);

  api
    .listApiKeys()
    .then((list) => (keys = list))
    .catch((e) => (loadError = `Schlüssel nicht geladen: ${e.message}`));

  function showMessage(text, isError = false) {
    message = text;
    messageIsError = isError;
  }

  function formatDate(value) {
    return value ? new Date(value).toLocaleString('de-DE', { dateStyle: 'medium', timeStyle: 'short' }) : 'noch nie';
  }

  async function add(event) {
    event.preventDefault();
    if (name.trim() === '') return;
    busy = true;
    try {
      const { key, ...info } = await api.createApiKey(name);
      keys = [info, ...keys];
      created = { name: info.name, key };
      copied = false;
      name = '';
      message = '';
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      busy = false;
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(created.key);
      copied = true;
    } catch {
      // Ohne Zugriff auf die Zwischenablage (z. B. ohne HTTPS) bleibt
      // markieren und kopieren von Hand.
      showMessage('Kopieren nicht möglich – bitte den Schlüssel markieren und kopieren.', true);
    }
  }

  async function remove(key) {
    if (!confirm(`Schlüssel „${key.name}“ sperren? Das Gerät hat danach keinen Zugriff mehr.`)) return;
    try {
      await api.deleteApiKey(key.id);
      keys = keys.filter((k) => k.id !== key.id);
      if (created?.name === key.name) created = null;
      showMessage(`„${key.name}“ gesperrt.`);
    } catch (e) {
      showMessage(e.message, true);
    }
  }
</script>

<section class="card">
  <h2>App-Zugang</h2>
  <p class="hint">
    Die Android-App greift mit einem eigenen Schlüssel pro Gerät auf den Rezeptplaner zu. Ohne gültigen Schlüssel lehnt
    der App-Zugang jede Anfrage ab.
  </p>

  {#if created}
    <div class="created" role="status">
      <strong>Schlüssel für „{created.name}“</strong>
      <p>Jetzt in die App übernehmen – er wird nur dieses eine Mal angezeigt.</p>
      <code>{created.key}</code>
      <div class="row">
        <button type="button" onclick={copy}>{copied ? 'Kopiert' : 'Kopieren'}</button>
        <button type="button" onclick={() => (created = null)}>Fertig</button>
      </div>
    </div>
  {/if}

  {#if loadError}
    <p class="error">{loadError}</p>
  {:else if keys === null}
    <p>Lädt …</p>
  {:else}
    {#if keys.length > 0}
      <ul>
        {#each keys as key (key.id)}
          <li>
            <span class="name">{key.name}</span>
            <small>angelegt {formatDate(key.created_at)} · zuletzt benutzt {formatDate(key.last_used_at)}</small>
            <button type="button" class="danger" onclick={() => remove(key)}>Sperren</button>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted">Noch kein Schlüssel angelegt.</p>
    {/if}

    <form class="row" onsubmit={add}>
      <input type="text" maxlength="60" placeholder="Gerät, z. B. Handy Thomas" bind:value={name} oninput={() => (message = '')} />
      <button type="submit" class="primary" disabled={busy || name.trim() === ''}>Schlüssel anlegen</button>
    </form>
  {/if}
  {#if message}<p class:error={messageIsError} class:ok={!messageIsError}>{message}</p>{/if}
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

  p {
    margin: 0 0 0.75rem;
  }

  .hint,
  .muted,
  small {
    color: var(--text-muted);
  }

  .hint {
    font-size: 0.9rem;
  }

  ul {
    list-style: none;
    margin: 0 0 0.75rem;
    padding: 0;
  }

  li {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 0 0.75rem;
    align-items: center;
    padding: 0.5rem 0;
    border-top: 1px solid var(--border);
  }

  li:first-child {
    border-top: none;
  }

  li small {
    grid-column: 1;
    font-size: 0.8rem;
  }

  li button {
    grid-column: 2;
    grid-row: 1 / span 2;
  }

  .name {
    font-weight: 600;
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  input {
    flex: 1 1 14rem;
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .created {
    margin-bottom: 1rem;
    padding: 0.75rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
  }

  .created p {
    margin: 0.25rem 0 0.5rem;
    font-size: 0.9rem;
  }

  code {
    display: block;
    margin-bottom: 0.5rem;
    padding: 0.5rem;
    overflow-wrap: anywhere;
    user-select: all;
    background: var(--surface-hover);
    border-radius: calc(var(--radius) - 0.2rem);
  }

  button {
    padding: 0.4rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
  }

  button:hover {
    background: var(--surface-hover);
  }

  .primary {
    border-color: var(--accent);
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

  .danger {
    color: var(--danger);
  }

  .error {
    color: var(--danger);
  }

  .ok {
    color: var(--accent);
  }
</style>
