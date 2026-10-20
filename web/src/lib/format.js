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
