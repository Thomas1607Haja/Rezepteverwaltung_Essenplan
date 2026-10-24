<script module>
  /**
   * Kann dieser Browser QR-Codes mit der Kamera lesen? Nötig sind die
   * BarcodeDetector-Schnittstelle (z. B. Chrome auf Android) und Zugriff auf
   * die Kamera (nur über https oder auf dem eigenen Rechner). Sonst wird
   * der Scan-Knopf ausgeblendet.
   */
  export async function qrScanSupported() {
    if (!('BarcodeDetector' in window) || !navigator.mediaDevices?.getUserMedia) return false;
    try {
      return (await window.BarcodeDetector.getSupportedFormats()).includes('qr_code');
    } catch {
      return false;
    }
  }
</script>

<script>
  import { onMount } from 'svelte';

  // Liest einen QR-Code mit der Kamera. onresult(text) liefert den Inhalt,
  // onclose() schließt ohne Ergebnis.
  let { onresult, onclose } = $props();

  let dialog;
  let video;
  let error = $state('');

  onMount(() => {
    dialog.showModal();
    let stream = null;
    let timer = null;
    let stopped = false;

    (async () => {
      try {
        stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' }, audio: false });
        if (stopped) return;
        video.srcObject = stream;
        await video.play();
        const detector = new window.BarcodeDetector({ formats: ['qr_code'] });
        // Alle 250 ms ein Bild prüfen – genug für einen ruhig gehaltenen Ausdruck.
        timer = setInterval(async () => {
          try {
            const codes = await detector.detect(video);
            if (codes.length > 0 && !stopped) {
              stopped = true;
              onresult(codes[0].rawValue);
            }
          } catch {
            // einzelne Bilder dürfen fehlschlagen
          }
        }, 250);
      } catch (e) {
        error =
          e.name === 'NotAllowedError'
            ? 'Kein Zugriff auf die Kamera. Bitte im Browser erlauben – in der Home-Assistant-App ist die Kamera hier oft gesperrt; dann die Adresse einfügen.'
            : `Kamera nicht verfügbar (${e.message}). Bitte die Adresse einfügen.`;
      }
    })();

    // Beim Schließen Kamera und Prüfung beenden.
    return () => {
      stopped = true;
      clearInterval(timer);
      stream?.getTracks().forEach((t) => t.stop());
    };
  });
</script>

<dialog bind:this={dialog} oncancel={onclose} aria-labelledby="qr-title">
  <div class="content">
    <h2 id="qr-title">QR-Code scannen</h2>
    {#if error}
      <p class="error">{error}</p>
    {:else}
      <p class="hint">Den QR-Code des Rezepts (z. B. auf einem Chefkoch-Ausdruck) vor die Kamera halten.</p>
      <!-- muted + playsinline: sonst startet das Video auf Handys nicht von selbst -->
      <video bind:this={video} muted playsinline></video>
    {/if}
    <footer><button type="button" onclick={onclose}>Abbrechen</button></footer>
  </div>
</dialog>

<style>
  dialog {
    width: min(26rem, calc(100vw - 2rem));
    padding: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
  }

  dialog::backdrop {
    background: rgb(0 0 0 / 0.5);
  }

  .content {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 1rem;
  }

  h2 {
    margin: 0;
    font-size: 1.1rem;
  }

  p {
    margin: 0;
  }

  .hint {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .error {
    color: var(--danger);
  }

  video {
    width: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    border-radius: calc(var(--radius) - 0.2rem);
    background: #000;
  }

  footer {
    display: flex;
    justify-content: flex-end;
  }

  footer button {
    padding: 0.45rem 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }
</style>
