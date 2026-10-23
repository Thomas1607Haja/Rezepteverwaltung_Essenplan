<script>
  import { api } from '../lib/api.js';
  import { formatMinutes } from '../lib/format.js';

  // Datei-Import (ZIP oder JSON, z. B. von der Claude-Skill). Ablauf:
  // Dateien wählen → Server prüft und liefert eine Vorschau (noch nichts
  // gespeichert) → Rezepte abwählen → Importieren.
  let step = $state('choose'); // choose | uploading | preview | importing | done
  let error = $state('');
  let preview = $state(null);
  let selected = $state(new Set());
  let result = $state(null);
  let dragging = $state(false);
  let input = $state();

  const importable = (r) => r.errors.length === 0;

  async function upload(files) {
    if (!files || files.length === 0) return;
    error = '';
    step = 'uploading';
    try {
      preview = await api.importFiles([...files]);
      // Vorauswahl: alles Importierbare außer Dubletten.
      selected = new Set(
        preview.recipes.filter((r) => importable(r) && r.duplicates.length === 0).map((r) => r.index),
      );
      step = 'preview';
    } catch (e) {
      error = e.message;
      step = 'choose';
    }
  }

  function toggle(index) {
    const next = new Set(selected);
    if (next.has(index)) next.delete(index);
    else next.add(index);
    selected = next;
  }

  async function confirmImport() {
    error = '';
    step = 'importing';
    try {
      result = await api.confirmImport(preview.token, [...selected].sort((a, b) => a - b));
      step = 'done';
    } catch (e) {
      error = e.message;
      // Abgelaufen o. Ä.: zurück zur Auswahl, sonst Vorschau behalten.
      step = e.status === 404 ? 'choose' : 'preview';
    }
  }

  function restart() {
    preview = null;
    result = null;
    error = '';
    step = 'choose';
  }

  function onDrop(event) {
    event.preventDefault();
    dragging = false;
    upload(event.dataTransfer?.files);
  }

  const count = (n, one, many) => `${n} ${n === 1 ? one : many}`;

  function amounts(r) {
    const parts = [
      count(r.recipe.base_servings, 'Portion', 'Portionen'),
      count(r.recipe.ingredients.length, 'Zutat', 'Zutaten'),
      count(r.recipe.steps.length, 'Schritt', 'Schritte'),
    ];
    if (r.recipe.total_minutes != null) parts.push(formatMinutes(r.recipe.total_minutes));
    return parts.join(' · ');
  }

  function quantity(ing) {
    if (ing.quantity == null) return ing.unit;
    return `${ing.quantity.toLocaleString('de-DE')} ${ing.unit}`.trim();
  }
</script>

<div class="page-head">
  <h1>Datei importieren</h1>
  <a href="#/recipes">Zurück zu den Rezepten</a>
</div>

{#if error}<p class="error" role="alert">{error}</p>{/if}

{#if step === 'choose' || step === 'uploading'}
  <section
    class="drop"
    class:dragging
    role="region"
    aria-label="Dateien hier ablegen"
    ondragover={(e) => {
      e.preventDefault();
      dragging = true;
    }}
    ondragleave={() => (dragging = false)}
    ondrop={onDrop}
  >
    <p>
      ZIP-Datei (z. B. von der Claude-Skill „rezeptplaner-import“) oder JSON-Datei hier ablegen oder auswählen. Es wird
      noch nichts gespeichert – zuerst kommt eine Vorschau.
    </p>
    <input
      bind:this={input}
      type="file"
      multiple
      accept=".zip,.json,application/zip,application/json"
      hidden
      onchange={(e) => upload(e.currentTarget.files)}
    />
    <button type="button" class="primary" disabled={step === 'uploading'} onclick={() => input.click()}>
      {step === 'uploading' ? 'Wird geprüft …' : 'Datei auswählen'}
    </button>
  </section>
{/if}

{#if (step === 'preview' || step === 'importing') && preview}
  {#each preview.file_errors as fe (fe.file + fe.error)}
    <p class="error">{fe.file}: {fe.error}</p>
  {/each}

  {#if preview.recipes.length === 0}
    <p>Keine Rezepte gefunden.</p>
  {/if}

  <ul class="recipes">
    {#each preview.recipes as r (r.index)}
      <li class:disabled={!importable(r)}>
        <label class="pick">
          <input
            type="checkbox"
            checked={selected.has(r.index)}
            disabled={!importable(r) || step === 'importing'}
            onchange={() => toggle(r.index)}
          />
          <span class="sr-only">Importieren</span>
        </label>
        {#if r.thumbnail}
          <img src={api.importThumbnailUrl(preview.token, r.index)} alt="" />
        {:else}
          <div class="noimg" aria-hidden="true"></div>
        {/if}
        <div class="info">
          <strong>{r.recipe.title || '(ohne Titel)'}</strong>
          <small>{amounts(r)}</small>
          {#if r.recipe.source_name || r.recipe.source_url}
            <small>Quelle: {r.recipe.source_name || r.recipe.source_url}</small>
          {/if}
          {#if r.recipe.tags.length > 0}
            <div class="tags">
              {#each r.recipe.tags as tag (tag.id)}<span>{tag.name}</span>{/each}
            </div>
          {/if}
          {#each r.errors as msg, i (i)}<p class="error">{msg}</p>{/each}
          {#if r.duplicates.length > 0}
            <p class="warning">
              Gibt es schon:
              {#each r.duplicates as d, i (d.id)}{i > 0 ? ', ' : ''}<a
                  href="#/recipes/{d.id}"
                  target="_blank"
                  rel="noopener">{d.title}</a
                >{/each}
            </p>
          {/if}
          {#each r.warnings as msg, i (i)}<p class="warning">{msg}</p>{/each}
          {#if importable(r)}
            <details>
              <summary>Zutaten und Schritte</summary>
              <ul class="ingredients">
                {#each r.recipe.ingredients as ing, i (i)}
                  <li class:pantry={!ing.shopping_relevant}>
                    {#if ing.group_name && (i === 0 || r.recipe.ingredients[i - 1].group_name !== ing.group_name)}
                      <em>{ing.group_name}:</em>
                    {/if}
                    {quantity(ing)}
                    {ing.name}{ing.note ? `, ${ing.note}` : ''}
                    {#if !ing.shopping_relevant}<small>(nicht einkaufen)</small>{/if}
                  </li>
                {/each}
              </ul>
              <ol>
                {#each r.recipe.steps as s, i (i)}<li>{s.text}</li>{/each}
              </ol>
              {#if r.recipe.notes}<p class="notes">Notizen: {r.recipe.notes}</p>{/if}
            </details>
          {/if}
        </div>
      </li>
    {/each}
  </ul>

  <div class="actions">
    <button type="button" class="primary" disabled={selected.size === 0 || step === 'importing'} onclick={confirmImport}>
      {step === 'importing'
        ? 'Importiert …'
        : `${selected.size} ${selected.size === 1 ? 'Rezept' : 'Rezepte'} importieren`}
    </button>
    <button type="button" disabled={step === 'importing'} onclick={restart}>Andere Datei</button>
  </div>
{/if}

{#if step === 'done' && result}
  <section class="result">
    {#if result.created.length > 0}
      <h2>{result.created.length} {result.created.length === 1 ? 'Rezept' : 'Rezepte'} importiert</h2>
      <ul>
        {#each result.created as c (c.id)}
          <li>
            <a href="#/recipes/{c.id}">{c.title}</a>
            {#each c.warnings as w, i (i)}<span class="warning"> – {w}</span>{/each}
          </li>
        {/each}
      </ul>
    {/if}
    {#if result.failed.length > 0}
      <h2>Nicht importiert</h2>
      <ul>
        {#each result.failed as f (f.index)}
          <li class="error">{f.title || `Rezept ${f.index + 1}`}: {f.error}</li>
        {/each}
      </ul>
    {/if}
    <div class="actions">
      <a class="button primary" href="#/recipes">Zu den Rezepten</a>
      <button type="button" onclick={restart}>Weitere Datei importieren</button>
    </div>
  </section>
{/if}

<style>
  .page-head {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.5rem 1rem;
    margin-bottom: 1rem;
  }

  .page-head h1 {
    margin: 0;
  }

  .drop {
    padding: 1.5rem 1rem;
    text-align: center;
    background: var(--surface);
    border: 2px dashed var(--border);
    border-radius: var(--radius);
  }

  .drop.dragging {
    border-color: var(--accent);
    background: var(--accent-soft);
  }

  .drop p {
    max-width: 36rem;
    margin: 0 auto 1rem;
    color: var(--text-muted);
  }

  .recipes {
    list-style: none;
    margin: 0 0 1rem;
    padding: 0;
  }

  .recipes > li {
    display: grid;
    grid-template-columns: auto 4.5rem 1fr;
    gap: 0.75rem;
    align-items: start;
    margin-bottom: 0.5rem;
    padding: 0.75rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .recipes > li.disabled {
    opacity: 0.8;
  }

  .pick {
    padding-top: 0.2rem;
  }

  .pick input {
    width: 1.2rem;
    height: 1.2rem;
  }

  img,
  .noimg {
    width: 4.5rem;
    height: 4.5rem;
    object-fit: cover;
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface-hover);
  }

  .info {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
  }

  .info small {
    color: var(--text-muted);
  }

  .info p {
    margin: 0;
    font-size: 0.9rem;
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    margin: 0.2rem 0;
  }

  .tags span {
    padding: 0 0.5rem;
    font-size: 0.8rem;
    background: var(--accent-soft);
    border-radius: 1rem;
  }

  details {
    margin-top: 0.3rem;
    font-size: 0.9rem;
  }

  summary {
    cursor: pointer;
    color: var(--accent);
  }

  .ingredients {
    margin: 0.4rem 0;
    padding-left: 1.2rem;
  }

  .ingredients .pantry {
    color: var(--text-muted);
  }

  ol {
    margin: 0.4rem 0;
    padding-left: 1.4rem;
  }

  .notes {
    white-space: pre-line;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  button,
  .button {
    padding: 0.5rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    text-decoration: none;
    cursor: pointer;
  }

  .primary {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }

  button:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .result {
    padding: 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .result h2 {
    margin: 0 0 0.5rem;
    font-size: 1.1rem;
  }

  .error {
    color: var(--danger);
  }

  .warning {
    color: var(--warning);
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
  }
</style>
