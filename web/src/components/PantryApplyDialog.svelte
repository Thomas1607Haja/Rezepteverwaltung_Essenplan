<script>
  import { onMount } from 'svelte';

  // changes: Vorschau des Servers (siehe api.applyPantry). onchoice wird mit
  // 'apply' oder 'cancel' aufgerufen.
  let { changes, recipes, busy = false, onchoice } = $props();

  // Änderungen nach Rezept zusammenfassen; der Server liefert sie schon
  // nach Rezepttitel sortiert.
  const groups = $derived.by(() => {
    const result = [];
    for (const c of changes) {
      const last = result.at(-1);
      if (last?.id === c.recipe_id) last.items.push(c);
      else result.push({ id: c.recipe_id, title: c.recipe_title, items: [c] });
    }
    return result;
  });

  let dialog;
  onMount(() => dialog.showModal());
</script>

<dialog bind:this={dialog} oncancel={() => onchoice('cancel')} aria-labelledby="apply-title">
  <div class="content">
    <h2 id="apply-title">Vorratsliste auf alle Rezepte übertragen?</h2>
    <p>
      {changes.length === 1 ? 'Eine Zutat' : `${changes.length} Zutaten`} in
      {recipes === 1 ? 'einem Rezept' : `${recipes} Rezepten`} ändern sich. Von Hand gesetzte Häkchen werden dabei überschrieben.
    </p>
    <ul class="recipes">
      {#each groups as g (g.id)}
        <li>
          <strong>{g.title}</strong>
          <ul>
            {#each g.items as c (c.ingredient_id)}
              <li>
                {c.ingredient_name}:
                <span class:in={!c.shopping_relevant} class:out={c.shopping_relevant}>
                  {c.shopping_relevant ? 'wird wieder eingekauft' : 'kommt in den Vorrat'}
                </span>
              </li>
            {/each}
          </ul>
        </li>
      {/each}
    </ul>
    <footer>
      <button type="button" class="secondary" onclick={() => onchoice('cancel')} disabled={busy}>Abbrechen</button>
      <button type="button" class="primary" onclick={() => onchoice('apply')} disabled={busy}>
        {busy ? 'Überträgt …' : 'Übertragen'}
      </button>
    </footer>
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
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  p {
    margin: 0;
  }

  /* Lange Listen scrollen, die Knöpfe bleiben sichtbar. */
  .recipes {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .recipes > li + li {
    margin-top: 0.5rem;
  }

  .recipes ul {
    margin: 0.15rem 0 0;
    padding-left: 1.25rem;
    font-size: 0.95rem;
  }

  .in {
    color: var(--text-muted);
  }

  .out {
    color: var(--accent);
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
