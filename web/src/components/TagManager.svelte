<script>
  import { flip } from 'svelte/animate';
  import { dragHandleZone } from 'svelte-dnd-action';
  import { api } from '../lib/api.js';
  import { flipDurationMs, dropTargetStyle, isShadow } from '../lib/dnd.js';
  import DragHandle from './DragHandle.svelte';

  // Kategorien und Gruppen verwalten (Schritt 8.C). Gruppen und Kategorien
  // lassen sich ziehen – Kategorien auch in eine andere Gruppe. Jede
  // Änderung geht sofort an den Server; danach wird neu geladen, damit die
  // Anzeige genau dem Stand auf dem Server entspricht.

  let groups = $state(null); // null = lädt noch
  let loadError = $state('');
  let message = $state('');
  let messageIsError = $state(false);
  let busy = $state(false);

  // Geöffnetes Bearbeiten: { kind: 'group' | 'tag', id } bzw. null
  let editing = $state(null);
  let draft = $state({ name: '', groupId: 0, mergeInto: 0 });
  let newGroup = $state('');
  // Eingabe „Neue Kategorie“ je Gruppe
  let newTag = $state({});
  // Gruppe, die gelöscht werden soll, solange der Dialog offen ist
  let deleting = $state(null);
  let deleteMode = $state('move');
  let deleteTarget = $state(0);
  // Text für die Claude-Skill, falls Kopieren in die Zwischenablage nicht geht
  let copyText = $state('');

  // Stand des Servers: Gruppe → IDs ihrer Kategorien (zum Erkennen, welche
  // Liste nach dem Ziehen gespeichert werden muss).
  let saved = {};

  async function load() {
    try {
      const list = await api.adminTagGroups();
      saved = Object.fromEntries(list.map((g) => [g.id, g.tags.map((t) => t.id)]));
      groups = list;
      loadError = '';
    } catch (e) {
      loadError = `Kategorien nicht geladen: ${e.message}`;
    }
  }
  load();

  function showMessage(text, isError = false) {
    message = text;
    messageIsError = isError;
  }

  /** Führt eine Änderung aus, meldet Fehler und lädt danach neu. */
  async function run(action, success = '') {
    busy = true;
    message = '';
    try {
      await action();
      if (success) showMessage(success);
      editing = null;
    } catch (e) {
      showMessage(e.message, true);
    } finally {
      await load();
      busy = false;
    }
  }

  const recipes = (n) => (n === 1 ? '1 Rezept' : `${n} Rezepte`);
  const allTags = $derived(groups?.flatMap((g) => g.tags.map((t) => ({ ...t, group: g.name }))) ?? []);

  // --- Ziehen ------------------------------------------------------------

  function considerGroups(event) {
    groups = event.detail.items;
  }

  function finalizeGroups(event) {
    groups = event.detail.items;
    run(() => api.orderTagGroups(groups.map((g) => g.id)));
  }

  function considerTags(group, event) {
    group.tags = event.detail.items;
  }

  /**
   * Ende des Ziehens. Die Bibliothek meldet es für beide Listen, die alte
   * und die neue Gruppe. Gespeichert wird nur eine Liste, die etwas Neues
   * enthält oder umsortiert wurde – die alte Gruppe hat nur eine Kategorie
   * verloren, das erledigt das Speichern der neuen Gruppe mit.
   */
  function finalizeTags(group, event) {
    group.tags = event.detail.items;
    const ids = group.tags.map((t) => t.id);
    const before = saved[group.id] ?? [];
    const onlyLost = ids.every((id) => before.includes(id)) && ids.join() === before.filter((id) => ids.includes(id)).join();
    if (onlyLost) return;
    run(() => api.orderTags(group.id, ids));
  }

  // --- Gruppen -----------------------------------------------------------

  function addGroup(event) {
    event.preventDefault();
    const name = newGroup.trim();
    if (name === '') return;
    run(async () => {
      await api.createTagGroup(name);
      newGroup = '';
    }, `Gruppe „${name}“ angelegt.`);
  }

  function toggleCourse(group) {
    const on = group.role !== 'course';
    run(() => api.setCourseGroup(group.id, on), on ? `„${group.name}“ ist jetzt die Gang-Gruppe.` : '');
  }

  function startEditGroup(group) {
    editing = { kind: 'group', id: group.id };
    draft = { name: group.name, groupId: 0, mergeInto: 0 };
    message = '';
  }

  function renameGroup(event, group) {
    event.preventDefault();
    run(() => api.renameTagGroup(group.id, draft.name));
  }

  function askDeleteGroup(group) {
    if (group.tags.length === 0) {
      if (confirm(`Gruppe „${group.name}“ löschen?`)) run(() => api.deleteTagGroup(group.id), `Gruppe „${group.name}“ gelöscht.`);
      return;
    }
    deleting = group;
    deleteMode = 'move';
    deleteTarget = groups.find((g) => g.id !== group.id)?.id ?? 0;
    if (deleteTarget === 0) deleteMode = 'delete';
  }

  function confirmDeleteGroup() {
    const group = deleting;
    deleting = null;
    const options = deleteMode === 'move' ? { moveTo: deleteTarget } : { withTags: true };
    run(() => api.deleteTagGroup(group.id, options), `Gruppe „${group.name}“ gelöscht.`);
  }

  // --- Kategorien --------------------------------------------------------

  function addTag(event, group) {
    event.preventDefault();
    const name = (newTag[group.id] ?? '').trim();
    if (name === '') return;
    run(async () => {
      await api.createTag(name, group.id);
      newTag[group.id] = '';
    }, `Kategorie „${name}“ angelegt.`);
  }

  function startEditTag(tag, group) {
    editing = { kind: 'tag', id: tag.id };
    draft = { name: tag.name, groupId: group.id, mergeInto: 0 };
    message = '';
  }

  function saveTag(event, tag) {
    event.preventDefault();
    run(() => api.updateTag(tag.id, draft.name, draft.groupId));
  }

  function mergeTag(tag) {
    const into = allTags.find((t) => t.id === draft.mergeInto);
    if (!into) return;
    const text =
      tag.recipe_count > 0
        ? `${tag.recipe_count === 1 ? 'Das Rezept' : `Alle ${tag.recipe_count} Rezepte`} mit „${tag.name}“ ${tag.recipe_count === 1 ? 'bekommt' : 'bekommen'} „${into.name}“. Danach wird „${tag.name}“ gelöscht.`
        : `„${tag.name}“ hat keine Rezepte und wird gelöscht.`;
    if (!confirm(`Zusammenführen?\n\n${text}`)) return;
    run(() => api.mergeTag(tag.id, into.id), `„${tag.name}“ in „${into.name}“ zusammengeführt.`);
  }

  function deleteTag(tag) {
    const text =
      tag.recipe_count > 0
        ? `„${tag.name}“ ist ${tag.recipe_count === 1 ? 'einem Rezept' : `${tag.recipe_count} Rezepten`} zugeordnet.\n\nKategorie löschen? Die Rezepte bleiben unverändert, nur die Zuordnung fällt weg.`
        : `Kategorie „${tag.name}“ löschen? Sie ist keinem Rezept zugeordnet.`;
    if (!confirm(text)) return;
    run(() => api.deleteTag(tag.id), `„${tag.name}“ gelöscht.`);
  }

  // --- Liste für die Claude-Skill -----------------------------------------

  /** Aufbau wie skills/rezeptplaner-import/categories.md (## Gruppe, - Name). */
  function skillList() {
    const today = new Date().toLocaleDateString('de-DE');
    const lines = [
      '# Kategorien des Rezeptplaners',
      '',
      `Stand ${today}. Verwende für „categories“ ausschließlich diese Namen, genau so geschrieben – sie ersetzen categories.md der Skill (als Datei speichern und beim Bauen mit --categories angeben).`,
    ];
    for (const g of groups) {
      if (g.tags.length === 0) continue;
      lines.push('', `## ${g.name}`, ...g.tags.map((t) => `- ${t.name}`));
    }
    return lines.join('\n') + '\n';
  }

  async function copyForClaude() {
    const text = skillList();
    try {
      // Die Zwischenablage gibt es nur über https oder auf dem eigenen Rechner.
      await navigator.clipboard.writeText(text);
      copyText = '';
      showMessage('Kategorieliste kopiert – im Chat mit Claude einfügen.');
    } catch {
      copyText = text;
      showMessage('Kopieren hier nicht möglich – bitte den Text unten markieren und kopieren.', true);
    }
  }
</script>

<section class="card">
  <h2>Kategorien</h2>
  <p class="hint">
    Gruppen und Kategorien zum Filtern der Rezepte. Am Griff ziehen zum Sortieren – Kategorien auch in eine andere Gruppe.
    Namen gibt es nur einmal (Groß-/Kleinschreibung und Umlaute zählen nicht).
  </p>

  {#if loadError}
    <p class="error">{loadError}</p>
  {:else if groups === null}
    <p>Lädt …</p>
  {:else}
    <div
      class="groups"
      aria-label="Gruppen"
      use:dragHandleZone={{ items: groups, flipDurationMs, dropTargetStyle, type: 'tag-groups', dragDisabled: busy }}
      onconsider={considerGroups}
      onfinalize={finalizeGroups}
    >
      {#each groups as group (group.id)}
        <div class="group" class:shadow={isShadow(group)} aria-label="Gruppe {group.name}" animate:flip={{ duration: flipDurationMs }}>
          <div class="group-head">
            <DragHandle label="Gruppe {group.name} verschieben" />
            {#if editing?.kind === 'group' && editing.id === group.id}
              <form class="inline" onsubmit={(e) => renameGroup(e, group)}>
                <input aria-label="Name der Gruppe" bind:value={draft.name} maxlength="50" required />
                <button type="button" onclick={() => (editing = null)} disabled={busy}>Abbrechen</button>
                <button type="submit" class="primary" disabled={busy}>Speichern</button>
              </form>
            {:else}
              <strong class="group-name">
                {group.name}
                {#if group.role === 'course'}<span class="badge" title="Schnellfilter-Reiter und Gänge im Menü">Gang</span>{/if}
              </strong>
              <button type="button" onclick={() => toggleCourse(group)} disabled={busy}>
                {group.role === 'course' ? 'Kein Gang' : 'Als Gang'}
              </button>
              <button type="button" onclick={() => startEditGroup(group)} disabled={busy}>Umbenennen</button>
              <button type="button" class="remove" onclick={() => askDeleteGroup(group)} disabled={busy} aria-label="Gruppe {group.name} löschen">×</button>
            {/if}
          </div>

          <ul
            class="tags"
            aria-label="Kategorien der Gruppe {group.name}"
            use:dragHandleZone={{ items: group.tags, flipDurationMs, dropTargetStyle, type: 'tags', dragDisabled: busy }}
            onconsider={(e) => considerTags(group, e)}
            onfinalize={(e) => finalizeTags(group, e)}
          >
            {#each group.tags as tag (tag.id)}
              <li class:shadow={isShadow(tag)} aria-label="Kategorie {tag.name}" animate:flip={{ duration: flipDurationMs }}>
                {#if editing?.kind === 'tag' && editing.id === tag.id}
                  <div class="tag-edit">
                    <form class="inline" onsubmit={(e) => saveTag(e, tag)}>
                      <input aria-label="Name der Kategorie" bind:value={draft.name} maxlength="50" required />
                      <select aria-label="Gruppe" bind:value={draft.groupId}>
                        {#each groups as g (g.id)}<option value={g.id}>{g.name}</option>{/each}
                      </select>
                      <button type="submit" class="primary" disabled={busy}>Speichern</button>
                    </form>
                    <div class="inline">
                      <select aria-label="Zusammenführen mit" bind:value={draft.mergeInto}>
                        <option value={0}>Zusammenführen mit …</option>
                        {#each groups as g (g.id)}
                          {#if g.tags.some((t) => t.id !== tag.id)}
                            <optgroup label={g.name}>
                              {#each g.tags.filter((t) => t.id !== tag.id) as t (t.id)}<option value={t.id}>{t.name}</option>{/each}
                            </optgroup>
                          {/if}
                        {/each}
                      </select>
                      <button type="button" onclick={() => mergeTag(tag)} disabled={busy || !draft.mergeInto}>Zusammenführen</button>
                    </div>
                    <div class="inline end">
                      <button type="button" class="danger" onclick={() => deleteTag(tag)} disabled={busy}>Löschen</button>
                      <button type="button" onclick={() => (editing = null)} disabled={busy}>Schließen</button>
                    </div>
                  </div>
                {:else}
                  <div class="row">
                    <DragHandle label="Kategorie {tag.name} verschieben" />
                    <span class="text">{tag.name} <small>{recipes(tag.recipe_count)}</small></span>
                    <button type="button" onclick={() => startEditTag(tag, group)} disabled={busy} aria-label="{tag.name} bearbeiten">Bearbeiten</button>
                  </div>
                {/if}
              </li>
            {/each}
          </ul>

          <form class="inline add" onsubmit={(e) => addTag(e, group)}>
            <input aria-label="Neue Kategorie in {group.name}" placeholder="Neue Kategorie" bind:value={newTag[group.id]} maxlength="50" />
            <button type="submit" disabled={busy || !(newTag[group.id] ?? '').trim()}>Hinzufügen</button>
          </form>
        </div>
      {/each}
    </div>

    <form class="inline add-group" onsubmit={addGroup}>
      <input aria-label="Neue Gruppe" placeholder="Neue Gruppe, z. B. Anlass" bind:value={newGroup} maxlength="50" />
      <button type="submit" class="primary" disabled={busy || newGroup.trim() === ''}>Gruppe anlegen</button>
    </form>

    <p class="hint">
      Die Gang-Gruppe (Vorspeise, Hauptgericht, Kuchen …) steuert die Reiter in der Rezeptübersicht.
      <a href="#/settings/courses">Gänge zuordnen</a> – Vorschläge für Rezepte ohne Gang.
    </p>

    <div class="claude">
      <button type="button" onclick={copyForClaude} disabled={busy}>Kategorieliste für Claude kopieren</button>
      {#if message}<span class:error={messageIsError} class:ok={!messageIsError}>{message}</span>{/if}
    </div>
    <p class="hint">
      Die Claude-Skill kennt nur die Startkategorien. Nach Änderungen hier die Liste kopieren und im Chat mit Claude
      einfügen; beim Import lassen sich unbekannte Kategorien außerdem zuordnen oder neu anlegen.
    </p>
    {#if copyText}
      <textarea readonly rows="8" aria-label="Kategorieliste für Claude" onfocus={(e) => e.currentTarget.select()}>{copyText}</textarea>
    {/if}
  {/if}
</section>

{#if deleting}
  <div class="backdrop" role="presentation">
    <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="delete-group-title">
      <h3 id="delete-group-title">Gruppe „{deleting.name}“ löschen</h3>
      <p>
        Sie enthält {deleting.tags.length === 1 ? 'eine Kategorie' : `${deleting.tags.length} Kategorien`}:
        {deleting.tags.map((t) => t.name).join(', ')}.
      </p>
      {#if groups.length > 1}
        <label class="choice">
          <input type="radio" bind:group={deleteMode} value="move" />
          In Gruppe verschieben:
          <select bind:value={deleteTarget} disabled={deleteMode !== 'move'}>
            {#each groups.filter((g) => g.id !== deleting.id) as g (g.id)}<option value={g.id}>{g.name}</option>{/each}
          </select>
        </label>
      {/if}
      <label class="choice">
        <input type="radio" bind:group={deleteMode} value="delete" />
        Kategorien mitlöschen ({recipes(deleting.tags.reduce((n, t) => n + t.recipe_count, 0))} verlieren die Zuordnung, die Rezepte bleiben)
      </label>
      <div class="inline end">
        <button type="button" onclick={() => (deleting = null)}>Abbrechen</button>
        <button type="button" class="danger" onclick={confirmDeleteGroup}>Gruppe löschen</button>
      </div>
    </div>
  </div>
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

  h3 {
    margin: 0;
    font-size: 1.05rem;
  }

  p {
    margin: 0;
  }

  .hint {
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .groups {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin: 0.75rem 0;
  }

  .group {
    padding: 0.5rem 0.75rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg, var(--surface));
  }

  .group-head {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .group-name {
    flex: 1;
  }

  .badge {
    margin-left: 0.4rem;
    padding: 0.05rem 0.45rem;
    border-radius: 999px;
    background: var(--accent-soft);
    color: var(--accent);
    font-size: 0.75rem;
    font-weight: 600;
  }

  .tags {
    min-height: 2.2rem; /* leere Gruppe bleibt Ziel zum Hineinziehen */
    margin: 0.4rem 0;
    padding: 0;
    list-style: none;
    border-top: 1px solid var(--border);
  }

  .tags > li {
    padding: 0.25rem 0;
    border-bottom: 1px solid var(--border);
    background: var(--surface);
  }

  .shadow {
    opacity: 0.4;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .text {
    flex: 1;
    min-width: 0;
  }

  .text small {
    margin-left: 0.4rem;
    color: var(--text-muted);
  }

  .tag-edit {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.4rem 0;
  }

  .inline {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
    flex: 1;
  }

  .inline input,
  .inline select {
    flex: 1;
    min-width: 8rem;
  }

  .end {
    justify-content: flex-end;
  }

  .add-group {
    margin-bottom: 1rem;
  }

  input,
  select,
  textarea {
    padding: 0.45rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    font: inherit;
  }

  textarea {
    width: 100%;
    margin-top: 0.5rem;
    font-family: ui-monospace, monospace;
    font-size: 0.85rem;
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

  .danger {
    color: var(--danger);
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

  .claude {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.35rem;
  }

  .error {
    color: var(--danger);
  }

  .ok {
    color: var(--accent);
  }

  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 20;
    display: grid;
    place-items: center;
    padding: 1rem;
    background: rgb(0 0 0 / 0.4);
  }

  .dialog {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    width: min(30rem, 100%);
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .choice {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
  }
</style>
