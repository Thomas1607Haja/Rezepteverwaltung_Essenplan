<script>
  import { tick } from 'svelte';
  import { api, ApiError } from '../lib/api.js';
  import { navigate } from '../lib/router.svelte.js';
  import { takeFreshVariant } from '../lib/freshVariant.js';
  import { toForm, toPayload, validationText } from '../lib/recipeForm.js';
  import { t, errorText } from '../lib/i18n.svelte.js';
  import { yieldUnits } from '../lib/format.js';
  import TagFilter from '../components/TagFilter.svelte';
  import IngredientEditor from '../components/IngredientEditor.svelte';
  import StepEditor from '../components/StepEditor.svelte';
  import ImagePicker from '../components/ImagePicker.svelte';
  import QrScanDialog, { qrScanSupported } from '../components/QrScanDialog.svelte';

  // id fehlt bei #/recipes/new, sonst kommt es aus #/recipes/:id/edit.
  let { id = null } = $props();
  // Die Seite wird bei jeder neuen Adresse neu aufgebaut ({#key} in
  // App.svelte), id ändert sich also nie; nur der Anfangswert zählt.
  // svelte-ignore state_referenced_locally
  const recipeId = id;
  const isNew = recipeId === null;
  // Gerade per „Variante speichern“ angelegt? Dann die ID des
  // Ausgangsrezepts, sonst null.
  const variantSource = isNew ? null : takeFreshVariant(Number(recipeId));
  let titleInput = $state(null);

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
  // Bildadresse der Quellseite nach einem URL-Import (leer = keine).
  let remoteImage = $state('');

  // URL-Import (nur bei neuen Rezepten)
  let importAddress = $state('');
  // QR-Scanner (nur wenn der Browser ihn kann, siehe QrScanDialog)
  let canScan = $state(false);
  let scanning = $state(false);
  qrScanSupported().then((ok) => (canScan = ok));

  /** Inhalt des QR-Codes: nur eine http(s)-Adresse wird übernommen und gleich eingelesen. */
  function onScanned(text) {
    scanning = false;
    const match = text.trim().match(/https?:\/\/\S+/i);
    if (!match) {
      importError = t('editor.url_import.no_url_in_qr', { text: text.slice(0, 80) });
      return;
    }
    importAddress = match[0];
    importRecipe(new Event('submit'));
  }
  let importing = $state(false);
  let importError = $state('');
  let duplicates = $state([]);

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
        // Neue Variante: Titel markieren, damit man ihn gleich überschreibt.
        if (variantSource !== null) tick().then(() => titleInput?.select());
      })
      .catch((e) => (loadError = e.status === 404 ? t('editor.not_found') : errorText(e)));
  }

  function start(f) {
    form = f;
    initial = JSON.stringify(f);
  }

  /** Lädt das Rezept von importAddress und füllt damit das Formular. */
  async function importRecipe(event) {
    event.preventDefault();
    importError = '';
    if (importAddress.trim() === '') {
      importError = t('editor.url_import.address_missing');
      return;
    }
    if (JSON.stringify(form) !== initial && !confirm(t('editor.url_import.replace_confirm'))) return;

    importing = true;
    try {
      const draft = await api.importUrl(importAddress.trim());
      // initial bleibt beim leeren Formular: Der Entwurf ist noch nicht
      // gespeichert, Abbrechen fragt also nach.
      form = toForm(draft.recipe);
      errors = {};
      remoteImage = draft.image_url;
      imageFile = null;
      duplicates = draft.duplicates ?? [];
      importAddress = '';
    } catch (e) {
      importError = importMessage(e);
    } finally {
      importing = false;
    }
  }

  /** Verständliche Meldung zu einem fehlgeschlagenen Import. */
  function importMessage(e) {
    switch (e.status) {
      case 0:
        return errorText(e);
      case 422:
        return t('editor.url_import.no_recipe');
      case 502:
        // Mit Code ist der Text schon vollständig (z. B. „Die Seite ließ sich nicht laden“).
        return e.code ? errorText(e) : t('editor.url_import.page_failed', { message: errorText(e) });
      default:
        return t('editor.url_import.failed', { message: errorText(e) });
    }
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
      saveError = t('editor.check_fields');
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
        else if (remoteImage) await api.fetchImage(saved.id, remoteImage);
        else if (removeImage) await api.deleteImage(saved.id);
      } catch (e) {
        // Das Rezept selbst ist gespeichert; nur das Bild hat nicht geklappt.
        alert(t('editor.image_failed', { message: errorText(e) }));
      }
      navigate(`/recipes/${saved.id}`);
    } catch (e) {
      // Prüfregeln des Servers melden das betroffene Feld mit.
      saveError =
        e instanceof ApiError && e.field
          ? t('editor.save_error_field', { message: errorText(e), field: e.field })
          : t('editor.save_failed', { message: errorText(e) });
    } finally {
      saving = false;
    }
  }

  async function cancel() {
    // Gerade angelegte Variante: Abbrechen heißt, sie wieder zu verwerfen.
    if (variantSource !== null && confirm(t('editor.discard_variant'))) {
      try {
        await api.deleteRecipe(recipeId);
        navigate(`/recipes/${variantSource}`);
      } catch (e) {
        alert(t('editor.delete_failed', { message: errorText(e) }));
      }
      return;
    }
    const changed = JSON.stringify(form) !== initial || imageFile !== null || removeImage || remoteImage !== '';
    if (changed && !confirm(t('editor.discard_changes'))) return;
    navigate(isNew ? '/recipes' : `/recipes/${recipeId}`);
  }
</script>

<p class="back"><button type="button" class="link" onclick={cancel}>← {t('common.back')}</button></p>
<h1>{isNew ? t('editor.title_new') : variantSource !== null ? t('editor.title_variant') : t('editor.title_edit')}</h1>

{#if loadError}
  <p class="error">{loadError}</p>
{:else if !form}
  <p class="muted">{t('common.loading')}</p>
{:else}
  <!-- novalidate: Die Prüfung macht toPayload, mit übersetzten Meldungen. -->
  {#if isNew}
    <form class="import" onsubmit={importRecipe} novalidate>
      <h2>{t('editor.url_import.heading')}</h2>
      <p class="hint">{t('editor.url_import.hint')}</p>
      <div class="import-row">
        <input
          type="url"
          placeholder="https://…"
          aria-label={t('editor.url_import.address_label')}
          bind:value={importAddress}
          class:invalid={importError}
          disabled={importing}
        />
        {#if canScan}
          <button type="button" class="secondary" disabled={importing} onclick={() => (scanning = true)} title={t('editor.url_import.scan_title')}>
            {t('editor.url_import.scan')}
          </button>
        {/if}
        <button type="submit" class="primary" disabled={importing}>{importing ? t('common.loading') : t('common.apply')}</button>
      </div>
      {#if importError}<p class="error">{importError}</p>{/if}
    </form>
  {/if}

  {#if scanning}
    <QrScanDialog onresult={onScanned} onclose={() => (scanning = false)} />
  {/if}

  {#if duplicates.length > 0}
    <div class="duplicates" role="status">
      <strong>{t('editor.duplicates.title')}</strong>
      <ul>
        {#each duplicates as d (d.id)}
          <li><a href="#/recipes/{d.id}">{d.title}</a></li>
        {/each}
      </ul>
      <span>{t('editor.duplicates.hint')}</span>
    </div>
  {/if}

  <form onsubmit={save} novalidate>
    <section>
      <h2>{t('editor.sections.general')}</h2>
      <label class="field">
        <span>{t('editor.fields.title')}</span>
        <input bind:value={form.title} bind:this={titleInput} class:invalid={errors.title} />
        {#if errors.title}<small class="error">{validationText(errors.title)}</small>{/if}
      </label>

      <label class="field">
        <span>{t('editor.fields.description')}</span>
        <textarea rows="2" bind:value={form.description}></textarea>
      </label>

      <div class="row">
        <label class="field">
          <span>{form.yield_unit.trim() ? t('editor.fields.amount') : t('editor.fields.servings')}</span>
          <input type="number" min="1" step="1" bind:value={form.base_servings} class:invalid={errors.base_servings} />
          {#if errors.base_servings}<small class="error">{validationText(errors.base_servings)}</small>{/if}
        </label>
        <label class="field">
          <span>{t('editor.fields.unit')}</span>
          <!-- Leer = Portionen; für Kuchen & Co. z. B. „Springform 26 cm“ -->
          <input list="yield-units" maxlength="40" placeholder={t('editor.fields.servings')} bind:value={form.yield_unit} />
          <datalist id="yield-units">
            {#each yieldUnits() as u (u)}<option value={u}></option>{/each}
          </datalist>
        </label>
        <label class="field">
          <span>{t('editor.fields.prep')}</span>
          <input type="number" min="0" step="1" bind:value={form.prep_minutes} class:invalid={errors.prep_minutes} />
          {#if errors.prep_minutes}<small class="error">{validationText(errors.prep_minutes)}</small>{/if}
        </label>
        <label class="field">
          <span>{t('editor.fields.total')}</span>
          <input type="number" min="0" step="1" bind:value={form.total_minutes} class:invalid={errors.total_minutes} />
          {#if errors.total_minutes}<small class="error">{validationText(errors.total_minutes)}</small>{/if}
        </label>
      </div>

      <label class="field">
        <span>{t('editor.fields.source_url')}</span>
        <input type="url" placeholder="https://…" bind:value={form.source_url} />
      </label>
      <label class="field">
        <span>{t('editor.fields.source_name')}</span>
        <input type="text" maxlength="200" placeholder={t('editor.fields.source_name_placeholder')} bind:value={form.source_name} />
      </label>
    </section>

    <section>
      <h2>{t('editor.sections.image')}</h2>
      <ImagePicker current={currentImage} bind:file={imageFile} bind:remove={removeImage} bind:remoteUrl={remoteImage} />
    </section>

    <section>
      <h2>{t('editor.sections.categories')}</h2>
      <TagFilter groups={tagGroups} selected={form.tagIds} ontoggle={toggleTag} />
    </section>

    <section>
      <h2>{t('editor.sections.ingredients')}</h2>
      <p class="hint">{t('editor.ingredients_hint')}</p>
      <IngredientEditor bind:items={form.items} {errors} />
    </section>

    <section>
      <h2>{t('editor.sections.steps')}</h2>
      <StepEditor bind:steps={form.steps} />
    </section>

    <section>
      <h2>{t('editor.sections.notes')}</h2>
      <textarea rows="3" aria-label={t('editor.sections.notes')} bind:value={form.notes}></textarea>
    </section>

    <div class="actions">
      {#if saveError}<p class="error">{saveError}</p>{/if}
      <button type="button" class="secondary" onclick={cancel}>{t('common.cancel')}</button>
      <button type="submit" class="primary" disabled={saving}>{saving ? t('common.saving') : t('common.save')}</button>
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

  section,
  .import {
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .import {
    gap: 0;
    margin-bottom: 1rem;
  }

  .import .error {
    margin: 0.5rem 0 0;
  }

  .import-row {
    display: flex;
    gap: 0.5rem;
  }

  .duplicates {
    margin-bottom: 1rem;
    padding: 0.75rem 1rem;
    border: 1px solid var(--danger);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .duplicates ul {
    margin: 0.25rem 0;
    padding-left: 1.25rem;
  }

  .duplicates span {
    font-size: 0.85rem;
    color: var(--text-muted);
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
