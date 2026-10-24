<script>
  // Schnellfilter-Reiter nach Gang (lib/courseTabs.js). value ist der key
  // des gewählten Reiters, onchange(key) meldet den Wechsel.
  let { tabs, value, onchange } = $props();
</script>

{#if tabs.length > 1}
  <div class="tabs" role="tablist" aria-label="Gang">
    {#each tabs as tab (tab.key)}
      <button type="button" role="tab" aria-selected={value === tab.key} onclick={() => onchange(tab.key)}>{tab.label}</button>
    {/each}
  </div>
{/if}

<style>
  .tabs {
    display: flex;
    gap: 0.4rem;
    margin: 0.5rem 0;
    overflow-x: auto;
    scrollbar-width: none;
  }

  button {
    flex: none;
    padding: 0.35rem 0.8rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.9rem;
    cursor: pointer;
    white-space: nowrap;
  }

  button[aria-selected='true'] {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
    font-weight: 600;
  }
</style>
