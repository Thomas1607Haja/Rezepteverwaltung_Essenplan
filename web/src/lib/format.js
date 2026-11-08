// Hilfsfunktionen für die Anzeige. Texte und Zahlenformate kommen aus
// i18n.svelte.js und folgen der Sprache der Oberfläche.
import { currentLocale, formatNumber, t } from './i18n.svelte.js';

/** Minuten als Text: 45 -> "45 Min.", 75 -> "1 Std. 15 Min." (bzw. "1 h 15 min"). */
export function formatMinutes(minutes) {
  if (minutes == null) return '';
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) return t('format.minutes', { count: rest });
  if (rest === 0) return t('format.hours', { count: hours });
  return t('format.hours_minutes', { hours, minutes: rest });
}

// Brüche, die beim Kochen üblich sind, mit ihrem Zahlenwert.
const fractions = [
  [1 / 4, '¼'],
  [1 / 3, '⅓'],
  [1 / 2, '½'],
  [2 / 3, '⅔'],
  [3 / 4, '¾'],
];

/**
 * Menge so, wie man sie in der Küche liest:
 * 333.33 -> "333", 12.46 -> "12,5", 1.5 -> "1 ½", 0.33 -> "⅓", 2.2 -> "2,2".
 */
export function formatQuantity(value) {
  if (value == null) return '';
  if (value >= 100) return String(Math.round(value));
  if (value >= 10) return formatDecimal(value, 1);

  const whole = Math.floor(value);
  const part = value - whole;
  if (part < 0.04) return String(whole);
  if (part > 0.96) return String(whole + 1);
  for (const [fraction, symbol] of fractions) {
    if (Math.abs(part - fraction) < 0.04) {
      return whole === 0 ? symbol : `${whole} ${symbol}`;
    }
  }
  return formatDecimal(value, 2);
}

/** Zahl mit höchstens digits Nachkommastellen im Format der Sprache. */
function formatDecimal(value, digits) {
  return formatNumber(value, digits);
}

/** Größe in Bytes lesbar: 1536 -> "1,5 KB", 23_000_000 -> "21,9 MB". */
export function formatBytes(bytes) {
  if (bytes == null) return '–';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  const digits = unit === 0 || value >= 100 ? 0 : 1;
  return `${formatNumber(value, digits)} ${units[unit]}`;
}

/** Dauer in Sekunden lesbar: 93784 -> "1 Tag 2 Std.", 300 -> "5 Min.". */
export function formatUptime(seconds) {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return t('format.days_hours', { count: days, hours });
  if (hours > 0) return t('format.hours_minutes', { hours, minutes });
  return t('format.minutes', { count: minutes });
}

/** Heutiger Tag im Format "2026-10-21" (Zeitzone des Browsers). */
export function todayString() {
  const d = new Date();
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** Tage von a bis b, beide als "2026-10-21". Rechnet in UTC, damit die
 * Zeitumstellung keinen Tag verschluckt. */
export function daysBetween(a, b) {
  const utc = (s) => Date.UTC(Number(s.slice(0, 4)), Number(s.slice(5, 7)) - 1, Number(s.slice(8, 10)));
  return Math.round((utc(b) - utc(a)) / 86_400_000);
}

/**
 * Statushinweis eines Rezepts aus next_planned und last_planned:
 * „heute eingeplant“, „eingeplant in 5 Tagen“, „zuletzt vor 2 Wochen“,
 * „noch nie eingeplant“. Ein künftiger Termin hat Vorrang.
 */
export function planStatus(recipe, today = todayString()) {
  if (recipe.next_planned) {
    const days = daysBetween(today, recipe.next_planned);
    if (days <= 0) return t('format.planned_today');
    if (days === 1) return t('format.planned_tomorrow');
    if (days < 14) return t('format.planned_in_days', { count: days });
    return t('format.planned_in_weeks', { count: Math.round(days / 7) });
  }
  if (recipe.last_planned) {
    const days = daysBetween(recipe.last_planned, today);
    if (days <= 1) return t('format.last_yesterday');
    if (days < 14) return t('format.last_days', { count: days });
    if (days < 60) return t('format.last_weeks', { count: Math.round(days / 7) });
    if (days < 365) return t('format.last_months', { count: Math.round(days / 30) });
    return t('format.last_years', { count: Math.floor(days / 365) });
  }
  return t('format.never_planned');
}

/**
 * Menge mit Ergebnis-Einheit (Schritt 9.2): „4 Portionen“, „1 Portion“,
 * bei Backrezepten „1 Springform 26 cm“, „12 Stück“.
 */
export function amountText(n, unit = '') {
  if (!unit) return t('format.servings', { count: n });
  return `${n} ${unit}`;
}

/** Vorschläge für die Ergebnis-Einheit im Editor (in der Sprache der Oberfläche). */
export const yieldUnits = () => t('format.yield_units').split('|');

/** Tag date ("2026-10-21") plus n Tage, wieder als "2026-10-21". */
export function addDays(date, n) {
  const d = new Date(Date.UTC(Number(date.slice(0, 4)), Number(date.slice(5, 7)) - 1, Number(date.slice(8, 10)) + n));
  return d.toISOString().slice(0, 10);
}

/**
 * Kopf einer Tagesspalte in der Schreibweise der Sprache:
 * { weekday: "Mi", date: "21.10." } bzw. { weekday: "Wed", date: "10/21" }.
 */
export function dayLabel(date) {
  const d = new Date(Date.UTC(Number(date.slice(0, 4)), Number(date.slice(5, 7)) - 1, Number(date.slice(8, 10))));
  const locale = currentLocale();
  // Kurze Wochentage ohne Punkt („Mi.“ → „Mi“, „mer.“ → „mer“).
  const weekday = new Intl.DateTimeFormat(locale, { weekday: 'short', timeZone: 'UTC' }).format(d).replace(/\.$/, '');
  let day = new Intl.DateTimeFormat(locale, { day: '2-digit', month: '2-digit', timeZone: 'UTC' }).format(d);
  // Deutsch schreibt den Tag mit Punkt am Ende: „21.10.“
  if (locale.startsWith('de') && !day.endsWith('.')) day += '.';
  return { weekday, date: day };
}

/** Langes Datum, z. B. „Samstag, 24. Oktober“ bzw. „Saturday, October 24“. */
export function longDate(date) {
  const d = new Date(Date.UTC(Number(date.slice(0, 4)), Number(date.slice(5, 7)) - 1, Number(date.slice(8, 10))));
  return new Intl.DateTimeFormat(currentLocale(), { weekday: 'long', day: 'numeric', month: 'long', timeZone: 'UTC' }).format(d);
}

/** Uhrzeit "18:30" in der Schreibweise der Sprache ("6:30 PM" im amerikanischen Englisch). */
export function formatTime(hhmm) {
  if (!hhmm) return '';
  const [h, m] = hhmm.split(':').map(Number);
  return new Intl.DateTimeFormat(currentLocale(), { hour: 'numeric', minute: '2-digit', timeZone: 'UTC' }).format(new Date(Date.UTC(2000, 0, 1, h, m)));
}

/** Titel eines Menüs: vom Server (Titel oder Hauptgericht), sonst „Menü“. */
export const menuTitle = (menu) => menu?.display_title || t('menu.untitled');
