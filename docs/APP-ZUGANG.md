# App-Zugang über einen Cloudflare-Tunnel

Die Android-App spricht den Rezeptplaner nicht über Home Assistant an, sondern über einen eigenen Zugang: **Port 8100** der App. Dort verlangt der Server bei **jeder** Anfrage einen gültigen API-Schlüssel (`Authorization: Bearer rp_…`), sonst antwortet er mit 401. Die Weboberfläche und die Schlüsselverwaltung gibt es dort nicht.

Die Weboberfläche (Port 8099, Ingress) bleibt davon getrennt und ist nur über Home Assistant erreichbar.

In den Beispielen steht `rezepte.example.org` für deinen Hostnamen. Echte Domains gehören nicht in dieses Repository.

## 1. API-Schlüssel anlegen

1. Weboberfläche öffnen → **Einstellungen → App-Zugang**.
2. Gerätenamen eingeben (z. B. „Handy Thomas“) → **Schlüssel anlegen**.
3. Den Schlüssel sofort übernehmen. Er wird nur dieses eine Mal angezeigt, gespeichert wird nur sein Hash.

Ein verlorenes oder ausgemustertes Gerät sperrst du dort mit **Sperren**. Der Schlüssel gilt sofort nicht mehr.

## 2. Hostnamen im Tunnel einrichten

Port 8100 ist auf dem Green nach außen **nicht** freigegeben. Das ist auch nicht nötig: Die Cloudflared-App läuft im selben internen Netz des Supervisors und erreicht den Rezeptplaner über dessen Hostnamen.

1. Hostnamen der App nachsehen: **Einstellungen → Apps → Rezeptplaner**, im Abschnitt mit den Angaben zur App steht der **Hostname**, z. B. `a1b2c3d4-rezeptplaner`.
2. In der Konfiguration der Cloudflared-App einen weiteren Hostnamen eintragen (je nach App unter `additional_hosts`):

   ```yaml
   additional_hosts:
     - hostname: rezepte.example.org
       service: http://a1b2c3d4-rezeptplaner:8100
   ```

   Bei einem Tunnel, der im Cloudflare-Dashboard verwaltet wird: **Zero Trust → Networks → Tunnels → Tunnel → Public Hostname → Add**, Dienst `HTTP`, URL `a1b2c3d4-rezeptplaner:8100`.
3. Cloudflared-App neu starten.

Prüfen (Git Bash):

```bash
# ohne Schlüssel: 401
curl -s -o /dev/null -w "%{http_code}\n" https://rezepte.example.org/api/v1/health
# mit Schlüssel: 200
curl -s -H "Authorization: Bearer rp_DEIN_SCHLUESSEL" https://rezepte.example.org/api/v1/health
```

## 3. Cache-Regel

Wie beim Home-Assistant-Hostnamen soll Cloudflare hier nichts zwischenspeichern: **Caching → Cache Rules**, Bedingung *Hostname equals* `rezepte.example.org`, Aktion **Bypass cache**. (Der Server schickt für Bilder ohnehin `Cache-Control: private`.)

## 4. Zusätzlicher Schutz mit Cloudflare Access (empfohlen)

Mit Cloudflare Access lässt Cloudflare nur Anfragen mit einem gültigen **Service Token** überhaupt bis zum Tunnel durch. Ohne Token antwortet schon Cloudflare mit 403, der Rezeptplaner sieht die Anfrage gar nicht. Ein gestohlener API-Schlüssel allein reicht dann nicht mehr.

1. **Zero Trust → Access → Service Auth → Service Tokens → Create Service Token**, Name z. B. „Rezeptplaner-App“, Laufzeit nach Wunsch. **Client ID** und **Client Secret** notieren (das Secret wird nur einmal angezeigt).
2. **Zero Trust → Access → Applications → Add an application → Self-hosted**:
   - Name „Rezeptplaner-App“, Domain `rezepte.example.org`
   - Policy: Aktion **Service Auth**, Include → **Service Token** → den Token aus Schritt 1
3. Speichern.

Prüfen:

```bash
# ohne Service Token: 403 von Cloudflare
curl -s -o /dev/null -w "%{http_code}\n" https://rezepte.example.org/api/v1/health
# mit Service Token und API-Schlüssel: 200
curl -s https://rezepte.example.org/api/v1/health \
  -H "CF-Access-Client-Id: DEINE_CLIENT_ID.access" \
  -H "CF-Access-Client-Secret: DEIN_CLIENT_SECRET" \
  -H "Authorization: Bearer rp_DEIN_SCHLUESSEL"
```

Die Android-App kann **CF-Access-Client-Id** und **CF-Access-Client-Secret** optional mitschicken (Einstellung in der App, siehe Planung Phase 7). Ohne diese Angaben funktioniert sie nur, solange Access für den Hostnamen nicht aktiv ist.
