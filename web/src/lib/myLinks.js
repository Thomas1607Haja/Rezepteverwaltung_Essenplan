// My-Home-Assistant-Links (App installieren, Integration herunterladen und
// einrichten). Sie stehen nur in my-links.properties im Hauptordner; Vite liest
// die Datei beim Bauen ein, die Android-App liest dieselbe Datei.
import raw from '../../../my-links.properties?raw';

function parse(text) {
  const links = {};
  for (const line of text.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const i = trimmed.indexOf('=');
    if (i > 0) links[trimmed.slice(0, i).trim()] = trimmed.slice(i + 1).trim();
  }
  return links;
}

export const myLinks = parse(raw);
