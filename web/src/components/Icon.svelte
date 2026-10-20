<script>
  // Schlichte Linien-Symbole als SVG, ohne eigene Icon-Bibliothek.
  // Jedes Symbol ist eine Liste von Pfaden im 24×24-Raster.
  let { name, size = 22 } = $props();

  const paths = {
    calendar: ['M4 6h16v14H4z', 'M4 10h16', 'M8 3v4', 'M16 3v4'],
    book: ['M5 4h11a3 3 0 0 1 3 3v13H8a3 3 0 0 1-3-3z', 'M5 17a3 3 0 0 1 3-3h11'],
    cart: ['M3 4h2l2.5 11h10.5l2-8H6.2', 'M9 20h.01', 'M17 20h.01'],
    gear: [
      'M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z',
      'M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1 7 17M17 7l2.1-2.1',
    ],
  };
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="2"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#each paths[name] ?? [] as d (d)}
    <path {d} />
  {/each}
</svg>
