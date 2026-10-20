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
