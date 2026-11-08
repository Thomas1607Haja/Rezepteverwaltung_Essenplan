// Dashboard-Karte „Essensplan“ für Home Assistant.
//
// Zeigt den aktuellen Einkaufszeitraum oder die nächsten Tage ab heute
// (Einstellung days): je Tag ein kleines Bild und das Gericht, der heutige
// Tag hervorgehoben. Mit fester Höhe (height) scrollt der Inhalt. Antippen öffnet die Kochansicht
// mit den Portionen aus dem Plan. Bearbeitet wird in der Rezeptplaner-App.
//
// Die Daten kommen über die Endpunkte der Integration (/api/rezeptplaner/…),
// Bilder über signierte Adressen, weil <img> keine Anmeldung mitschicken
// kann. Keine fremden Bibliotheken: eine schlichte Web Component.

// Texte je Sprache. Reines JSON zwischen den Markierungen: Ein Go-Test
// (server/internal/locales) prüft, dass alle Sprachen dieselben Schlüssel
// haben. Es gilt die Sprache des angemeldeten Nutzers, Rückfall Englisch.
// i18n:begin
const TEXTS = {
  "en": {
    "card_name": "Meal plan",
    "card_description": "Meal plan from Rezeptplaner with cooking view – shopping period or the next days",
    "unreachable": "Rezeptplaner not reachable – trying again in 30 seconds.",
    "unknown_error": "unknown error",
    "loading": "Loading …",
    "open": "Open",
    "close": "Close",
    "lunch": "Lunch",
    "coffee": "Tea",
    "courses": {
      "one": "{count} course",
      "other": "{count} courses"
    },
    "menu_title": "Menu: {title}",
    "menu": "Menu",
    "servings": {
      "one": "{count} serving",
      "other": "{count} servings"
    },
    "for_amount": "For {amount}",
    "recipe_base": "(recipe: {count})",
    "ingredients": "Ingredients",
    "steps": "Steps",
    "recipe_failed": "Recipe not loaded",
    "editor_title": "Title",
    "editor_days": "Number of days from today",
    "editor_height": "Card height",
    "helper_title": "Empty: “{name}”",
    "helper_days": "1 to {max}. Empty: current shopping period",
    "helper_height": "In pixels. Empty: as tall as the content. If not everything fits, the card scrolls.",
    "error_days": "days must be a whole number from 1 to {max}",
    "error_height": "height must be a number from {min} (pixels)"
  },
  "de": {
    "card_name": "Essensplan",
    "card_description": "Essensplan aus dem Rezeptplaner mit Kochansicht – Einkaufszeitraum oder die nächsten Tage",
    "unreachable": "Rezeptplaner nicht erreichbar – neuer Versuch in 30 Sekunden.",
    "unknown_error": "unbekannter Fehler",
    "loading": "Lädt …",
    "open": "Öffnen",
    "close": "Schließen",
    "lunch": "Mittag",
    "coffee": "Kaffee",
    "courses": {
      "one": "{count} Gang",
      "other": "{count} Gänge"
    },
    "menu_title": "Menü: {title}",
    "menu": "Menü",
    "servings": {
      "one": "{count} Portion",
      "other": "{count} Portionen"
    },
    "for_amount": "Für {amount}",
    "recipe_base": "(Rezept: {count})",
    "ingredients": "Zutaten",
    "steps": "Zubereitung",
    "recipe_failed": "Rezept nicht geladen",
    "editor_title": "Titel",
    "editor_days": "Anzahl Tage ab heute",
    "editor_height": "Höhe der Karte",
    "helper_title": "Leer: „{name}“",
    "helper_days": "1 bis {max}. Leer: aktueller Einkaufszeitraum",
    "helper_height": "In Pixel. Leer: so hoch wie der Inhalt. Passt nicht alles hinein, wird innerhalb der Karte gescrollt.",
    "error_days": "days muss eine ganze Zahl von 1 bis {max} sein",
    "error_height": "height muss eine Zahl ab {min} (Pixel) sein"
  },
  "fr": {
    "card_name": "Planning des repas",
    "card_description": "Planning des repas de Rezeptplaner avec vue cuisine – période de courses ou prochains jours",
    "unreachable": "Rezeptplaner injoignable – nouvel essai dans 30 secondes.",
    "unknown_error": "erreur inconnue",
    "loading": "Chargement …",
    "open": "Ouvrir",
    "close": "Fermer",
    "lunch": "Déjeuner",
    "coffee": "Goûter",
    "courses": {
      "one": "{count} plat",
      "other": "{count} plats"
    },
    "menu_title": "Menu : {title}",
    "menu": "Menu",
    "servings": {
      "one": "{count} portion",
      "other": "{count} portions"
    },
    "for_amount": "Pour {amount}",
    "recipe_base": "(recette : {count})",
    "ingredients": "Ingrédients",
    "steps": "Préparation",
    "recipe_failed": "Recette non chargée",
    "editor_title": "Titre",
    "editor_days": "Nombre de jours à partir d'aujourd'hui",
    "editor_height": "Hauteur de la carte",
    "helper_title": "Vide : « {name} »",
    "helper_days": "1 à {max}. Vide : période de courses actuelle",
    "helper_height": "En pixels. Vide : aussi haute que le contenu. Si tout ne tient pas, la carte défile.",
    "error_days": "days doit être un nombre entier de 1 à {max}",
    "error_height": "height doit être un nombre à partir de {min} (pixels)"
  },
  "es": {
    "card_name": "Plan de comidas",
    "card_description": "Plan de comidas de Rezeptplaner con vista de cocina: periodo de compra o los próximos días",
    "unreachable": "No se puede conectar con Rezeptplaner: nuevo intento en 30 segundos.",
    "unknown_error": "error desconocido",
    "loading": "Cargando…",
    "open": "Abrir",
    "close": "Cerrar",
    "lunch": "Comida",
    "coffee": "Merienda",
    "courses": {
      "one": "{count} plato",
      "other": "{count} platos"
    },
    "menu_title": "Menú: {title}",
    "menu": "Menú",
    "servings": {
      "one": "{count} ración",
      "other": "{count} raciones"
    },
    "for_amount": "Para {amount}",
    "recipe_base": "(receta: {count})",
    "ingredients": "Ingredientes",
    "steps": "Preparación",
    "recipe_failed": "Receta no cargada",
    "editor_title": "Título",
    "editor_days": "Número de días desde hoy",
    "editor_height": "Altura de la tarjeta",
    "helper_title": "Vacío: «{name}»",
    "helper_days": "1 a {max}. Vacío: periodo de compra actual",
    "helper_height": "En píxeles. Vacío: tan alta como el contenido. Si no cabe todo, la tarjeta se desplaza.",
    "error_days": "days debe ser un número entero de 1 a {max}",
    "error_height": "height debe ser un número desde {min} (píxeles)"
  },
  "it": {
    "card_name": "Piano pasti",
    "card_description": "Piano pasti di Rezeptplaner con vista cucina – periodo della spesa o i prossimi giorni",
    "unreachable": "Rezeptplaner non raggiungibile – nuovo tentativo tra 30 secondi.",
    "unknown_error": "errore sconosciuto",
    "loading": "Caricamento…",
    "open": "Apri",
    "close": "Chiudi",
    "lunch": "Pranzo",
    "coffee": "Merenda",
    "courses": {
      "one": "{count} portata",
      "other": "{count} portate"
    },
    "menu_title": "Menù: {title}",
    "menu": "Menù",
    "servings": {
      "one": "{count} porzione",
      "other": "{count} porzioni"
    },
    "for_amount": "Per {amount}",
    "recipe_base": "(ricetta: {count})",
    "ingredients": "Ingredienti",
    "steps": "Preparazione",
    "recipe_failed": "Ricetta non caricata",
    "editor_title": "Titolo",
    "editor_days": "Numero di giorni da oggi",
    "editor_height": "Altezza della scheda",
    "helper_title": "Vuoto: «{name}»",
    "helper_days": "Da 1 a {max}. Vuoto: periodo della spesa attuale",
    "helper_height": "In pixel. Vuoto: alta quanto il contenuto. Se non entra tutto, la scheda scorre.",
    "error_days": "days deve essere un numero intero da 1 a {max}",
    "error_height": "height deve essere un numero da {min} (pixel)"
  }
};
// i18n:end

const REGIONS = { en: 'en-US', de: 'de-DE', fr: 'fr-FR', es: 'es-ES', it: 'it-IT' };

/** Sprache des Nutzers aus hass (oder der Seite), sonst Englisch. */
function languageOf(hass) {
  const tag = hass?.locale?.language || hass?.language || document.documentElement.lang || 'en';
  const base = String(tag).slice(0, 2).toLowerCase();
  return TEXTS[base] ? base : 'en';
}

/** Locale für Zahlen und Daten: die genaue Angabe aus hass, wenn sie passt. */
function localeOf(hass) {
  const lang = languageOf(hass);
  const tag = hass?.locale?.language || hass?.language || '';
  return tag.toLowerCase().startsWith(lang) && tag.length > 2 ? tag : REGIONS[lang];
}

/** Text in der Sprache lang; params ersetzen {name}, count wählt die Pluralform. */
function tr(lang, key, params = {}) {
  let value = TEXTS[lang]?.[key] ?? TEXTS.en[key] ?? key;
  if (typeof value === 'object') {
    const form = new Intl.PluralRules(REGIONS[lang]).select(Number(params.count ?? 0));
    value = value[form] ?? value.other;
  }
  return value.replace(/\{(\w+)\}/g, (all, name) => (name in params ? String(params[name]) : all));
}
const RELOAD_MS = 5 * 60 * 1000; // Plan höchstens alle 5 Minuten neu holen
const RETRY_MS = 30 * 1000; // nach einem Fehler schneller neu versuchen
const MAX_DAYS = 31; // wie MAX_DAYS in views.py
const MIN_HEIGHT = 100;

// Brüche wie in der Kochansicht der App (web/src/lib/format.js)
const FRACTIONS = [
  [1 / 4, '¼'],
  [1 / 3, '⅓'],
  [1 / 2, '½'],
  [2 / 3, '⅔'],
  [3 / 4, '¾'],
];

// „4 Portionen“ bzw. mit Ergebnis-Einheit „1 Springform 26 cm“ (Rezeptplaner 0.14).
function amountText(lang, n, unit) {
  if (unit) return `${n} ${unit}`;
  return tr(lang, 'servings', { count: n });
}

function formatDecimal(value, digits, locale) {
  return value.toLocaleString(locale, { maximumFractionDigits: digits });
}

/** Menge wie in der Küche: 333.33 -> "333", 1.5 -> "1 ½", 0.33 -> "⅓". */
function formatQuantity(value, locale) {
  if (value == null) return '';
  if (value >= 100) return String(Math.round(value));
  if (value >= 10) return formatDecimal(value, 1, locale);
  const whole = Math.floor(value);
  const part = value - whole;
  if (part < 0.04) return String(whole);
  if (part > 0.96) return String(whole + 1);
  for (const [fraction, symbol] of FRACTIONS) {
    if (Math.abs(part - fraction) < 0.04) return whole === 0 ? symbol : `${whole} ${symbol}`;
  }
  return formatDecimal(value, 2, locale);
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

/** Wochentag und Datum in der Schreibweise der Sprache („Mi“ „21.10.“ bzw. „Wed“ „10/21“). */
function dayLabel(date, locale) {
  const d = new Date(`${date}T00:00:00Z`);
  const weekday = new Intl.DateTimeFormat(locale, { weekday: 'short', timeZone: 'UTC' }).format(d).replace(/\.$/, '');
  let day = new Intl.DateTimeFormat(locale, { day: '2-digit', month: '2-digit', timeZone: 'UTC' }).format(d);
  if (locale.startsWith('de') && !day.endsWith('.')) day += '.';
  return { weekday, date: day };
}

/** Text sicher in HTML einsetzen. */
function esc(text) {
  return String(text ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
}

const STYLE = `
  :host { display: block; }
  ha-card { display: block; padding: 12px 16px 16px; box-sizing: border-box; }
  /* Feste Höhe: Überschrift bleibt stehen, die Tage scrollen. */
  ha-card.fixed { display: flex; flex-direction: column; overflow: hidden; }
  ha-card.fixed .content { flex: 1; min-height: 0; overflow-y: auto; overscroll-behavior: contain; }
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

    // Einstellungen (auch aus dem visuellen Editor):
    //   title  – Überschrift, Standard „Essensplan“
    //   days   – Anzahl Tage ab heute; ohne Angabe der Einkaufszeitraum
    //   height – Höhe in Pixel; ohne Angabe so hoch wie der Inhalt
    setConfig(config) {
      config = config || {};
      if (config.days != null && !(Number.isInteger(config.days) && config.days >= 1 && config.days <= MAX_DAYS)) {
        throw new Error(tr(languageOf(this._hass), 'error_days', { max: MAX_DAYS }));
      }
      if (config.height != null && !(Number.isFinite(config.height) && config.height >= MIN_HEIGHT)) {
        throw new Error(tr(languageOf(this._hass), 'error_height', { min: MIN_HEIGHT }));
      }
      const reload = this._config && this._config.days !== config.days;
      this._config = config;
      this._card.classList.toggle('fixed', config.height != null);
      this._card.style.height = config.height != null ? `${config.height}px` : '';
      if (reload) {
        // Anderer Zeitraum: gleich neu laden (z. B. beim Ändern im Editor).
        this._data = null;
        this._loadedAt = 0;
        if (this._hass && !this._loading) this._load();
      }
      if (this._hass) this._render();
    }

    // Home Assistant setzt hass bei jeder Zustandsänderung neu. Geladen wird
    // nur beim ersten Mal und danach höchstens alle RELOAD_MS.
    set hass(hass) {
      const changed = languageOf(hass) !== languageOf(this._hass);
      this._hass = hass;
      if (!this._loading && Date.now() - this._loadedAt > RELOAD_MS) this._load();
      // Sprache des Nutzers gewechselt: gleich neu zeichnen.
      else if (changed) this._render();
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
      // Eine Einheit entspricht etwa 50 Pixeln.
      return this._config?.height ? Math.ceil(this._config.height / 50) : 6;
    }

    static getStubConfig() {
      return {};
    }

    static getConfigElement() {
      return document.createElement('rezeptplaner-card-editor');
    }

    async _load() {
      this._loading = true;
      try {
        const days = this._config?.days;
        this._data = await this._hass.callApi('GET', days ? `rezeptplaner/plan?days=${days}` : 'rezeptplaner/plan');
        this._error = '';
        await this._signImages(this._data.entries.map((e) => e.recipe?.thumbnail));
      } catch (e) {
        // Für die Anzeige nur den Grund; die Karte meldet „nicht erreichbar“.
        this._error = e?.body?.error || e?.error || e?.message || tr(languageOf(this._hass), 'unknown_error');
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
      const lang = languageOf(this._hass);
      const locale = localeOf(this._hass);
      const title = this._config?.title ?? tr(lang, 'card_name');
      const slotName = (slot) => (slot === 'lunch' || slot === 'coffee' ? tr(lang, slot) : '');
      let body;
      if (this._error) {
        body = `<p class="error">${esc(tr(lang, 'unreachable'))}</p><p class="muted">${esc(this._error)}</p>`;
      } else if (!this._data) {
        body = `<p class="muted">${esc(tr(lang, 'loading'))}</p>`;
      } else {
        // range: gezeigter Zeitraum (ältere Integrationen kennen nur shopping_period)
        const { entries, today } = this._data;
        const period = this._data.range ?? this._data.shopping_period;
        body = '<ul>';
        for (const date of daysBetween(period.from, period.to)) {
          const label = dayLabel(date, locale);
          // Gänge eines Menüs (Rezeptplaner 0.16) stehen in seiner Zeile.
          const dishes = entries.filter((e) => e.date === date && !e.menu_id);
          const menus = (this._data.menus ?? []).filter((m) => m.date === date);
          const cls = date === today ? 'day today' : date < today ? 'day past' : 'day';
          body += `<li class="${cls}"><span class="label"><b>${label.weekday}</b><small>${label.date}</small></span><div class="dishes">`;
          if (dishes.length === 0 && menus.length === 0) body += '<span class="muted">–</span>';
          for (const m of menus) {
            const courses = entries.filter((e) => e.menu_id === m.id);
            const slot = slotName(m.slot);
            const info = [slot, tr(lang, 'courses', { count: courses.length }), ...courses.map((e) => e.recipe ? e.recipe.title : e.note)]
              .filter(Boolean).join(' · ');
            body += `<button class="dish" disabled>${this._img(courses.find((e) => e.recipe?.thumbnail)?.recipe.thumbnail, 'thumb')}
              <span class="title">${esc(tr(lang, 'menu_title', { title: m.display_title || tr(lang, 'menu') }))}<small>${esc(info)}</small></span></button>`;
          }
          for (const e of dishes) {
            const name = e.recipe ? e.recipe.title : e.note;
            const slot = slotName(e.slot);
            const info = [slot, e.recipe ? amountText(lang, e.servings, e.recipe.yield_unit) : ''].filter(Boolean).join(' · ');
            body += `<button class="dish" data-id="${e.id}" ${e.recipe_id ? '' : 'disabled'}>
              ${this._img(e.recipe?.thumbnail, 'thumb')}
              <span class="title">${esc(name)}${info ? `<small>${esc(info)}</small>` : ''}</span></button>`;
          }
          body += '</div></li>';
        }
        body += '</ul>';
      }
      const link = this._data?.app_url ? `<a href="${esc(this._data.app_url)}">${esc(tr(lang, 'open'))}</a>` : '';
      this._card.innerHTML = `<header><h2>${esc(title)}</h2>${link}</header><div class="content">${body}</div>`;
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
      const lang = languageOf(this._hass);
      dialog.innerHTML = `<div class="sheet"><p class="muted">${esc(tr(lang, 'loading'))}</p></div>`;
      dialog.showModal();
      try {
        const recipe = await this._hass.callApi('GET', `rezeptplaner/recipes/${entry.recipe_id}`);
        await this._signImages([recipe.image]);
        dialog.innerHTML = this._sheet(recipe, entry.servings);
      } catch (e) {
        dialog.innerHTML = `<div class="sheet"><header><h2>${esc(entry.recipe?.title)}</h2><button class="close" aria-label="${esc(tr(lang, 'close'))}">×</button></header>
          <p class="error">${esc(e?.body?.error || e?.message || tr(lang, 'recipe_failed'))}</p></div>`;
      }
    }

    _sheet(recipe, servings) {
      const lang = languageOf(this._hass);
      const locale = localeOf(this._hass);
      const factor = servings / recipe.base_servings;
      let ingredients = '';
      let group = '';
      for (const ing of recipe.ingredients) {
        if (ing.group_name && ing.group_name !== group) {
          group = ing.group_name;
          ingredients += `</ul><h4>${esc(group)}</h4><ul class="ingredients">`;
        }
        const qty = ing.quantity == null ? '' : formatQuantity(ing.quantity * factor, locale);
        ingredients += `<li><span class="qty">${esc([qty, ing.unit].filter(Boolean).join(' '))}</span>
          <span>${esc(ing.name)}${ing.note ? ` <span class="note">${esc(ing.note)}</span>` : ''}</span></li>`;
      }
      const steps = recipe.steps.map((s) => `<li>${esc(s.text)}</li>`).join('');
      return `<div class="sheet">
        <header><h2>${esc(recipe.title)}</h2><button class="close" aria-label="${esc(tr(lang, 'close'))}">×</button></header>
        ${recipe.image && this._signed.get(recipe.image) ? this._img(recipe.image, 'hero') : ''}
        <p class="servings">${esc(tr(lang, 'for_amount', { amount: amountText(lang, servings, recipe.yield_unit) }))}${servings !== recipe.base_servings ? ` ${esc(tr(lang, 'recipe_base', { count: recipe.base_servings }))}` : ''}</p>
        ${recipe.ingredients.length ? `<h3>${esc(tr(lang, 'ingredients'))}</h3><ul class="ingredients">${ingredients}</ul>` : ''}
        ${steps ? `<h3>${esc(tr(lang, 'steps'))}</h3><ol>${steps}</ol>` : ''}
      </div>`;
    }
  };
}

// Visueller Editor im Dashboard: dieselben Formularfelder wie bei den
// eingebauten Karten (ha-form von Home Assistant).
const EDITOR_SCHEMA = [
  { name: 'title', selector: { text: {} } },
  { name: 'days', selector: { number: { min: 1, max: MAX_DAYS, step: 1, mode: 'box' } } },
  { name: 'height', selector: { number: { min: MIN_HEIGHT, max: 3000, step: 10, mode: 'box', unit_of_measurement: 'px' } } },
];
const editorLabel = (lang, name) => tr(lang, `editor_${name}`);
const editorHelper = (lang, name) =>
  ({
    title: tr(lang, 'helper_title', { name: tr(lang, 'card_name') }),
    days: tr(lang, 'helper_days', { max: MAX_DAYS }),
    height: tr(lang, 'helper_height'),
  })[name];

function createEditorClass() {
  return class RezeptplanerCardEditor extends HTMLElement {
    setConfig(config) {
      this._config = { ...config };
      this._update();
    }

    set hass(hass) {
      this._hass = hass;
      this._update();
    }

    _update() {
      if (!this._form) {
        this._form = document.createElement('ha-form');
        this._form.schema = EDITOR_SCHEMA;
        this._form.computeLabel = (field) => editorLabel(languageOf(this._hass), field.name);
        this._form.computeHelper = (field) => editorHelper(languageOf(this._hass), field.name);
        this._form.addEventListener('value-changed', (ev) => this._changed(ev.detail.value));
        this.append(this._form);
      }
      this._form.hass = this._hass;
      this._form.data = this._config ?? {};
    }

    /** Leere Felder fallen aus der Konfiguration, dann gilt der Standard. */
    _changed(value) {
      const config = { ...this._config };
      for (const key of ['title', 'days', 'height']) {
        const v = value[key];
        if (v == null || v === '') delete config[key];
        else config[key] = v;
      }
      this._config = config;
      this.dispatchEvent(new CustomEvent('config-changed', { detail: { config }, bubbles: true, composed: true }));
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
    if (!window.customElements.get('rezeptplaner-card-editor')) {
      window.customElements.define('rezeptplaner-card-editor', createEditorClass());
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
    // Beim Laden gibt es noch kein hass; die Seite trägt die Sprache im lang-Attribut.
    name: tr(languageOf(null), 'card_name'),
    description: tr(languageOf(null), 'card_description'),
    preview: false,
  });
}
