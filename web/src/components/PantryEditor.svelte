<script>
  import { api } from '../lib/api.js';
  import PantryApplyDialog from './PantryApplyDialog.svelte';

  let items = $state(null); // null = lädt noch
  let loadError = $state('');
  let message = $state('');
  let messageIsError = $state(false);

  // Bearbeitete Zeile (ID) bzw. null; draft hält deren Eingaben.
  let editingId = $state(null);
  let draft = $state({ name: '', synonyms: '' });
  // Eingaben für einen neuen Artikel
  let fresh = $state({ name: '', synonyms: '' });
  let busy = $state(false);

  // Vorschau des Übertragens, solange der Dialog offen ist
  let preview = $state(null);

  api
    .listPantry()
    .then((list) => (items = list))
    .catch((e) => (loadError = `Vorratsliste nicht geladen: ${e.message}`));

  function showMessage(text, isError = false) {
    message = text;
    messageIsError = isError;
  }

  /** Synonyme werden als kommagetrenntes Textfeld bearbeitet. */
  function toPayload(input) {
    return { name: input.name, synonyms: input.synonyms.split(',') };
  }

  /** Neuer Stand eines Artikels in die Liste, alphabetisch wie auf dem Server. */
  function upsert(item) {
    const rest = items.filter((it) => it.id !== item.id);
    items = [...rest, item].sort((a, b) => a.name.localeCompare(b.name, 'de'));
  }

  async function add(event) {
    event.preventDefault();
    if (fresh.name.trim() === '') return;
    busy = true;
    try {
      const item = await api.createPantryItem(toPayload(fresh));
      upsert(item);
      fresh = { name: '', synonyms: '' };
      showMessage(`„${item.name}“ hinzugefügt.`);
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      busy = false;
    }
  }

  function startEdit(item) {
    editingId = item.id;
    draft = { name: item.name, synonyms: item.synonyms.join(', ') };
    message = '';
  }

  async function saveEdit(event) {
    event.preventDefault();
    busy = true;
    try {
      upsert(await api.updatePantryItem(editingId, toPayload(draft)));
      editingId = null;
      message = '';
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      busy = false;
    }
  }

  async function remove(item) {
    if (!confirm(`„${item.name}“ von der Vorratsliste nehmen?`)) return;
    busy = true;
    try {
      await api.deletePantryItem(item.id);
      items = items.filter((it) => it.id !== item.id);
      showMessage(`„${item.name}“ entfernt.`);
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      busy = false;
    }
  }

  /** Erst die Vorschau holen; gibt es nichts zu tun, gleich Bescheid geben. */
  async function startApply() {
    busy = true;
    message = '';
    try {
      const result = await api.applyPantry(false);
      if (result.changes.length === 0) showMessage('Alle Rezepte passen schon zur Vorratsliste.');
      else preview = result;
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      busy = false;
    }
  }

  async function finishApply(choice) {
    if (choice !== 'apply') {
      preview = null;
      return;
    }
    busy = true;
    try {
      const result = await api.applyPantry(true);
      const n = result.changes.length;
      showMessage(`${n === 1 ? 'Eine Zutat' : `${n} Zutaten`} in ${result.recipes === 1 ? 'einem Rezept' : `${result.recipes} Rezepten`} aktualisiert.`);
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      busy = false;
      preview = null;
    }
  }
</script>

<section class="card">
  <h2>Vorratsliste</h2>
  <p class="hint">
    Zutaten, die immer im Haus sind. Sie werden in neuen Rezepten automatisch nicht zum Einkaufen vorgemerkt.
  </p>

  {#if loadError}
    <p class="error">{loadError}</p>
  {:else if items === null}
    <p>Lädt …</p>
  {:else}
    <ul class="items">
      {#each items as item (item.id)}
        <li>
          {#if editingId === item.id}
            <form class="row-form" onsubmit={saveEdit}>
              <input aria-label="Name" bind:value={draft.name} required />
              <input aria-label="Synonyme" placeholder="Synonyme, durch Komma getrennt" bind:value={draft.synonyms} />
              <div class="actions">
                <button type="button" onclick={() => (editingId = null)} disabled={busy}>Abbrechen</button>
                <button type="submit" class="primary" disabled={busy}>Speichern</button>
              </div>
            </form>
          {:else}
            <div class="row">
              <span class="text">
                <strong>{item.name}</strong>
                {#if item.synonyms.length > 0}<small>{item.synonyms.join(', ')}</small>{/if}
              </span>
              <button type="button" onclick={() => startEdit(item)} disabled={busy} aria-label="{item.name} bearbeiten">Bearbeiten</button>
              <button type="button" class="remove" onclick={() => remove(item)} disabled={busy} aria-label="{item.name} entfernen">×</button>
            </div>
          {/if}
        </li>
      {:else}
        <li class="empty">Noch nichts im Vorrat.</li>
      {/each}
    </ul>

    <form class="row-form add" onsubmit={add}>
      <input aria-label="Neuer Vorratsartikel" placeholder="Neuer Artikel, z. B. Mehl" bind:value={fresh.name} />
      <input aria-label="Synonyme des neuen Artikels" placeholder="Synonyme, z. B. Weizenmehl, Dinkelmehl" bind:value={fresh.synonyms} />
      <div class="actions">
        <button type="submit" class="primary" disabled={busy || fresh.name.trim() === ''}>Hinzufügen</button>
      </div>
    </form>

    <div class="apply">
      <button type="button" onclick={startApply} disabled={busy}>Auf alle Rezepte übertragen …</button>
      {#if message}<span class:error={messageIsError} class:ok={!messageIsError}>{message}</span>{/if}
    </div>
    <p class="hint">Bestehende Rezepte ändern sich erst mit diesem Knopf. Vorher siehst du, welche Zutaten betroffen sind.</p>
  {/if}
</section>

{#if preview}
  <PantryApplyDialog changes={preview.changes} recipes={preview.recipes} {busy} onchoice={finishApply} />
{/if}

<style>
  .card {
    margin-bottom: 1rem;
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  h2 {
    margin: 0 0 0.25rem;
    font-size: 1.1rem;
  }

  p {
    margin: 0;
  }

  .hint {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .items {
    margin: 0.75rem 0;
    padding: 0;
    list-style: none;
    border-top: 1px solid var(--border);
  }

  .items > li {
    padding: 0.4rem 0;
    border-bottom: 1px solid var(--border);
  }

  .empty {
    color: var(--text-muted);
  }

  .row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    column-gap: 0.6rem;
  }

  .text small {
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  .row-form {
    display: grid;
    grid-template-columns: 1fr;
    gap: 0.4rem;
  }

  @media (min-width: 45rem) {
    .row-form {
      grid-template-columns: 12rem 1fr auto;
      align-items: center;
    }
  }

  .actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  input {
    width: 100%;
    min-width: 0;
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
  }

  button {
    padding: 0.4rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    cursor: pointer;
    white-space: nowrap;
  }

  button:hover {
    background: var(--surface-hover);
  }

  button:disabled {
    opacity: 0.5;
    cursor: default;
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

  .remove {
    width: 2rem;
    height: 2rem;
    padding: 0;
    border: none;
    border-radius: 999px;
    background: none;
    color: var(--text-muted);
    font-size: 1.3rem;
    line-height: 1;
  }

  .remove:hover {
    color: var(--danger);
  }

  .apply {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin: 1rem 0 0.35rem;
  }

  .error {
    color: var(--danger);
  }

  .ok {
    color: var(--accent);
  }
</style>
