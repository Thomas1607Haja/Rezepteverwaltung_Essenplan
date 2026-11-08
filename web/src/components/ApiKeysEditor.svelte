<script>
  import { api } from '../lib/api.js';
  import { setupUri } from '../lib/appSetup.js';
  import { t, errorText, noteText, currentLocale } from '../lib/i18n.svelte.js';
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
    .catch((e) => (loadError = t('devices.settings_load_failed', { message: errorText(e) })));

  let checking = $state(false);

  async function saveAppUrl(event) {
    event.preventDefault();
    try {
      const s = await api.updateSettings({ app_url: appUrl });
      appUrl = savedAppUrl = s.app_url;
      urlMessage = t('common.saved');
      urlIsError = false;
    } catch (e) {
      urlMessage = errorText(e);
      urlIsError = true;
      return;
    }
    // Gleich prüfen, ob dort der App-Zugang antwortet – ein falscher
    // Hostname fällt sonst erst am Handy auf.
    await checkAppUrl();
  }

  async function checkAppUrl() {
    checking = true;
    urlMessage = t('devices.checking_url');
    urlIsError = false;
    try {
      const result = await api.checkAppUrl(appUrl);
      urlMessage = noteText(result);
      urlIsError = !result.ok;
    } catch (e) {
      urlMessage = errorText(e);
      urlIsError = true;
    } finally {
      checking = false;
    }
  }

  const qrText = $derived(
    created && savedAppUrl ? setupUri({ url: savedAppUrl, key: created.key, cfId, cfSecret }) : '',
  );

  api
    .listApiKeys()
    .then((list) => (keys = list))
    .catch((e) => (loadError = t('devices.keys_load_failed', { message: errorText(e) })));

  function showMessage(text, isError = false) {
    message = text;
    messageIsError = isError;
  }

  function formatDate(value) {
    return value
      ? new Date(value).toLocaleString(currentLocale(), { dateStyle: 'medium', timeStyle: 'short' })
      : t('devices.never');
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
      showMessage(errorText(e), true);
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
      showMessage(t('devices.copy_failed'), true);
    }
  }

  async function remove(key) {
    if (!confirm(t('devices.revoke_confirm', { name: key.name }))) return;
    try {
      await api.deleteApiKey(key.id);
      keys = keys.filter((k) => k.id !== key.id);
      if (created?.name === key.name) created = null;
      showMessage(t('devices.revoked', { name: key.name }));
    } catch (e) {
      showMessage(errorText(e), true);
    }
  }
</script>

<section class="card">
  <h2>{t('devices.title')}</h2>
  <p class="hint">{t('devices.intro')}</p>

  <AppInstallStep />

  <h3>{t('devices.step2')}</h3>

  <form class="row url" onsubmit={saveAppUrl}>
    <label for="app-url">{t('devices.app_url')}</label>
    <input
      id="app-url"
      type="url"
      placeholder="https://rezepte.example.org"
      bind:value={appUrl}
      oninput={() => (urlMessage = '')}
    />
    <button type="submit" disabled={appUrl.trim() === savedAppUrl}>{t('common.save')}</button>
    <button type="button" disabled={checking || appUrl.trim() === '' || appUrl.trim() !== savedAppUrl} onclick={checkAppUrl}
      >{t('devices.check_url')}</button
    >
  </form>
  <p class="hint">{t('devices.app_url_hint')}</p>
  {#if urlMessage}<p class:error={urlIsError} class:ok={!urlIsError}>{urlMessage}</p>{/if}

  {#if created}
    <div class="created" role="status">
      <strong>{t('devices.connect_title', { name: created.name })}</strong>
      <p>
        {t('devices.connect_scan')} <strong>{t('devices.connect_once')}</strong>
        {t('devices.connect_gone')}
      </p>
      {#if qrText}
        <div class="qr">
          <QrCode text={qrText} label={t('devices.qr_label')} />
        </div>
      {/if}
      <details bind:open={showCf}>
        <summary>{t('devices.cf_summary')}</summary>
        <p class="hint">{t('devices.cf_hint')}</p>
        <div class="cf">
          <input type="text" placeholder={t('devices.cf_id')} bind:value={cfId} autocomplete="off" />
          <input type="password" placeholder={t('devices.cf_secret')} bind:value={cfSecret} autocomplete="off" />
        </div>
      </details>
      <details>
        <summary>{t('devices.key_as_text')}</summary>
        <code>{created.key}</code>
        <button type="button" onclick={copy}>{copied ? t('devices.copied') : t('devices.copy')}</button>
      </details>
      <div class="row">
        <button type="button" class="primary" onclick={() => (created = null)}>{t('devices.done')}</button>
      </div>
    </div>
  {:else if connecting}
    <form class="row" onsubmit={add}>
      <!-- svelte-ignore a11y_autofocus -->
      <input
        type="text"
        maxlength="60"
        placeholder={t('devices.device_placeholder')}
        autofocus
        bind:value={name}
        oninput={() => (message = '')}
      />
      <button type="submit" class="primary" disabled={busy || name.trim() === ''}>{t('devices.create_qr')}</button>
      <button type="button" onclick={() => (connecting = false)}>{t('common.cancel')}</button>
    </form>
  {:else}
    <div class="row">
      <button type="button" class="primary" disabled={!savedAppUrl} onclick={() => (connecting = true)}>
        {t('devices.connect_new')}
      </button>
      {#if !savedAppUrl}<span class="hint">{t('devices.save_url_first')}</span>{/if}
    </div>
  {/if}

  <h3>{t('devices.connected')}</h3>
  {#if loadError}
    <p class="error">{loadError}</p>
  {:else if keys === null}
    <p>{t('common.loading')}</p>
  {:else if keys.length > 0}
    <ul>
      {#each keys as key (key.id)}
        <li>
          <span class="name">{key.name}</span>
          <small>{t('devices.key_dates', { created: formatDate(key.created_at), used: formatDate(key.last_used_at) })}</small>
          <button type="button" class="danger" onclick={() => remove(key)}>{t('devices.revoke')}</button>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="muted">{t('devices.none')}</p>
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
    overflow-wrap: break-word;
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
