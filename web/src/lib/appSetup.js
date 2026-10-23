// Inhalt des QR-Codes, mit dem die Android-App eingerichtet wird.
//
// Format: rezeptplaner://setup?url=…&key=…[&cf_id=…&cf_secret=…]
//   url        Adresse des App-Zugangs (Einstellung app_url)
//   key        API-Schlüssel des Geräts
//   cf_id,     optional: Service Token von Cloudflare Access; wird nirgends
//   cf_secret  gespeichert, nur in den QR-Code übernommen
// Die App prüft das Schema "rezeptplaner" und den Pfad "setup".

/** Baut den Text des QR-Codes; leere Cloudflare-Angaben fallen weg. */
export function setupUri({ url, key, cfId = '', cfSecret = '' }) {
  const params = new URLSearchParams({ url, key });
  if (cfId.trim() && cfSecret.trim()) {
    params.set('cf_id', cfId.trim());
    params.set('cf_secret', cfSecret.trim());
  }
  return `rezeptplaner://setup?${params}`;
}
