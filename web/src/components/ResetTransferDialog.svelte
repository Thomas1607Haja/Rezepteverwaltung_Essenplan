<script>
  import { onMount } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import { api } from '../lib/api.js';
  import { dayLabel } from '../lib/format.js';

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
      error = e.message;
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
    const names = e.lists.map((l) => `„${l.name}“`);
    if (e.unknown_list) names.push('unbekannte Liste');
    if (names.length === 0) return 'keine Artikel gesendet';
    return `an ${names.join(', ')}`;
  }

  async function confirm() {
    busy = true;
    error = '';
    try {
      result = await api.resetTransfer([...selected], remove, remove ? fallbackList : '');
      ondone?.(result);
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }

  function count(n, one, many) {
    return n === 1 ? `1 ${one}` : `${n} ${many}`;
  }
</script>

<dialog bind:this={dialog} oncancel={onclose} aria-labelledby="reset-title">
  <div class="content">
    <h2 id="reset-title">Übertragung zurücksetzen</h2>

    {#if result}
      <div role="status" class="result">
        <p>{count(result.entries, 'Gericht', 'Gerichte')} wieder geplant.</p>
        {#if result.removed.length > 0}
          <p>Von der alten Liste entfernt: {result.removed.join(', ')}.</p>
        {/if}
        {#if result.reduced.length > 0}
          <p>Menge verringert (andere Gerichte brauchen sie noch): {result.reduced.join(', ')}.</p>
        {/if}
        {#if result.not_found.length > 0}
          <p class="muted">Nicht mehr offen auf der Liste, unverändert: {result.not_found.join(', ')}.</p>
        {/if}
        {#if result.skipped.length > 0}
          <p class="muted">Nicht entfernt, Liste unbekannt: {result.skipped.join(', ')}.</p>
        {/if}
        {#if result.failed.length > 0}
          <p class="error">Nicht entfernt (bitte von Hand löschen): {result.failed.map((f) => f.name).join(', ')}.</p>
        {/if}
        {#if preview.current_list.entity_id}
          <p>Jetzt „An Bring senden“ – es geht an „{preview.current_list.name}“.</p>
        {/if}
      </div>
      <footer>
        <button type="button" class="primary" onclick={onclose}>Schließen</button>
      </footer>
    {:else if !preview}
      <p class={error ? 'error' : 'muted'}>{error || 'Lädt …'}</p>
      <footer>
        <button type="button" class="secondary" onclick={onclose}>Abbrechen</button>
      </footer>
    {:else}
      <p>
        Die Gerichte gelten danach wieder als geplant und lassen sich neu an Bring senden. Abgewählte Zutaten
        („Zutaten vorhanden“) bleiben erhalten.
      </p>

      {#if preview.entries.length === 0}
        <p class="muted">In diesem Zeitraum ist nichts übertragen.</p>
      {:else}
        <ul class="entries">
          {#each preview.entries as e (e.id)}
            <li>
              <label>
                <input type="checkbox" checked={selected.has(e.id)} onchange={() => toggle(e.id)} />
                <span class="title">{e.title}</span>
                <small>{day(e.date)} · {target(e)}{e.items > 0 ? ` · ${count(e.items, 'Artikel', 'Artikel')}` : ''}</small>
              </label>
            </li>
          {/each}
        </ul>

        <label class="check">
          <input type="checkbox" bind:checked={remove} />
          Artikel aus der alten Liste entfernen
        </label>
        {#if remove}
          <p class="hint">
            Was andere, weiter übertragene Gerichte brauchen, bleibt mit kleinerer Menge stehen. Schon abgehakte Artikel
            bleiben unberührt.
          </p>
          {#if unknown > 0}
            <label class="field">
              <span>
                Für {count(unknown, 'Gericht', 'Gerichte')} ist die damalige Liste nicht gespeichert (vor Version 0.18
                gesendet). Entfernen aus:
              </span>
              <select bind:value={fallbackList}>
                <option value="">– nicht entfernen –</option>
                {#each preview.lists as l (l.entity_id)}
                  <option value={l.entity_id}>{l.name}</option>
                {/each}
              </select>
            </label>
            {#if preview.lists_error}
              <p class="error">Home Assistant nicht erreichbar – Listen nicht geladen.</p>
            {/if}
          {/if}
        {/if}

        {#if preview.current_list.entity_id}
          <p class="hint">Neu senden geht danach an „{preview.current_list.name}“ (Einstellungen).</p>
        {/if}
      {/if}

      {#if error}<p class="error">{error}</p>{/if}

      <footer>
        <button type="button" class="secondary" onclick={onclose} disabled={busy}>Abbrechen</button>
        <button type="button" class="primary" onclick={confirm} disabled={busy || selected.size === 0}>
          {busy ? 'Setzt zurück …' : `Zurücksetzen (${selected.size})`}
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
