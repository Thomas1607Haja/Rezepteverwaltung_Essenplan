<script>
  import { api } from '../lib/api.js';
  import { setupUri } from '../lib/appSetup.js';
  import QrCode from './QrCode.svelte';
  import AppInstallStep from './AppInstallStep.svelte';

  // API-Schlüssel für den App-Zugang (Android-App): einer pro Gerät. Der
  // Schlüssel selbst ist nur direkt nach dem Anlegen sichtbar; der Server
  // speichert nur seinen Hash.
  let keys = $state(null); // null = lädt noch
  let loadError = $state('');
  let message = $state('');
  let messageIsError = $state(false);
  let name = $state('');
  let busy = $state(false);
  // Gerätename wird abgefragt; ein Schlüssel entsteht erst beim Bestätigen.
  let connecting = $state(false);
  // Gerade angelegter Schlüssel: { name, key } bis zum Schließen des Hinweises
  let created = $state(null);
  let copied = $state(false);

  // Adresse, unter der die App den App-Zugang erreicht (Cloudflare-Tunnel).
  // Sie steht zusammen mit dem Schlüssel im QR-Code.
  let appUrl = $state('');
  let savedAppUrl = $state('');
  let urlMessage = $state('');
  let urlIsError = $state(false);
  // Optional: Service Token von Cloudflare Access. Wird nicht gespeichert,
  // nur in den QR-Code übernommen.
  let cfId = $state('');
  let cfSecret = $state('');
  let showCf = $state(false);

  api
    .getSettings()
    .then((s) => (appUrl = savedAppUrl = s.app_url ?? ''))
    .catch((e) => (loadError = `Einstellungen nicht geladen: ${e.message}`));

  async function saveAppUrl(event) {
    event.preventDefault();
    try {
      const s = await api.updateSettings({ app_url: appUrl });
      appUrl = savedAppUrl = s.app_url;
      urlMessage = 'Gespeichert.';
      urlIsError = false;
    } catch (e) {
      urlMessage = e.message;
      urlIsError = true;
    }
  }

  const qrText = $derived(
    created && savedAppUrl ? setupUri({ url: savedAppUrl, key: created.key, cfId, cfSecret }) : '',
  );

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
      connecting = false;
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
  <h2>App installieren</h2>
  <p class="hint">
    Ein neues Handy in zwei Schritten einrichten: Android-App installieren, dann mit diesem Rezeptplaner verbinden. Jedes
    Gerät bekommt einen eigenen Schlüssel; ohne gültigen Schlüssel lehnt der App-Zugang jede Anfrage ab.
  </p>

  <AppInstallStep />

  <h3>Schritt 2 – App verbinden</h3>

  <form class="row url" onsubmit={saveAppUrl}>
    <label for="app-url">Adresse für die App</label>
    <input
      id="app-url"
      type="url"
      placeholder="https://rezepte.example.org"
      bind:value={appUrl}
      oninput={() => (urlMessage = '')}
    />
    <button type="submit" disabled={appUrl.trim() === savedAppUrl}>Speichern</button>
  </form>
  <p class="hint">Hostname im Cloudflare-Tunnel, der auf Port 8100 zeigt (Anleitung in docs/APP-ZUGANG.md). Einmalig eintragen.</p>
  {#if urlMessage}<p class:error={urlIsError} class:ok={!urlIsError}>{urlMessage}</p>{/if}

  {#if created}
    <div class="created" role="status">
      <strong>„{created.name}“ verbinden</strong>
      <p>
        In der App „Einrichten“ wählen und diesen Code scannen. <strong>Er wird nur jetzt angezeigt</strong> – nach
        „Fertig“ oder beim Verlassen der Seite ist er weg (dann einfach ein neues Gerät verbinden).
      </p>
      {#if qrText}
        <div class="qr">
          <QrCode text={qrText} label="QR-Code zum Verbinden der App" />
        </div>
      {/if}
      <details bind:open={showCf}>
        <summary>Cloudflare Access (optional)</summary>
        <p class="hint">
          Nur nötig, wenn der App-Hostname mit Cloudflare Access geschützt ist. Die Angaben kommen nur in den QR-Code und
          werden nicht gespeichert.
        </p>
        <div class="cf">
          <input type="text" placeholder="Client ID (….access)" bind:value={cfId} autocomplete="off" />
          <input type="password" placeholder="Client Secret" bind:value={cfSecret} autocomplete="off" />
        </div>
      </details>
      <details>
        <summary>Schlüssel als Text</summary>
        <code>{created.key}</code>
        <button type="button" onclick={copy}>{copied ? 'Kopiert' : 'Kopieren'}</button>
      </details>
      <div class="row">
        <button type="button" class="primary" onclick={() => (created = null)}>Fertig</button>
      </div>
    </div>
  {:else if connecting}
    <form class="row" onsubmit={add}>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        type="text"
        maxlength="60"
        placeholder="Gerät, z. B. Handy Thomas"
        autofocus
        bind:value={name}
        oninput={() => (message = '')}
      />
      <button type="submit" class="primary" disabled={busy || name.trim() === ''}>QR-Code erzeugen</button>
      <button type="button" onclick={() => (connecting = false)}>Abbrechen</button>
    </form>
  {:else}
    <div class="row">
      <button type="button" class="primary" disabled={!savedAppUrl} onclick={() => (connecting = true)}>
        Neues Gerät verbinden
      </button>
      {#if !savedAppUrl}<span class="hint">Zuerst die Adresse für die App speichern.</span>{/if}
    </div>
  {/if}

  <h3>Verbundene Geräte</h3>
  {#if loadError}
    <p class="error">{loadError}</p>
  {:else if keys === null}
    <p>Lädt …</p>
  {:else if keys.length > 0}
    <ul>
      {#each keys as key (key.id)}
        <li>
          <span class="name">{key.name}</span>
          <small>verbunden {formatDate(key.created_at)} · zuletzt benutzt {formatDate(key.last_used_at)}</small>
          <button type="button" class="danger" onclick={() => remove(key)}>Sperren</button>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="muted">Noch kein Gerät verbunden.</p>
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

  h3 {
    margin: 1.25rem 0 0.5rem;
    font-size: 1rem;
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

  .url {
    align-items: center;
    margin-bottom: 0.25rem;
  }

  .url label {
    flex-basis: 100%;
    font-weight: 600;
  }

  .qr {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin: 0.5rem 0;
  }

  .qr p {
    flex: 1 1 10rem;
  }

  details {
    margin-bottom: 0.75rem;
  }

  summary {
    cursor: pointer;
  }

  .cf {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
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
