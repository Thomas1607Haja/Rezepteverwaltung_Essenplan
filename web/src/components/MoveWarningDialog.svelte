<script>
  import { onMount } from 'svelte';
  import { noteText, t } from '../lib/i18n.svelte.js';

  // warnings: Hinweise des Servers zum Verschieben. onchoice wird mit
  // 'move' (trotzdem verschieben), 'have_ingredients' (verschieben und
  // Zutaten als vorhanden markieren), 'send' (verschieben und die Zutaten
  // gleich an Bring senden) oder 'cancel' aufgerufen.
  let { warnings, onchoice } = $props();

  // „Zutaten sind vorhanden“ passt nur, wenn ein geplantes Gericht in den
  // Einkaufszeitraum rutscht.
  const canMarkAvailable = $derived(warnings.some((w) => w.type === 'planned_into_period'));

  let dialog;
  onMount(() => dialog.showModal());
</script>

<dialog bind:this={dialog} oncancel={() => onchoice('cancel')} aria-labelledby="move-title">
  <div class="content">
    <h2 id="move-title">{t('plan.move.title')}</h2>
    {#each warnings as w (w.entry_id + w.type)}
      <p>{noteText({ code: w.type, params: w.params, message: w.message })}</p>
    {/each}
    <footer>
      <button type="button" class="secondary" onclick={() => onchoice('cancel')}>{t('common.cancel')}</button>
      {#if canMarkAvailable}
        <button type="button" class="secondary" onclick={() => onchoice('have_ingredients')}>{t('plan.move.have_ingredients')}</button>
        <button type="button" class="secondary" onclick={() => onchoice('send')}>{t('plan.move.send_to_bring')}</button>
      {/if}
      <button type="button" class="primary" onclick={() => onchoice('move')}>{t('plan.move.move_anyway')}</button>
    </footer>
  </div>
</dialog>

<style>
  dialog {
    width: min(28rem, calc(100vw - 2rem));
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
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  p {
    margin: 0;
  }

  footer {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 0.5rem;
    margin-top: 0.25rem;
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
</style>
