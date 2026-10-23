<script>
  import { encode } from 'uqr';

  // QR-Code als SVG. Die Module werden zu einem einzigen Pfad
  // zusammengefasst (ein Quadrat je dunklem Modul); weißer Rand und
  // schwarze Module unabhängig vom Farbschema, damit jede Kamera ihn liest.
  let { text, size = 220, label = 'QR-Code' } = $props();

  const qr = $derived(encode(text, { ecc: 'M', border: 2 }));
  const path = $derived.by(() => {
    let d = '';
    qr.data.forEach((row, y) =>
      row.forEach((dark, x) => {
        if (dark) d += `M${x} ${y}h1v1h-1z`;
      }),
    );
    return d;
  });
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 {qr.size} {qr.size}"
  shape-rendering="crispEdges"
  role="img"
  aria-label={label}
>
  <rect width={qr.size} height={qr.size} fill="#fff" />
  <path d={path} fill="#000" />
</svg>

<style>
  svg {
    display: block;
    max-width: 100%;
    height: auto;
  }
</style>
