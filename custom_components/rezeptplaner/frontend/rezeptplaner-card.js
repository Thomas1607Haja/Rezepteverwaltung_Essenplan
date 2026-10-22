// Dashboard-Karte „Essensplan“ für Home Assistant.
//
// Zeigt den aktuellen Einkaufszeitraum: je Tag ein kleines Bild und das
// Gericht, der heutige Tag hervorgehoben. Antippen öffnet die Kochansicht
// mit den Portionen aus dem Plan. Bearbeitet wird in der Rezeptplaner-App.
//
// Die Daten kommen über die Endpunkte der Integration (/api/rezeptplaner/…),
// Bilder über signierte Adressen, weil <img> keine Anmeldung mitschicken
// kann. Keine fremden Bibliotheken: eine schlichte Web Component.

const WEEKDAYS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa'];
const RELOAD_MS = 5 * 60 * 1000; // Plan höchstens alle 5 Minuten neu holen
const RETRY_MS = 30 * 1000; // nach einem Fehler schneller neu versuchen

// Brüche wie in der Kochansicht der App (web/src/lib/format.js)
const FRACTIONS = [
  [1 / 4, '¼'],
  [1 / 3, '⅓'],
  [1 / 2, '½'],
  [2 / 3, '⅔'],
  [3 / 4, '¾'],
];

function formatDecimal(value, digits) {
  return value.toLocaleString('de-DE', { maximumFractionDigits: digits });
}

/** Menge wie in der Küche: 333.33 -> "333", 1.5 -> "1 ½", 0.33 -> "⅓". */
function formatQuantity(value) {
  if (value == null) return '';
  if (value >= 100) return String(Math.round(value));
  if (value >= 10) return formatDecimal(value, 1);
  const whole = Math.floor(value);
  const part = value - whole;
  if (part < 0.04) return String(whole);
  if (part > 0.96) return String(whole + 1);
  for (const [fraction, symbol] of FRACTIONS) {
    if (Math.abs(part - fraction) < 0.04) return whole === 0 ? symbol : `${whole} ${symbol}`;
  }
  return formatDecimal(value, 2);
}

/** Tage von from bis to (beide "2026-10-24"), eingeschlossen. */
function daysBetween(from, to) {
  const days = [];
  const d = new Date(`${from}T00:00:00Z`);
  const end = new Date(`${to}T00:00:00Z`);
  while (d <= end) {
    days.push(d.toISOString().slice(0, 10));
    d.setUTCDate(d.getUTCDate() + 1);
  }
  return days;
}

function dayLabel(date) {
  const d = new Date(`${date}T00:00:00Z`);
  return { weekday: WEEKDAYS[d.getUTCDay()], date: `${date.slice(8, 10)}.${date.slice(5, 7)}.` };
}

/** Text sicher in HTML einsetzen. */
function esc(text) {
  return String(text ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
}

const STYLE = `
  :host { display: block; }
  ha-card { display: block; padding: 12px 16px 16px; }
  header { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 8px; }
  h2 { margin: 0; font-size: 1.2rem; font-weight: 500; }
  header a { color: var(--primary-color); text-decoration: none; font-size: 0.9rem; }
  .error { color: var(--error-color); margin: 4px 0; }
  .error + .muted { margin: 0; font-size: 0.85rem; }
  .muted { color: var(--secondary-text-color); }
  ul { list-style: none; margin: 0; padding: 0; }
  li.day { display: grid; grid-template-columns: 3.2rem 1fr; gap: 8px; padding: 6px 4px; border-top: 1px solid var(--divider-color); }
  li.day:first-child { border-top: none; }
  li.today { background: color-mix(in srgb, var(--primary-color) 12%, transparent); border-radius: 8px; border-top-color: transparent; }
  li.today + li.day { border-top-color: transparent; }
  li.past { opacity: 0.55; }
  .label { display: flex; flex-direction: column; line-height: 1.2; padding-top: 4px; }
  .label b { font-weight: 600; }
  .label small { color: var(--secondary-text-color); font-size: 0.8rem; }
  .dishes { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
  .dishes .muted { padding-top: 4px; }
  button.dish { display: flex; align-items: center; gap: 10px; width: 100%; padding: 0; border: none; background: none; color: inherit; font: inherit; text-align: left; cursor: pointer; }
  button.dish:disabled { cursor: default; }
  .thumb { flex: none; width: 48px; height: 48px; border-radius: 6px; object-fit: cover; background: var(--secondary-background-color); }
  .title { min-width: 0; overflow-wrap: anywhere; }
  .title small { display: block; color: var(--secondary-text-color); font-size: 0.8rem; }

  dialog { width: min(40rem, calc(100vw - 24px)); max-height: calc(100vh - 24px); padding: 0; border: none; border-radius: 12px;
    background: var(--card-background-color, #fff); color: var(--primary-text-color); }
  dialog::backdrop { background: rgb(0 0 0 / 0.5); }
  .sheet { padding: 16px; }
  .sheet header { align-items: flex-start; }
  .sheet h2 { font-size: 1.3rem; }
  .close { flex: none; width: 36px; height: 36px; border: none; border-radius: 50%; background: none; color: inherit; font-size: 1.6rem; line-height: 1; cursor: pointer; }
  .hero { display: block; width: 100%; max-height: 16rem; object-fit: cover; border-radius: 8px; margin-bottom: 12px; }
  .servings { margin: 0 0 12px; color: var(--secondary-text-color); }
  h3 { margin: 16px 0 6px; font-size: 1.05rem; font-weight: 600; }
  h4 { margin: 10px 0 2px; font-size: 0.95rem; font-weight: 600; color: var(--secondary-text-color); }
  .ingredients li { display: grid; grid-template-columns: 6rem 1fr; gap: 8px; padding: 3px 0; }
  .qty { text-align: right; font-variant-numeric: tabular-nums; }
  .note { color: var(--secondary-text-color); }
  ol { margin: 0; padding-left: 1.4rem; }
  ol li { margin: 0 0 8px; line-height: 1.45; }
`;

// Die Klasse entsteht erst beim Anmelden (siehe register): Das Frontend
// ersetzt beim Start auch window.HTMLElement. Eine schon beim Laden der
// Datei gebildete Klasse erbte sonst vom alten HTMLElement und ließe sich
// danach nicht mehr erzeugen.
function createCardClass() {
  return class RezeptplanerCard extends HTMLElement {
    constructor() {
      super();
      // Grundgerüst einmal anlegen; _render ersetzt nur den Inhalt der Karte,
      // damit eine offene Kochansicht beim Aktualisieren erhalten bleibt.
      this.attachShadow({ mode: 'open' }).innerHTML = `<style>${STYLE}</style><ha-card></ha-card><dialog></dialog>`;
      this._card = this.shadowRoot.querySelector('ha-card');
      this._dialog = this.shadowRoot.querySelector('dialog');
      // Klick neben den Inhalt (auf den Hintergrund) oder auf × schließt.
      this._dialog.addEventListener('click', (ev) => {
        if (ev.target === this._dialog || ev.target.closest('.close')) this._dialog.close();
      });
      this._data = null;
      this._error = '';
      this._loadedAt = 0;
      this._loading = false;
      this._signed = new Map(); // Bildname -> signierte Adresse
    }

    setConfig(config) {
      this._config = config || {};
    }

    // Home Assistant setzt hass bei jeder Zustandsänderung neu. Geladen wird
    // nur beim ersten Mal und danach höchstens alle RELOAD_MS.
    set hass(hass) {
      this._hass = hass;
      if (!this._loading && Date.now() - this._loadedAt > RELOAD_MS) this._load();
    }

    connectedCallback() {
      // Beim Zurückkehren zum Dashboard (z. B. Handy aus dem Ruhezustand)
      // aktualisieren, wenn die Daten alt sind.
      this._onVisible = () => {
        if (document.visibilityState === 'visible' && this._hass && Date.now() - this._loadedAt > RELOAD_MS) this._load();
      };
      document.addEventListener('visibilitychange', this._onVisible);
    }

    disconnectedCallback() {
      document.removeEventListener('visibilitychange', this._onVisible);
      clearTimeout(this._retry);
    }

    getCardSize() {
      return 6;
    }

    static getStubConfig() {
      return {};
    }

    async _load() {
      this._loading = true;
      try {
        this._data = await this._hass.callApi('GET', 'rezeptplaner/plan');
        this._error = '';
        await this._signImages(this._data.entries.map((e) => e.recipe?.thumbnail));
      } catch (e) {
        // Für die Anzeige nur den Grund; die Karte meldet „nicht erreichbar“.
        this._error = e?.body?.error || e?.error || e?.message || 'unbekannter Fehler';
      } finally {
        this._loadedAt = Date.now();
        this._loading = false;
        this._render();
      }
      // Nach einem Fehler (Server startet, Integration noch nicht bereit)
      // nicht 5 Minuten warten, sondern bald neu versuchen.
      clearTimeout(this._retry);
      if (this._error) {
        this._retry = setTimeout(() => {
          if (this.isConnected && !this._loading) this._load();
        }, RETRY_MS);
      }
    }

    /** Holt für Bildnamen signierte Adressen (eine Stunde gültig). */
    async _signImages(names) {
      const missing = [...new Set(names.filter((n) => n && !this._signed.has(n)))];
      await Promise.all(
        missing.map(async (name) => {
          try {
            const res = await this._hass.callWS({ type: 'auth/sign_path', path: `/api/rezeptplaner/images/${name}`, expires: 3600 });
            this._signed.set(name, res.path);
          } catch {
            // Ohne Bild geht es auch.
          }
        }),
      );
      // Signaturen laufen ab; beim nächsten Laden nach 50 Minuten neu holen.
      clearTimeout(this._expire);
      this._expire = setTimeout(() => this._signed.clear(), 50 * 60 * 1000);
    }

    _img(name, cls) {
      const src = name && this._signed.get(name);
      return src ? `<img class="${cls}" src="${esc(src)}" alt="" loading="lazy">` : `<span class="${cls}"></span>`;
    }

    _render() {
      const title = this._config?.title ?? 'Essensplan';
      let body;
      if (this._error) {
        body = `<p class="error">Rezeptplaner nicht erreichbar – neuer Versuch in 30 Sekunden.</p><p class="muted">${esc(this._error)}</p>`;
      } else if (!this._data) {
        body = '<p class="muted">Lädt …</p>';
      } else {
        const { entries, today, shopping_period: period } = this._data;
        body = '<ul>';
        for (const date of daysBetween(period.from, period.to)) {
          const label = dayLabel(date);
          const dishes = entries.filter((e) => e.date === date);
          const cls = date === today ? 'day today' : date < today ? 'day past' : 'day';
          body += `<li class="${cls}"><span class="label"><b>${label.weekday}</b><small>${label.date}</small></span><div class="dishes">`;
          if (dishes.length === 0) body += '<span class="muted">–</span>';
          for (const e of dishes) {
            const name = e.recipe ? e.recipe.title : e.note;
            const info = [e.slot === 'lunch' ? 'Mittag' : '', e.recipe ? `${e.servings} Portionen` : ''].filter(Boolean).join(' · ');
            body += `<button class="dish" data-id="${e.id}" ${e.recipe_id ? '' : 'disabled'}>
              ${this._img(e.recipe?.thumbnail, 'thumb')}
              <span class="title">${esc(name)}${info ? `<small>${esc(info)}</small>` : ''}</span></button>`;
          }
          body += '</div></li>';
        }
        body += '</ul>';
      }
      const link = this._data?.app_url ? `<a href="${esc(this._data.app_url)}">Öffnen</a>` : '';
      this._card.innerHTML = `<header><h2>${esc(title)}</h2>${link}</header>${body}`;
      for (const btn of this._card.querySelectorAll('button.dish:not([disabled])')) {
        btn.addEventListener('click', () => this._open(Number(btn.dataset.id)));
      }
      // Link zur App innerhalb von Home Assistant öffnen, ohne Seite neu zu laden.
      this._card.querySelector('header a')?.addEventListener('click', (ev) => {
        ev.preventDefault();
        history.pushState(null, '', this._data.app_url);
        window.dispatchEvent(new CustomEvent('location-changed'));
      });
    }

    /** Kochansicht eines Planeintrags im Dialog. */
    async _open(entryId) {
      const entry = this._data.entries.find((e) => e.id === entryId);
      const dialog = this._dialog;
      dialog.innerHTML = `<div class="sheet"><p class="muted">Lädt …</p></div>`;
      dialog.showModal();
      try {
        const recipe = await this._hass.callApi('GET', `rezeptplaner/recipes/${entry.recipe_id}`);
        await this._signImages([recipe.image]);
        dialog.innerHTML = this._sheet(recipe, entry.servings);
      } catch (e) {
        dialog.innerHTML = `<div class="sheet"><header><h2>${esc(entry.recipe?.title)}</h2><button class="close" aria-label="Schließen">×</button></header>
          <p class="error">${esc(e?.body?.error || e?.message || 'Rezept nicht geladen')}</p></div>`;
      }
    }

    _sheet(recipe, servings) {
      const factor = servings / recipe.base_servings;
      let ingredients = '';
      let group = '';
      for (const ing of recipe.ingredients) {
        if (ing.group_name && ing.group_name !== group) {
          group = ing.group_name;
          ingredients += `</ul><h4>${esc(group)}</h4><ul class="ingredients">`;
        }
        const qty = ing.quantity == null ? '' : formatQuantity(ing.quantity * factor);
        ingredients += `<li><span class="qty">${esc([qty, ing.unit].filter(Boolean).join(' '))}</span>
          <span>${esc(ing.name)}${ing.note ? ` <span class="note">${esc(ing.note)}</span>` : ''}</span></li>`;
      }
      const steps = recipe.steps.map((s) => `<li>${esc(s.text)}</li>`).join('');
      return `<div class="sheet">
        <header><h2>${esc(recipe.title)}</h2><button class="close" aria-label="Schließen">×</button></header>
        ${recipe.image && this._signed.get(recipe.image) ? this._img(recipe.image, 'hero') : ''}
        <p class="servings">Für ${servings} ${servings === 1 ? 'Portion' : 'Portionen'}${servings !== recipe.base_servings ? ` (Rezept: ${recipe.base_servings})` : ''}</p>
        ${recipe.ingredients.length ? `<h3>Zutaten</h3><ul class="ingredients">${ingredients}</ul>` : ''}
        ${steps ? `<h3>Zubereitung</h3><ol>${steps}</ol>` : ''}
      </div>`;
    }
  };
}

// Anmelden erst, wenn das Frontend von Home Assistant läuft. Es tauscht
// beim Start window.customElements aus (Polyfill für „scoped registries“).
// Lief die Karte vorher – möglich, wenn sie früh geladen wird, z. B. bei
// einem harten Neuladen –, landete sie in der alten Registry, und das
// Dashboard meldete „Custom element doesn't exist“. Das Element
// <home-assistant> definiert das Frontend erst nach dem Austausch; danach
// ist window.customElements die richtige Registry.
function register() {
  try {
    if (!window.customElements.get('rezeptplaner-card')) {
      window.customElements.define('rezeptplaner-card', createCardClass());
    }
  } catch (err) {
    console.error('Rezeptplaner-Karte: Anmelden fehlgeschlagen', err);
  }
}
if (customElements.get('home-assistant')) {
  register();
} else {
  customElements.whenDefined('home-assistant').then(register);
  // Notnagel, falls die Karte einmal außerhalb der normalen Oberfläche läuft.
  setTimeout(register, 10000);
}

// Banner in der Konsole: zeigt sofort, ob und welche Fassung geladen ist.
// Die Version steht im Parameter v der Adresse (siehe Integration).
const VERSION = new URL(import.meta.url).searchParams.get('v') || 'unbekannt';
console.info(`%c REZEPTPLANER-KARTE %c ${VERSION} `, 'color: white; background: #c2552d; font-weight: 700;', 'color: #c2552d;');

// Eintrag in der Kartenauswahl des Dashboard-Editors
window.customCards = window.customCards || [];
if (!window.customCards.some((c) => c.type === 'rezeptplaner-card')) {
  window.customCards.push({
    type: 'rezeptplaner-card',
    name: 'Essensplan',
    description: 'Aktueller Einkaufszeitraum aus dem Rezeptplaner mit Kochansicht',
  });
}
