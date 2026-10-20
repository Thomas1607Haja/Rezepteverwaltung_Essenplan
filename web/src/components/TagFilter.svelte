<script>
  // Kategorie-Chips nach Gruppen. selected enthält die gewählten IDs,
  // ontoggle wird mit der ID einer angetippten Kategorie aufgerufen.
  let { groups, selected, ontoggle } = $props();
</script>

<div class="groups">
  {#each groups as group (group.name)}
    <div class="group">
      <span class="group-name">{group.name}</span>
      <div class="chips">
        {#each group.tags as tag (tag.id)}
          <button
            type="button"
            class="chip"
            aria-pressed={selected.includes(tag.id)}
            onclick={() => ontoggle(tag.id)}
          >
            {tag.name}
          </button>
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  .groups {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .group-name {
    display: block;
    margin-bottom: 0.3rem;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }

  .chip {
    padding: 0.3rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    font-size: 0.875rem;
    cursor: pointer;
  }

  .chip:hover {
    background: var(--surface-hover);
  }

  .chip[aria-pressed='true'] {
    border-color: var(--accent);
    background: var(--accent);
    color: var(--on-accent);
  }
</style>
