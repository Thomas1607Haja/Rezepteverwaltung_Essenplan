<script>
  // Schlichte Linien-Symbole als SVG, ohne eigene Icon-Bibliothek.
  // Jedes Symbol ist eine Liste von Pfaden im 24×24-Raster.
  // filled füllt die Fläche aus (z. B. für ein gesetztes Herz).
  let { name, size = 22, filled = false } = $props();

  const paths = {
    calendar: ['M4 6h16v14H4z', 'M4 10h16', 'M8 3v4', 'M16 3v4'],
    book: ['M5 4h11a3 3 0 0 1 3 3v13H8a3 3 0 0 1-3-3z', 'M5 17a3 3 0 0 1 3-3h11'],
    cart: ['M3 4h2l2.5 11h10.5l2-8H6.2', 'M9 20h.01', 'M17 20h.01'],
    gear: [
      'M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z',
      'M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1 7 17M17 7l2.1-2.1',
    ],
    heart: ['M12 20s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 10c0 5.6-7 10-7 10z'],
    search: ['M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14z', 'M20 20l-4-4'],
    filter: ['M4 5h16l-6 8v6l-4-2v-4z'],
    clock: ['M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18z', 'M12 7v5l3 2'],
    brush: ['M14 4l6 6-8 8H6v-6z', 'M4 20h4'],
    dish: ['M3 15h18', 'M5 15a7 7 0 0 1 14 0', 'M12 6v2', 'M4 19h16'],
    check: ['M5 12.5l4.5 4.5L19 7.5'],
    chart: ['M4 20h16', 'M7 16v-5', 'M12 16V6', 'M17 16v-8'],
    sparkle: ['M12 3l2 6 6 2-6 2-2 6-2-6-6-2 6-2z'],
    'chevron-left': ['M15 5l-7 7 7 7'],
    'chevron-right': ['M9 5l7 7-7 7'],
  };
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill={filled ? 'currentColor' : 'none'}
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
