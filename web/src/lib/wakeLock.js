// Hält den Bildschirm eingeschaltet, solange die Kochansicht offen ist.
//
// Die Wake Lock API gibt es nur in sicheren Umgebungen (HTTPS oder
// localhost). Fehlt sie, passiert einfach nichts.

/**
 * Fordert die Sperre an und liefert eine Funktion, die sie wieder freigibt.
 * Der Browser gibt die Sperre selbst frei, wenn der Tab in den Hintergrund
 * geht; beim Zurückkehren wird sie daher neu angefordert.
 */
export function keepScreenOn() {
  if (!('wakeLock' in navigator)) return () => {};

  let sentinel = null;
  let released = false;

  async function acquire() {
    try {
      sentinel = await navigator.wakeLock.request('screen');
      // Seite schon verlassen, während die Anfrage lief? Dann sofort freigeben.
      if (released) sentinel.release().catch(() => {});
    } catch {
      // z. B. Energiesparmodus: ohne Sperre weitermachen
    }
  }

  function onVisibilityChange() {
    if (!released && document.visibilityState === 'visible') acquire();
  }

  acquire();
  document.addEventListener('visibilitychange', onVisibilityChange);

  return () => {
    released = true;
    document.removeEventListener('visibilitychange', onVisibilityChange);
    sentinel?.release().catch(() => {});
  };
}
