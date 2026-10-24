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
  // Unbekannte Kategorien: Wahl je Name ('' = weglassen, 'tag:7' = zuordnen,
  // 'new:3' = in Gruppe 3 neu anlegen) und die Gruppen zur Auswahl.
  let categoryChoice = $state({});
  let tagGroups = $state([]);

  const importable = (r) => r.errors.length === 0;

  async function upload(files) {
    if (!files || files.length === 0) return;
    error = '';
    step = 'uploading';
    try {
      preview = await api.importFiles([...files]);
      categoryChoice = Object.fromEntries(preview.unknown_categories.map((name) => [name, '']));
      if (preview.unknown_categories.length > 0) tagGroups = await api.adminTagGroups();
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
      result = await api.confirmImport(preview.token, [...selected].sort((a, b) => a - b), categoryChoices());
      step = 'done';
    } catch (e) {
      error = e.message;
      // Abgelaufen o. Ä.: zurück zur Auswahl, sonst Vorschau behalten.
      step = e.status === 404 ? 'choose' : 'preview';
    }
  }

  /** Wahl für den Server: { Name: { tag_id } | { new_in_group } }; Weglassen fehlt. */
  function categoryChoices() {
    const out = {};
    for (const [name, value] of Object.entries(categoryChoice)) {
      const [kind, id] = value.split(':');
      if (kind === 'tag') out[name] = { tag_id: Number(id) };
      if (kind === 'new') out[name] = { new_in_group: Number(id) };
    }
    return out;
  }

  // Hinweise zu unbekannten Kategorien stehen gesammelt im eigenen Abschnitt.
  const isCategoryHint = (w) => w.startsWith('Kategorie „') && w.includes('gibt es nicht');

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
      {step === 'uploading' ? 'Wird geprüft (Fotos von Rezeptseiten laden kann bis zu 20 s dauern) …' : 'Datei auswählen'}
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
          <figure>
            <img src={api.importThumbnailUrl(preview.token, r.index)} alt="" />
            {#if r.web_image}<figcaption>Foto aus Link</figcaption>{/if}
          </figure>
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
          {#if r.web_image}
            <!-- Der Server meldet das auch als Hinweis; hier als Info mit Link statt als Warnung. -->
            <p class="from-web">
              Foto von <a href={r.recipe.source_url} target="_blank" rel="noopener noreferrer">{r.web_image}</a> wird übernommen
            </p>
          {/if}
          {#if r.unknown_categories.length > 0}
            <p class="warning">Unbekannte Kategorien: {r.unknown_categories.join(', ')} (siehe unten)</p>
          {/if}
          {#each r.warnings.filter((w) => !(r.web_image && w.startsWith('Foto von ')) && !isCategoryHint(w)) as msg, i (i)}<p class="warning">
              {msg}
            </p>{/each}
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

  {#if preview.unknown_categories.length > 0}
    <section class="unknown">
      <h2>Unbekannte Kategorien</h2>
      <p class="hint">Diese Kategorien gibt es im Rezeptplaner nicht. Einer vorhandenen zuordnen, neu anlegen oder weglassen.</p>
      {#each preview.unknown_categories as name (name)}
        <label class="mapping">
          <span>„{name}“</span>
          <select bind:value={categoryChoice[name]} disabled={step === 'importing'}>
            <option value="">Weglassen</option>
            {#each tagGroups as g (g.id)}
              {#if g.tags.length > 0}
                <optgroup label="Zuordnen – {g.name}">
                  {#each g.tags as t (t.id)}<option value="tag:{t.id}">{t.name}</option>{/each}
                </optgroup>
              {/if}
            {/each}
            <optgroup label="Neu anlegen in Gruppe">
              {#each tagGroups as g (g.id)}<option value="new:{g.id}">„{name}“ in {g.name}</option>{/each}
            </optgroup>
          </select>
        </label>
      {/each}
    </section>
  {/if}

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
    {#each result.warnings ?? [] as w, i (i)}<p class="warning">{w}</p>{/each}
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

  figure {
    position: relative;
    margin: 0;
  }

  figcaption {
    position: absolute;
    right: 0.2rem;
    bottom: 0.2rem;
    left: 0.2rem;
    padding: 0 0.2rem;
    border-radius: 0.3rem;
    background: rgb(0 0 0 / 0.6);
    color: #fff;
    font-size: 0.65rem;
    text-align: center;
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

  .from-web {
    color: var(--accent);
  }

  .unknown {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin: 1rem 0;
    padding: 0.75rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .unknown h2 {
    margin: 0;
    font-size: 1.05rem;
  }

  .unknown .hint {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  .mapping {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
  }

  .mapping span {
    min-width: 9rem;
    font-weight: 600;
  }

  .mapping select {
    flex: 1;
    min-width: 12rem;
    padding: 0.4rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    font: inherit;
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
  }
</style>
