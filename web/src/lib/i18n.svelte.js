// Übersetzungen der Weboberfläche (Mehrsprachigkeit, docs/PLANUNG.md).
//
// Die Texte liegen in src/locales/<bereich>/<sprache>.json, z. B.
// locales/plan/fr.json. Schlüssel beginnen mit dem Bereich:
// t('plan.title') sucht "title" in locales/plan/<sprache>.json.
// Englisch ist immer geladen und gilt, wo eine Übersetzung fehlt.
//
// Welche Sprache gilt (siehe GET /api/v1/language):
//   1. die Einstellung „Sprache“, wenn nicht „Automatisch“,
//   2. die Systemsprache von Home Assistant,
//   3. die Sprache des Browsers,
//   4. Englisch.
//
// Platzhalter: t('plan.moved', { title: 'Lasagne' }) ersetzt {title}.
// Pluralformen: Ist der Text ein Objekt { one: …, other: … }, wählt
// Intl.PluralRules die Form passend zu params.count.

export const LANGUAGES = ['en', 'de', 'fr', 'es', 'it'];

// Englisch sofort mitliefern (Rückfallsprache), die anderen Sprachen erst
// bei Bedarf laden: Vite legt jede Datei als eigenes kleines Stück ab.
const english = import.meta.glob('../locales/*/en.json', { eager: true, import: 'default' });
const others = import.meta.glob(['../locales/*/*.json', '!../locales/*/en.json'], { import: 'default' });

/** Fügt die Dateien einer Sprache zu { bereich: {…} } zusammen. */
function merge(files) {
  const dict = {};
  for (const [path, content] of Object.entries(files)) {
    const area = path.split('/').at(-2);
    dict[area] = content;
  }
  return dict;
}

const EN = merge(english);

// Übliche Region je Sprache für Datums- und Zahlenformate, wenn der Browser
// keine passende nennt (en → 10/24, de → 24.10., fr/es/it → 24/10).
const REGIONS = { en: 'en-US', de: 'de-DE', fr: 'fr-FR', es: 'es-ES', it: 'it-IT' };

const state = $state({ lang: 'en', locale: 'en-US', dict: EN, setting: 'auto', homeAssistant: '' });

/** Aktuelle Sprache ("de"). */
export const currentLanguage = () => state.lang;

/** Locale für Intl, z. B. "de-DE" oder – wenn der Browser es nennt – "en-GB". */
export const currentLocale = () => state.locale;

/** Einstellung der Sprache ("auto" oder Sprachcode) und Sprache von Home Assistant. */
export const languageInfo = () => ({ setting: state.setting, homeAssistant: state.homeAssistant });

function lookup(dict, key) {
  let value = dict;
  for (const part of key.split('.')) {
    if (value == null || typeof value !== 'object') return undefined;
    value = value[part];
  }
  return value;
}

/** Übersetzt key; fehlt der Text, gilt Englisch, sonst der Schlüssel selbst. */
export function t(key, params = {}) {
  let value = lookup(state.dict, key);
  if (value === undefined) value = lookup(EN, key);
  if (value === undefined) return key;
  if (typeof value === 'object') {
    const form = new Intl.PluralRules(state.locale).select(Number(params.count ?? 0));
    value = value[form] ?? value.other ?? '';
  }
  return value.replace(/\{(\w+)\}/g, (all, name) => {
    if (!(name in params)) return all;
    const v = params[name];
    return typeof v === 'number' && !Number.isInteger(v) ? formatNumber(v) : String(v);
  });
}

/** Gibt es zu key einen Text (in der aktuellen Sprache oder Englisch)? */
export const hasText = (key) => lookup(state.dict, key) !== undefined || lookup(EN, key) !== undefined;

/** Zahl in der Schreibweise der Sprache: 1.5 → "1,5" bzw. "1.5". */
export function formatNumber(value, maxDigits = 2) {
  return value.toLocaleString(state.locale, { maximumFractionDigits: maxDigits });
}

/** Wählt die Locale: passende Sprache des Browsers, sonst die übliche Region. */
function localeFor(lang) {
  const browser = (navigator.languages ?? [navigator.language]).find((l) => l?.toLowerCase().startsWith(lang));
  return browser ?? REGIONS[lang];
}

/** Erste unterstützte Sprache des Browsers oder "". */
function browserLanguage() {
  for (const tag of navigator.languages ?? [navigator.language]) {
    const base = tag?.slice(0, 2).toLowerCase();
    if (LANGUAGES.includes(base)) return base;
  }
  return '';
}

/** Wechselt die Sprache und lädt ihre Texte. */
async function use(lang) {
  let dict = EN;
  if (lang !== 'en') {
    const entries = Object.entries(others).filter(([path]) => path.endsWith(`/${lang}.json`));
    const loaded = await Promise.all(entries.map(async ([path, load]) => [path, await load()]));
    dict = merge(Object.fromEntries(loaded));
  }
  state.lang = lang;
  state.locale = localeFor(lang);
  state.dict = dict;
  document.documentElement.lang = lang;
}

/**
 * Bestimmt die Sprache aus Einstellung, Home Assistant und Browser und lädt
 * sie. Wird beim Start einmal aufgerufen (main.js), danach nach dem
 * Speichern der Einstellung. info: Antwort von GET /api/v1/language.
 */
export async function applyLanguage(info) {
  state.setting = info?.setting ?? 'auto';
  state.homeAssistant = info?.home_assistant ?? '';
  const lang =
    (state.setting !== 'auto' && LANGUAGES.includes(state.setting) && state.setting) ||
    (LANGUAGES.includes(state.homeAssistant) && state.homeAssistant) ||
    browserLanguage() ||
    'en';
  await use(lang);
}

/**
 * Text zu einem Fehler der API (ApiError): übersetzter Code, sonst der
 * englische Text des Servers. Andere Fehler: ihre Meldung.
 */
export function errorText(err) {
  if (err?.code && hasText(`errors.${err.code}`)) return t(`errors.${err.code}`, err.params ?? {});
  return err?.message ?? String(err);
}

/** Text zu einem Hinweis { code, params, message } (z. B. Import-Warnungen). */
export function noteText(note) {
  if (typeof note === 'string') return note;
  if (note?.code && hasText(`errors.${note.code}`)) return t(`errors.${note.code}`, note.params ?? {});
  return note?.message ?? '';
}
