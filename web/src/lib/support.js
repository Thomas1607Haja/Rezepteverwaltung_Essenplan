// Adresse der Seite „Unterstützen“ (Buy Me a Coffee). Sie steht nur in
// support-url.txt im Hauptordner des Repositorys; Vite liest die Datei beim
// Bauen ein (?raw = Inhalt als Text). Die Android-App liest dieselbe Datei.
import raw from '../../../support-url.txt?raw';

export const supportUrl = raw.trim();
