<script>
  import { newStep } from '../lib/recipeForm.js';

  let { steps = $bindable() } = $props();
</script>

<ol class="list">
  {#each steps as step, i (step.key)}
    <li class="card">
      <span class="number">{i + 1}</span>
      <textarea rows="2" placeholder="Was ist zu tun?" aria-label="Schritt {i + 1}" bind:value={step.text}></textarea>
      <button type="button" class="remove" aria-label="Schritt {i + 1} entfernen" onclick={() => steps.splice(i, 1)}>×</button>
    </li>
  {/each}
</ol>

<div class="add">
  <button type="button" onclick={() => steps.push(newStep())}>+ Schritt</button>
</div>

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .card {
    display: grid;
    grid-template-columns: 1.75rem 1fr auto;
    align-items: start;
    gap: 0.4rem;
    padding: 0.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--bg);
  }

  .number {
    padding-top: 0.4rem;
    text-align: center;
    font-weight: 700;
    color: var(--accent);
  }

  textarea {
    width: 100%;
    min-width: 0;
    padding: 0.4rem 0.5rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 0.2rem);
    background: var(--surface);
    resize: vertical;
    /* Wächst mit dem Text, wo der Browser es kann */
    field-sizing: content;
    min-height: 3.5rem;
  }

  textarea:focus {
    outline: 2px solid var(--accent);
    outline-offset: -1px;
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
    cursor: pointer;
  }

  .remove:hover {
    background: var(--surface-hover);
    color: var(--danger);
  }

  .add {
    margin-top: 0.75rem;
  }

  .add button {
    padding: 0.4rem 0.8rem;
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    background: none;
    color: var(--accent);
    cursor: pointer;
  }

  .add button:hover {
    background: var(--surface-hover);
  }
</style>
