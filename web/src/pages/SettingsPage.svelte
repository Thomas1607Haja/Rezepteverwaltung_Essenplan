<script>
  import { api } from '../lib/api.js';
  import { formatBytes, formatUptime } from '../lib/format.js';
  import { t, errorText, currentLocale, languageInfo, applyLanguage } from '../lib/i18n.svelte.js';
  import PantryEditor from '../components/PantryEditor.svelte';
  import ApiKeysEditor from '../components/ApiKeysEditor.svelte';
  import TagManager from '../components/TagManager.svelte';
  import { supportUrl } from '../lib/support.js';
  import { myLinks } from '../lib/myLinks.js';

  // Ein Promise direkt im Markup: {#await} zeigt je nach Zustand
  // "lädt", das Ergebnis oder den Fehler an. Neu zuweisen lädt neu.
  let system = $state(api.system());

  function reload() {
    system = api.system();
  }

  // Haushalt: Familiengröße und Einkaufstag. Gespeichert werden englische
  // Schlüssel; die Namen der Wochentage liefert Intl in der aktuellen Sprache
  // (1. 1. 2024 war ein Montag).
  const WEEKDAYS = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'];
  const weekdays = $derived.by(() => {
    const fmt = new Intl.DateTimeFormat(currentLocale(), { weekday: 'long' });
    return WEEKDAYS.map((value, i) => [value, fmt.format(new Date(2024, 0, 1 + i))]);
  });

  // Sprachen der Oberfläche, jeweils im eigenen Namen (nicht übersetzt)
  const LANGUAGE_NAMES = [
    ['en', 'English'],
    ['de', 'Deutsch'],
    ['fr', 'Français'],
    ['es', 'Español'],
    ['it', 'Italiano'],
  ];

  // Sprache, die Home Assistant meldet, als Name in der aktuellen Sprache
  const haLanguage = $derived.by(() => {
    const code = languageInfo().homeAssistant;
    if (!code) return '';
    try {
      return new Intl.DisplayNames(currentLocale(), { type: 'language' }).of(code) ?? code;
    } catch {
      return code;
    }
  });

  let household = $state(null);
  let saved = ''; // Stand beim Laden bzw. letzten Speichern
  let message = $state('');
  let messageIsError = $state(false);
  let saving = $state(false);

  api
    .getSettings()
    .then((s) => {
      s.ui_language ||= 'auto';
      household = s;
      saved = JSON.stringify(s);
    })
    .catch((e) => showMessage(t('settings.load_failed', { message: errorText(e) }), true));

  const changed = $derived(household !== null && JSON.stringify(household) !== saved);

  // To-do-Listen aus Home Assistant für die Auswahl der Bring-Liste
  let ha = $state(null);
  api
    .homeAssistantStatus()
    .then((s) => (ha = s))
    .catch((e) => (ha = { mode: '', lists: [], error: errorText(e), error_code: '' }));

  /** Fehlertext zum Status von Home Assistant: übersetzter Code, sonst der Text. */
  function haError(status) {
    return status.error_code ? errorText({ code: status.error_code, message: status.error }) : status.error;
  }

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
        ui_language: household.ui_language,
      });
      household.ui_language ||= 'auto';
      saved = JSON.stringify(household);
      // Neue Sprache sofort anwenden, ohne die Seite neu zu laden
      await applyLanguage(await api.language());
      showMessage(t('common.saved'));
    } catch (e) {
      showMessage(errorText(e), true);
    } finally {
      saving = false;
    }
  }
</script>

<h1>{t('settings.title')}</h1>

<section class="card">
  <h2>{t('settings.household.title')}</h2>
  {#if household}
    <form class="household" onsubmit={saveHousehold}>
      <label>
        <span>{t('settings.household.family_size')}</span>
        <input type="number" min="1" max="50" bind:value={household.family_size} oninput={() => (message = '')} />
        <small>{t('settings.household.family_size_hint')}</small>
      </label>
      <label>
        <span>{t('settings.household.shopping_day')}</span>
        <select bind:value={household.shopping_day} onchange={() => (message = '')}>
          {#each weekdays as [value, name] (value)}
            <option {value}>{name}</option>
          {/each}
        </select>
        <small>{t('settings.household.shopping_day_hint')}</small>
      </label>
      <label>
        <span>{t('settings.household.shopping_time')}</span>
        <input type="time" step="60" bind:value={household.shopping_time} oninput={() => (message = '')} />
        <small>{t('settings.household.shopping_time_hint')}</small>
      </label>
      <label>
        <span>{t('settings.household.bring_list')}</span>
        <select bind:value={household.bring_entity} onchange={() => (message = '')} disabled={!ha}>
          <option value="">{t('settings.household.bring_none')}</option>
          {#each ha?.lists ?? [] as list (list.entity_id)}
            <option value={list.entity_id}>{list.name}</option>
          {/each}
          {#if missingEntity}<option value={missingEntity}>{t('settings.household.bring_missing', { entity: missingEntity })}</option>{/if}
        </select>
        {#if !ha}
          <small>{t('settings.household.lists_loading')}</small>
        {:else if ha.error || ha.error_code}
          <small class="error">{t('common.error', { message: haError(ha) })}</small>
        {:else if ha.mode === 'test'}
          <small>{t('settings.household.test_mode')}</small>
        {:else if ha.lists.length === 0}
          <small>{t('settings.household.no_lists')}</small>
        {:else}
          <small>{t('settings.household.bring_hint')}</small>
        {/if}
      </label>
      <label>
        <span>{t('settings.household.language')}</span>
        <select bind:value={household.ui_language} onchange={() => (message = '')}>
          <option value="auto">{t('settings.household.language_auto')}</option>
          {#each LANGUAGE_NAMES as [value, name] (value)}
            <option {value} lang={value}>{name}</option>
          {/each}
        </select>
        <small>
          {haLanguage
            ? t('settings.household.ha_language', { language: haLanguage })
            : t('settings.household.ha_language_unknown')}
        </small>
      </label>
      <label class="toggle">
        <input type="checkbox" bind:checked={household.plan_preview} onchange={() => (message = '')} />
        <span>{t('settings.household.plan_preview')}</span>
        <small>{t('settings.household.plan_preview_hint')}</small>
      </label>
      <label class="toggle">
        <input type="checkbox" bind:checked={household.coffee_slot} onchange={() => (message = '')} />
        <span>{t('settings.household.coffee_slot')}</span>
        <small>{t('settings.household.coffee_slot_hint')}</small>
      </label>
      <fieldset class="times">
        <legend>{t('settings.household.times')}</legend>
        <label><span>{t('settings.household.time_lunch')}</span><input type="time" step="60" bind:value={household.lunch_time} oninput={() => (message = '')} /></label>
        {#if household.coffee_slot}
          <label><span>{t('settings.household.time_coffee')}</span><input type="time" step="60" bind:value={household.coffee_time} oninput={() => (message = '')} /></label>
        {/if}
        <label><span>{t('settings.household.time_dinner')}</span><input type="time" step="60" bind:value={household.dinner_time} oninput={() => (message = '')} /></label>
      </fieldset>
      <div class="save">
        <button type="submit" class="primary" disabled={!changed || saving}>{saving ? t('common.saving') : t('common.save')}</button>
        {#if message}<span class:error={messageIsError} class:ok={!messageIsError}>{message}</span>{/if}
      </div>
    </form>
  {:else if message}
    <p class="error">{message}</p>
  {:else}
    <p>{t('common.loading')}</p>
  {/if}
</section>

<PantryEditor />

<TagManager />

<ApiKeysEditor />

<!-- My-Home-Assistant-Links: öffnen my.home-assistant.io im neuen Tab, das zur
     passenden Seite im eigenen Home Assistant weiterleitet. -->
<section class="card">
  <h2>{t('settings.home_assistant.title')}</h2>
  <p class="hint">{t('settings.home_assistant.text')}</p>
  <h3>{t('settings.home_assistant.integration_title')}</h3>
  <p>{t('settings.home_assistant.integration_text')}</p>
  <div class="links">
    <a class="support" href={myLinks.integration_download} target="_blank" rel="noopener noreferrer">{t('settings.home_assistant.integration_download')}</a>
    <a class="support" href={myLinks.integration_setup} target="_blank" rel="noopener noreferrer">{t('settings.home_assistant.integration_setup')}</a>
  </div>
  <h3>{t('settings.home_assistant.app_title')}</h3>
  <p>{t('settings.home_assistant.app_text')}</p>
  <div class="links">
    <a class="support" href={myLinks.app_install} target="_blank" rel="noopener noreferrer">{t('settings.home_assistant.app_install')}</a>
    <a class="support" href={myLinks.app_repository} target="_blank" rel="noopener noreferrer">{t('settings.home_assistant.app_repository')}</a>
  </div>
</section>

<section class="card">
  <h2>{t('settings.system.title')}</h2>
  {#await system}
    <p>{t('settings.system.checking')}</p>
  {:then info}
    <dl>
      <dt>{t('settings.system.version')}</dt>
      <dd>{info.version}</dd>
      <dt>{t('settings.system.uptime')}</dt>
      <dd>{formatUptime(info.uptime_seconds)}</dd>
      <dt>{t('settings.system.memory')}</dt>
      <dd>{formatBytes(info.memory_bytes)}</dd>
      <dt>{t('settings.system.memory_peak')}</dt>
      <dd>{formatBytes(info.memory_peak_bytes)}</dd>
      <dt>{t('settings.system.database')}</dt>
      <dd>{formatBytes(info.database_bytes)}</dd>
      <dt>{t('settings.system.images')}</dt>
      <dd>{t('settings.system.images_value', { size: formatBytes(info.images_bytes), count: info.images_count })}</dd>
    </dl>
    <button type="button" onclick={reload}>{t('common.refresh')}</button>
  {:catch error}
    <p class="error">{t('common.not_connected', { message: errorText(error) })}</p>
  {/await}
</section>

<section class="card">
  <h2>{t('settings.support.title')}</h2>
  <p>{t('settings.support.text')}</p>
  <!-- Neuer Tab: Die Oberfläche läuft im Ingress-Rahmen von Home Assistant. -->
  <a class="support" href={supportUrl} target="_blank" rel="noopener noreferrer">☕ {t('settings.support.button')}</a>
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

  /* Link im Stil der Knöpfe dieser Seite */
  .support {
    display: inline-block;
    margin-top: 0.75rem;
    padding: 0.4rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    text-decoration: none;
  }

  .links {
    display: flex;
    flex-wrap: wrap;
    gap: 0 0.5rem;
  }

  h3 {
    margin: 1rem 0 0.25rem;
    font-size: 1rem;
  }

  .hint {
    color: var(--text-muted);
  }

  .support:hover {
    background: var(--surface-hover);
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
