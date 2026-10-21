// Hilfsfunktionen für die Anzeige.

/** Minuten als Text: 45 -> "45 Min.", 75 -> "1 Std. 15 Min.". */
export function formatMinutes(minutes) {
  if (minutes == null) return '';
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) return `${rest} Min.`;
  if (rest === 0) return `${hours} Std.`;
  return `${hours} Std. ${rest} Min.`;
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

/** Zahl mit höchstens digits Nachkommastellen und deutschem Komma. */
function formatDecimal(value, digits) {
  return value.toLocaleString('de-DE', { maximumFractionDigits: digits });
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
  return `${value.toLocaleString('de-DE', { maximumFractionDigits: digits })} ${units[unit]}`;
}

/** Dauer in Sekunden lesbar: 93784 -> "1 Tag 2 Std.", 300 -> "5 Min.". */
export function formatUptime(seconds) {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days} ${days === 1 ? 'Tag' : 'Tage'} ${hours} Std.`;
  if (hours > 0) return `${hours} Std. ${minutes} Min.`;
  return `${minutes} Min.`;
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
    if (days <= 0) return 'heute eingeplant';
    if (days === 1) return 'morgen eingeplant';
    if (days < 14) return `eingeplant in ${days} Tagen`;
    return `eingeplant in ${Math.round(days / 7)} Wochen`;
  }
  if (recipe.last_planned) {
    const days = daysBetween(recipe.last_planned, today);
    if (days <= 1) return 'zuletzt gestern';
    if (days < 14) return `zuletzt vor ${days} Tagen`;
    if (days < 60) return `zuletzt vor ${Math.round(days / 7)} Wochen`;
    if (days < 365) return `zuletzt vor ${Math.round(days / 30)} Monaten`;
    const years = Math.floor(days / 365);
    return years === 1 ? 'zuletzt vor über einem Jahr' : `zuletzt vor über ${years} Jahren`;
  }
  return 'noch nie eingeplant';
}
