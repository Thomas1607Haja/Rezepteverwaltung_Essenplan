<script>
  import { imageUrl, MAX_IMAGE_BYTES } from '../lib/api.js';
  import Icon from './Icon.svelte';

  // current: Dateiname des gespeicherten Bildes (leer = keins).
  // file: neu gewählte, noch nicht hochgeladene Datei (oder null).
  // remove: gespeichertes Bild beim Speichern entfernen.
  // remoteUrl: Bild einer Quellseite (URL-Import), das der Server beim
  // Speichern lädt; eine gewählte Datei hat Vorrang.
  let { current = '', file = $bindable(null), remove = $bindable(false), remoteUrl = $bindable('') } = $props();

  let error = $state('');

  // Vorschau einer gewählten Datei: URL.createObjectURL macht sie für <img>
  // ansprechbar, ohne sie hochzuladen. Die zurückgegebene Funktion gibt den
  // Speicher wieder frei, wenn sich die Datei ändert oder die Seite schließt.
  let fileUrl = $state('');
  $effect(() => {
    if (!file) {
      fileUrl = '';
      return;
    }
    const url = URL.createObjectURL(file);
    fileUrl = url;
    return () => URL.revokeObjectURL(url);
  });

  const preview = $derived(fileUrl || remoteUrl || (current && !remove ? imageUrl(current) : ''));

  function choose(event) {
    const chosen = event.currentTarget.files[0];
    event.currentTarget.value = ''; // dieselbe Datei später erneut wählbar
    if (!chosen) return;
    if (!chosen.type.startsWith('image/')) {
      error = 'Bitte eine Bilddatei wählen.';
      return;
    }
    if (chosen.size > MAX_IMAGE_BYTES) {
      error = 'Die Datei ist zu groß (höchstens 20 MB).';
      return;
    }
    error = '';
    file = chosen;
    remoteUrl = '';
    remove = false;
  }

  function clear() {
    file = null;
    remoteUrl = '';
    remove = Boolean(current);
    error = '';
  }
</script>

<div class="picker">
  <div class="preview">
    {#if preview}
      <!-- no-referrer: Die Quellseite erfährt nicht, wer ihr Bild anzeigt. -->
      <img src={preview} alt="Vorschau des Rezeptbildes" referrerpolicy="no-referrer" />
    {:else}
      <Icon name="dish" size={48} />
    {/if}
  </div>

  <div class="controls">
    <!-- Das eigentliche Dateifeld ist unsichtbar; das Label dient als Knopf.
         accept="image/*" bietet auf dem Handy auch die Kamera an. -->
    <label class="button">
      {preview ? 'Anderes Bild wählen' : 'Bild wählen'}
      <input type="file" accept="image/*" onchange={choose} />
    </label>
    {#if preview}
      <button type="button" class="button secondary" onclick={clear}>Bild entfernen</button>
    {/if}
    {#if file}
      <p class="hint">Wird beim Speichern hochgeladen und verkleinert.</p>
    {:else if remoteUrl}
      <p class="hint">Bild der Quellseite – wird beim Speichern übernommen.</p>
    {:else if remove}
      <p class="hint">Wird beim Speichern entfernt.</p>
    {/if}
    {#if error}
      <p class="error">{error}</p>
    {/if}
  </div>
</div>

<style>
  .picker {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    align-items: flex-start;
  }

  .preview {
    display: grid;
    place-items: center;
    width: 12rem;
    aspect-ratio: 4 / 3;
    overflow: hidden;
    border-radius: var(--radius);
    background: var(--accent-soft);
    color: var(--accent);
  }

  .preview img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .controls {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .button {
    display: inline-block;
    padding: 0.45rem 0.9rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--accent);
    cursor: pointer;
  }

  .button:hover {
    background: var(--surface-hover);
  }

  .button.secondary {
    border-color: var(--border);
    color: var(--text);
  }

  /* Dateifeld unsichtbar, aber per Tastatur erreichbar */
  .button input {
    position: absolute;
    width: 1px;
    height: 1px;
    opacity: 0;
  }

  .button:focus-within {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }

  .hint,
  .error {
    margin: 0;
    font-size: 0.85rem;
  }

  .hint {
    color: var(--text-muted);
  }

  .error {
    color: var(--danger);
  }
</style>
