<script>
  import { dragHandle } from 'svelte-dnd-action';

  // Griff zum Verschieben eines Kärtchens. Nur hier lässt sich ziehen,
  // damit Tippen und Markieren in den Eingabefeldern ungestört bleibt.
  let { label } = $props();
</script>

<span class="handle" use:dragHandle aria-label={label} title="Zum Verschieben ziehen">
  <svg width="16" height="20" viewBox="0 0 16 20" aria-hidden="true">
    {#each [4, 10, 16] as y (y)}
      <circle cx="5" cy={y} r="1.6" />
      <circle cx="11" cy={y} r="1.6" />
    {/each}
  </svg>
</span>

<style>
  .handle {
    display: inline-grid;
    place-items: center;
    width: 1.75rem;
    height: 2rem;
    border-radius: calc(var(--radius) - 0.2rem);
    color: var(--text-muted);
    fill: currentColor;
    cursor: grab;
    /* Auf Touch-Geräten soll der Finger am Griff ziehen, nicht scrollen. */
    touch-action: none;
  }

  .handle:hover,
  .handle:focus-visible {
    background: var(--surface-hover);
    color: var(--accent);
  }

  .handle:active {
    cursor: grabbing;
  }
</style>
