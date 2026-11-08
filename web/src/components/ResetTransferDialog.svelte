<script>
  import { onMount } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import { api } from '../lib/api.js';
  import { dayLabel } from '../lib/format.js';
  import { t, errorText, noteText } from '../lib/i18n.svelte.js';

  // Übertragung zurücksetzen: übertragene Gerichte wieder auf „geplant“
  // setzen, z. B. nach dem Senden an die falsche Bring-Liste. next wählt
  // den Einkaufszeitraum, entryId stattdessen genau ein Gericht. ondone
  // meldet, dass etwas zurückgesetzt wurde (die Seite lädt dann neu),
  // onclose das Schließen.
  let { next = false, entryId = null, ondone, onclose } = $props();

  let preview = $state(null);
  let error = $state('');
  let busy = $state(false);
  let result = $state(null);
  // Gewählte Gerichte; Standard: alle.
  const selected = new SvelteSet();
  let remove = $state(true);
  // Liste für Gerichte ohne gespeicherte Ziel-Liste (vor Version 0.18
  // übertragen); leer = deren Artikel nicht entfernen.
  let fallbackList = $state('');

  let dialog;
  onMount(async () => {
    dialog.showModal();
    try {
      preview = await api.resetPreview(next, entryId);
      for (const e of preview.entries) selected.add(e.id);
    } catch (e) {
      error = errorText(e);
    }
  });

  const chosen = $derived(preview ? preview.entries.filter((e) => selected.has(e.id)) : []);
  const unknown = $derived(chosen.filter((e) => e.unknown_list).length);

  function toggle(id) {
    if (selected.has(id)) selected.delete(id);
    else selected.add(id);
  }

  function day(date) {
    const d = dayLabel(date);
    return `${d.weekday} ${d.date}`;
  }

  /** „an „Testliste““, mehrere mit Komma; unbekannt = vor 0.18 übertragen. */
  function target(e) {
    const names = e.lists.map((l) => t('shopping.reset.list_name', { name: l.name }));
    if (e.unknown_list) names.push(t('shopping.reset.unknown_list'));
    if (names.length === 0) return t('shopping.reset.no_items');
    return t('shopping.reset.target', { lists: names.join(', ') });
  }

  async function confirm() {
    busy = true;
    error = '';
    try {
      result = await api.resetTransfer([...selected], remove, remove ? fallbackList : '');
      ondone?.(result);
    } catch (e) {
      error = errorText(e);
    } finally {
      busy = false;
    }
  }

  /** Fehler beim Laden der Bring-Listen (Code vom Server, sonst englischer Text). */
  const listsError = $derived(
    preview?.lists_error ? noteText({ code: preview.lists_error_code, message: preview.lists_error }) : '',
  );
</script>

<dialog bind:this={dialog} oncancel={onclose} aria-labelledby="reset-title">
  <div class="content">
    <h2 id="reset-title">{t('shopping.reset.title')}</h2>

    {#if result}
      <div role="status" class="result">
        <p>{t('shopping.reset.done', { count: result.entries })}</p>
        {#if result.removed.length > 0}
          <p>{t('shopping.reset.removed', { items: result.removed.join(', ') })}</p>
        {/if}
        {#if result.reduced.length > 0}
          <p>{t('shopping.reset.reduced', { items: result.reduced.join(', ') })}</p>
        {/if}
        {#if result.not_found.length > 0}
          <p class="muted">{t('shopping.reset.not_found', { items: result.not_found.join(', ') })}</p>
        {/if}
        {#if result.skipped.length > 0}
          <p class="muted">{t('shopping.reset.skipped', { items: result.skipped.join(', ') })}</p>
        {/if}
        {#if result.failed.length > 0}
          <p class="error">{t('shopping.reset.failed', { items: result.failed.map((f) => f.name).join(', ') })}</p>
        {/if}
        {#if preview.current_list.entity_id}
          <p>{t('shopping.reset.send_now', { name: preview.current_list.name })}</p>
        {/if}
      </div>
      <footer>
        <button type="button" class="primary" onclick={onclose}>{t('common.close')}</button>
      </footer>
    {:else if !preview}
      <p class={error ? 'error' : 'muted'}>{error || t('common.loading')}</p>
      <footer>
        <button type="button" class="secondary" onclick={onclose}>{t('common.cancel')}</button>
      </footer>
    {:else}
      <p>
        {t('shopping.reset.intro')}
      </p>

      {#if preview.entries.length === 0}
        <p class="muted">{t('shopping.reset.nothing')}</p>
      {:else}
        <ul class="entries">
          {#each preview.entries as e (e.id)}
            <li>
              <label>
                <input type="checkbox" checked={selected.has(e.id)} onchange={() => toggle(e.id)} />
                <span class="title">{e.title}</span>
                <small>{day(e.date)} · {target(e)}{e.items > 0 ? ` · ${t('shopping.reset.items', { count: e.items })}` : ''}</small>
              </label>
            </li>
          {/each}
        </ul>

        <label class="check">
          <input type="checkbox" bind:checked={remove} />
          {t('shopping.reset.remove')}
        </label>
        {#if remove}
          <p class="hint">
            {t('shopping.reset.remove_hint')}
          </p>
          {#if unknown > 0}
            <label class="field">
              <span>
                {t('shopping.reset.unknown_label', { count: unknown })}
              </span>
              <select bind:value={fallbackList}>
                <option value="">{t('shopping.reset.dont_remove')}</option>
                {#each preview.lists as l (l.entity_id)}
                  <option value={l.entity_id}>{l.name}</option>
                {/each}
              </select>
            </label>
            {#if listsError}
              <p class="error">{t('shopping.reset.lists_error', { message: listsError })}</p>
            {/if}
          {/if}
        {/if}

        {#if preview.current_list.entity_id}
          <p class="hint">{t('shopping.reset.resend_hint', { name: preview.current_list.name })}</p>
        {/if}
      {/if}

      {#if error}<p class="error">{error}</p>{/if}

      <footer>
        <button type="button" class="secondary" onclick={onclose} disabled={busy}>{t('common.cancel')}</button>
        <button type="button" class="primary" onclick={confirm} disabled={busy || selected.size === 0}>
          {busy ? t('shopping.reset.resetting') : t('shopping.reset.confirm', { count: selected.size })}
        </button>
      </footer>
    {/if}
  </div>
</dialog>

<style>
  dialog {
    width: min(32rem, calc(100vw - 2rem));
    max-height: calc(100vh - 2rem);
    padding: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
  }

  dialog::backdrop {
    background: rgb(0 0 0 / 0.4);
  }

  .content {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 1rem;
    max-height: calc(100vh - 2rem);
    box-sizing: border-box;
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  p {
    margin: 0;
  }

  .muted {
    color: var(--text-muted);
  }

  .error {
    color: var(--danger);
  }

  .hint {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  /* Lange Listen scrollen, die Knöpfe bleiben sichtbar. */
  .entries {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .entries label {
    display: grid;
    grid-template-columns: auto 1fr;
    column-gap: 0.6rem;
    padding: 0.4rem 0;
    border-top: 1px solid var(--border);
    cursor: pointer;
  }

  .entries li:first-child label {
    border-top: none;
  }

  .entries input {
    grid-row: 1 / 3;
    width: 1.15rem;
    height: 1.15rem;
    accent-color: var(--accent);
  }

  .entries small {
    grid-column: 2;
    color: var(--text-muted);
  }

  .title {
    font-weight: 600;
    overflow-wrap: break-word;
  }

  .check {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .check input {
    width: 1.15rem;
    height: 1.15rem;
    accent-color: var(--accent);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-size: 0.9rem;
  }

  select {
    padding: 0.4rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
  }

  .result {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.6rem 0.75rem;
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--accent-soft);
  }

  footer {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  .primary,
  .secondary {
    padding: 0.5rem 1rem;
    border-radius: var(--radius);
    font: inherit;
    cursor: pointer;
  }

  .primary {
    border: 1px solid var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .secondary {
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
  }

  button:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
