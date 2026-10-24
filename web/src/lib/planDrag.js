// Drag and Drop im Essensplan als Svelte-Action: <div use:planDrag={…}>.
//
// Regeln (docs/PLANUNG.md, „Verschieben und Tauschen“): Ziehen auf ein
// Gericht tauscht die beiden, Ziehen auf die freie Fläche eines Tages hängt
// das Gericht dort an. Dafür passt keine Sortier-Bibliothek, daher diese
// kleine eigene Umsetzung mit Pointer-Events (Maus, Finger und Stift).
//
// Erwartete Markierungen im HTML:
//   data-entry-id, data-date, data-slot  an jedem Eintrag
//   data-date                            an jeder Tagesspalte
//   data-slot-zone="lunch" | "coffee" | "dinner"    an den Bereichen Mittag/Kaffee/Abend
//
// Optionen: { ondrop(entryId, target), ontarget(target | null) }, wobei
// target = { type: 'swap', id, date, slot } oder { type: 'append', date, slot }.

const LONG_PRESS_MS = 400; // Finger: so lange halten, bis das Ziehen beginnt
const TOUCH_SLOP = 10; // Finger: mehr Bewegung vorher = normales Scrollen
const MOUSE_SLOP = 6; // Maus: ab dieser Strecke beginnt das Ziehen
const EDGE = 48; // Abstand zum Rand, ab dem der Plan mitscrollt (px)
const EDGE_SPEED = 14; // Scrollgeschwindigkeit am Rand (px je Bild)

export function planDrag(strip, options) {
  let opts = options;
  let press = null; // gedrückt, Ziehen noch nicht begonnen
  let drag = null; // Ziehen läuft
  let suppressClick = false; // Klick nach dem Ablegen nicht als Klick werten

  function onPointerDown(e) {
    if (e.button !== 0 || drag) return;
    const entryEl = e.target.closest('[data-entry-id]');
    // Der Menüknopf soll sich normal anklicken lassen.
    if (!entryEl || e.target.closest('button.menu')) return;
    press = {
      entryEl,
      id: Number(entryEl.dataset.entryId),
      startX: e.clientX,
      startY: e.clientY,
      pointerId: e.pointerId,
      touch: e.pointerType !== 'mouse',
      timer: null,
    };
    if (press.touch) press.timer = setTimeout(() => startDrag(press.startX, press.startY), LONG_PRESS_MS);
    window.addEventListener('pointermove', onPointerMove);
    window.addEventListener('pointerup', onPointerUp);
    window.addEventListener('pointercancel', cancel);
  }

  function onPointerMove(e) {
    if (drag) {
      if (e.pointerId !== drag.pointerId) return;
      drag.x = e.clientX;
      drag.y = e.clientY;
      update();
      return;
    }
    if (!press || e.pointerId !== press.pointerId) return;
    const distance = Math.hypot(e.clientX - press.startX, e.clientY - press.startY);
    if (press.touch && distance > TOUCH_SLOP) cleanup(); // der Finger scrollt
    else if (!press.touch && distance > MOUSE_SLOP) startDrag(e.clientX, e.clientY);
  }

  function startDrag(x, y) {
    if (!press) return;
    const rect = press.entryEl.getBoundingClientRect();
    // Eine Kopie des Eintrags folgt dem Zeiger; das Original bleibt blass
    // an seinem Platz stehen.
    const ghost = press.entryEl.cloneNode(true);
    Object.assign(ghost.style, {
      position: 'fixed',
      left: `${rect.left}px`,
      top: `${rect.top}px`,
      width: `${rect.width}px`,
      margin: '0',
      zIndex: '1000',
      pointerEvents: 'none', // sonst fände elementFromPoint nur die Kopie
      boxShadow: '0 8px 24px rgb(0 0 0 / 0.25)',
      opacity: '0.95',
    });
    document.body.append(ghost);
    press.entryEl.classList.add('drag-source');
    document.body.classList.add('plan-dragging');
    navigator.vibrate?.(30); // kurzes Rütteln als Zeichen auf dem Handy

    drag = { ...press, ghost, x, y, target: null, frame: 0 };
    press = null;
    update();
    drag.frame = requestAnimationFrame(autoScroll);
  }

  // Kopie verschieben und Ziel bestimmen.
  function update() {
    const { ghost, x, y, startX, startY } = drag;
    ghost.style.transform = `translate(${x - startX}px, ${y - startY}px) rotate(2deg)`;
    const target = findTarget(x, y);
    if (!sameTarget(target, drag.target)) {
      drag.target = target;
      opts.ontarget?.(target);
    }
  }

  function findTarget(x, y) {
    const el = document.elementFromPoint(x, y);
    if (!el || !strip.contains(el)) return null;
    const entryEl = el.closest('[data-entry-id]');
    if (entryEl && entryEl !== drag.entryEl) {
      return { type: 'swap', id: Number(entryEl.dataset.entryId), date: entryEl.dataset.date, slot: entryEl.dataset.slot };
    }
    const day = el.closest('[data-date]');
    if (!day || entryEl === drag.entryEl) return null;
    const zone = el.closest('[data-slot-zone]');
    return { type: 'append', date: day.dataset.date, slot: zone?.dataset.slotZone ?? 'dinner' };
  }

  // Nahe am linken/rechten Rand scrollt der Plan mit, solange gezogen wird.
  function autoScroll() {
    if (!drag) return;
    const r = strip.getBoundingClientRect();
    let delta = 0;
    if (drag.x < r.left + EDGE) delta = -EDGE_SPEED;
    else if (drag.x > r.right - EDGE) delta = EDGE_SPEED;
    if (delta) {
      strip.scrollLeft += delta;
      update(); // unter dem Zeiger liegt jetzt ein anderer Tag
    }
    drag.frame = requestAnimationFrame(autoScroll);
  }

  function onPointerUp(e) {
    if (press && e.pointerId === press.pointerId) {
      cleanup(); // nur geklickt bzw. getippt
      return;
    }
    if (!drag || e.pointerId !== drag.pointerId) return;
    const { id, target } = drag;
    suppressClick = true;
    setTimeout(() => (suppressClick = false), 0);
    cleanup();
    if (target) opts.ondrop(id, target);
  }

  function cancel() {
    cleanup();
  }

  function cleanup() {
    if (press?.timer) clearTimeout(press.timer);
    press = null;
    if (drag) {
      cancelAnimationFrame(drag.frame);
      drag.ghost.remove();
      drag.entryEl.classList.remove('drag-source');
      document.body.classList.remove('plan-dragging');
      drag = null;
      opts.ontarget?.(null);
    }
    window.removeEventListener('pointermove', onPointerMove);
    window.removeEventListener('pointerup', onPointerUp);
    window.removeEventListener('pointercancel', cancel);
  }

  // Ein Link (Eintrag mit Rezept) soll nach dem Ablegen nicht öffnen.
  function onClick(e) {
    if (suppressClick) {
      e.preventDefault();
      e.stopPropagation();
      suppressClick = false;
    }
  }

  // Während des Ziehens darf der Finger nicht scrollen. Dafür muss der
  // Listener ausdrücklich nicht "passive" sein.
  function onTouchMove(e) {
    if (drag) e.preventDefault();
  }

  // Langes Drücken öffnet auf dem Handy sonst das Kontextmenü des Links.
  function onContextMenu(e) {
    if (press?.touch || drag) e.preventDefault();
  }

  function onKeyDown(e) {
    if (e.key === 'Escape' && drag) cleanup();
  }

  strip.addEventListener('pointerdown', onPointerDown);
  strip.addEventListener('click', onClick, true);
  strip.addEventListener('touchmove', onTouchMove, { passive: false });
  strip.addEventListener('contextmenu', onContextMenu);
  // Links und Bilder zieht der Browser sonst selbst (natives Drag and Drop).
  strip.addEventListener('dragstart', preventDefault);
  window.addEventListener('keydown', onKeyDown);

  return {
    update(newOptions) {
      opts = newOptions;
    },
    destroy() {
      cleanup();
      strip.removeEventListener('pointerdown', onPointerDown);
      strip.removeEventListener('click', onClick, true);
      strip.removeEventListener('touchmove', onTouchMove);
      strip.removeEventListener('contextmenu', onContextMenu);
      strip.removeEventListener('dragstart', preventDefault);
      window.removeEventListener('keydown', onKeyDown);
    },
  };
}

function preventDefault(e) {
  e.preventDefault();
}

function sameTarget(a, b) {
  if (!a || !b) return a === b;
  return a.type === b.type && a.id === b.id && a.date === b.date && a.slot === b.slot;
}
