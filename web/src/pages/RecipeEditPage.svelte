<script>
  import { tick } from 'svelte';
  import { api, ApiError } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { toForm, toPayload } from '../lib/recipeForm.js';
  import TagFilter from '../components/TagFilter.svelte';
  import IngredientEditor from '../components/IngredientEditor.svelte';
  import StepEditor from '../components/StepEditor.svelte';
  import ImagePicker from '../components/ImagePicker.svelte';

  // id fehlt bei #/recipes/new, sonst kommt es aus #/recipes/:id/edit.
  let { id = null } = $props();
  // Die Seite wird bei jeder neuen Adresse neu aufgebaut ({#key} in
  // App.svelte), id ändert sich also nie; nur der Anfangswert zählt.
  // svelte-ignore state_referenced_locally
  const recipeId = id;
  const isNew = recipeId === null;

  let form = $state(null);
  let tagGroups = $state([]);
  let loadError = $state('');
  let errors = $state({});
  let saveError = $state('');
  let saving = $state(false);

  // Bild: gespeicherter Dateiname, neu gewählte Datei, Entfernen-Wunsch.
  // Hochgeladen wird erst beim Speichern (ein neues Rezept hat vorher
  // noch keine ID).
  let currentImage = $state('');
  let imageFile = $state(null);
  let removeImage = $state(false);

  // Stand beim Laden, um ungespeicherte Änderungen zu erkennen.
  let initial = '';

  api.listTagGroups().then((g) => (tagGroups = g)).catch(() => {});

  if (isNew) {
    start(toForm(null));
  } else {
    api
      .getRecipe(recipeId)
      .then((recipe) => {
        currentImage = recipe.image;
        start(toForm(recipe));
      })
      .catch((e) => (loadError = e.status === 404 ? 'Dieses Rezept gibt es nicht (mehr).' : e.message));
  }

  function start(f) {
    form = f;
    initial = JSON.stringify(f);
  }

  function toggleTag(tagId) {
    const i = form.tagIds.indexOf(tagId);
    if (i === -1) form.tagIds.push(tagId);
    else form.tagIds.splice(i, 1);
  }

  async function save(event) {
    event.preventDefault(); // Formular nicht vom Browser abschicken lassen
    saveError = '';
    const result = toPayload(form);
    errors = result.errors ?? {};
    if (result.errors) {
      saveError = 'Bitte die markierten Felder prüfen.';
      // tick wartet, bis Svelte die Markierungen angezeigt hat; dann zum
      // ersten markierten Feld scrollen.
      await tick();
      document.querySelector('form .invalid')?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      return;
    }

    saving = true;
    try {
      const saved = isNew ? await api.createRecipe(result.payload) : await api.updateRecipe(recipeId, result.payload);
      initial = JSON.stringify(form);
      try {
        if (imageFile) await api.setImage(saved.id, imageFile);
        else if (removeImage) await api.deleteImage(saved.id);
      } catch (e) {
        // Das Rezept selbst ist gespeichert; nur das Bild hat nicht geklappt.
        alert(`Rezept gespeichert, aber das Bild konnte nicht übernommen werden: ${e.message}`);
      }
      navigate(`/recipes/${saved.id}`);
    } catch (e) {
      // Prüfregeln des Servers melden das betroffene Feld mit.
      saveError = e instanceof ApiError && e.field ? `${e.message} (Feld: ${e.field})` : `Speichern fehlgeschlagen: ${e.message}`;
    } finally {
      saving = false;
    }
  }

  function cancel() {
    const changed = JSON.stringify(form) !== initial || imageFile !== null || removeImage;
    if (changed && !confirm('Änderungen verwerfen?')) return;
    navigate(isNew ? '/recipes' : `/recipes/${recipeId}`);
  }
</script>

<p class="back"><button type="button" class="link" onclick={cancel}>← Zurück</button></p>
<h1>{isNew ? 'Neues Rezept' : 'Rezept bearbeiten'}</h1>

{#if loadError}
  <p class="error">{loadError}</p>
{:else if !form}
  <p class="muted">Lädt …</p>
{:else}
  <!-- novalidate: Die Prüfung macht toPayload, mit deutschen Meldungen. -->
  <form onsubmit={save} novalidate>
    <section>
      <h2>Allgemein</h2>
      <label class="field">
        <span>Titel</span>
        <input bind:value={form.title} class:invalid={errors.title} />
        {#if errors.title}<small class="error">{errors.title}</small>{/if}
      </label>

      <label class="field">
        <span>Beschreibung</span>
        <textarea rows="2" bind:value={form.description}></textarea>
      </label>

      <div class="row">
        <label class="field">
          <span>Portionen</span>
          <input type="number" min="1" step="1" bind:value={form.base_servings} class:invalid={errors.base_servings} />
          {#if errors.base_servings}<small class="error">{errors.base_servings}</small>{/if}
        </label>
        <label class="field">
          <span>Vorbereitung (Min.)</span>
          <input type="number" min="0" step="1" bind:value={form.prep_minutes} class:invalid={errors.prep_minutes} />
          {#if errors.prep_minutes}<small class="error">{errors.prep_minutes}</small>{/if}
        </label>
        <label class="field">
          <span>Gesamt (Min.)</span>
          <input type="number" min="0" step="1" bind:value={form.total_minutes} class:invalid={errors.total_minutes} />
          {#if errors.total_minutes}<small class="error">{errors.total_minutes}</small>{/if}
        </label>
      </div>

      <label class="field">
        <span>Quelle (URL)</span>
        <input type="url" placeholder="https://…" bind:value={form.source_url} />
      </label>
    </section>

    <section>
      <h2>Bild</h2>
      <ImagePicker current={currentImage} bind:file={imageFile} bind:remove={removeImage} />
    </section>

    <section>
      <h2>Kategorien</h2>
      <TagFilter groups={tagGroups} selected={form.tagIds} ontoggle={toggleTag} />
    </section>

    <section>
      <h2>Zutaten</h2>
      <p class="hint">Mengen als 1,5 oder 1/2. Leere Zeilen werden beim Speichern weggelassen.</p>
      <IngredientEditor bind:items={form.items} {errors} />
    </section>

    <section>
      <h2>Zubereitung</h2>
      <StepEditor bind:steps={form.steps} />
    </section>

    <section>
      <h2>Notizen</h2>
      <textarea rows="3" aria-label="Notizen" bind:value={form.notes}></textarea>
    </section>

    <div class="actions">
      {#if saveError}<p class="error">{saveError}</p>{/if}
      <button type="button" class="secondary" onclick={cancel}>Abbrechen</button>
      <button type="submit" class="primary" disabled={saving}>{saving ? 'Speichert …' : 'Speichern'}</button>
    </div>
  </form>
{/if}

<style>
  .back {
    margin: 0 0 0.75rem;
  }

  .muted {
    color: var(--text-muted);
  }

  .error {
    color: var(--danger);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  section {
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  h2 {
    margin: 0 0 0.75rem;
    font-size: 1.15rem;
  }

  .hint {
    margin: -0.4rem 0 0.75rem;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    margin-bottom: 0.75rem;
  }

  .field span {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .field small {
    font-size: 0.85rem;
  }

  .row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: 0 0.75rem;
  }

  input,
  textarea {
    width: 100%;
    min-width: 0;
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
  }

  textarea {
    resize: vertical;
  }

  input:focus,
  textarea:focus {
    outline: 2px solid var(--accent);
    outline-offset: -1px;
  }

  .invalid {
    border-color: var(--danger);
  }

  /* Speichern bleibt auf dem Handy über der Navigationsleiste sichtbar. */
  .actions {
    position: sticky;
    bottom: calc(var(--nav-height) + 0.5rem);
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: flex-end;
    gap: 0.5rem;
    padding: 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: 0 -2px 10px rgb(0 0 0 / 0.06);
  }

  @media (min-width: 50rem) {
    .actions {
      bottom: 1rem;
    }
  }

  .actions .error {
    flex: 1 1 100%;
    margin: 0;
    text-align: right;
  }

  .primary,
  .secondary {
    padding: 0.55rem 1.2rem;
    border-radius: var(--radius);
    cursor: pointer;
  }

  .primary {
    border: 1px solid var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  .primary:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .secondary {
    border: 1px solid var(--border);
    background: var(--surface);
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    cursor: pointer;
  }
</style>
