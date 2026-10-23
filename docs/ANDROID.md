# Android-App: Entwicklung, Signatur und Installation

Die App liegt in `android/` (Kotlin, Jetpack Compose). Planung: [PLANUNG.md](PLANUNG.md), Abschnitte 7 und 10.

Befehle sind für **Git Bash** auf Windows geschrieben.

## 1. Android Studio installieren

1. Herunterladen von <https://developer.android.com/studio> und mit den Standardeinstellungen installieren.
2. Beim ersten Start den Einrichtungsassistenten mit **Standard** durchlaufen. Er lädt:
   - **Android SDK** samt **Platform-Tools** (`adb`)
   - die aktuelle **SDK-Plattform** und **Build-Tools**
   - den **Android Emulator** (optional, braucht viel Platz; zum Testen reicht das Handy)
   - Alle Lizenzen bestätigen.
3. Danach **More Actions → SDK Manager** (im Projekt: *Tools → SDK Manager*):
   - Reiter **SDK Platforms**: **Android 16.0 („Baklava“), API 36** anhaken.
   - Reiter **SDK Tools**: **Android SDK Build-Tools** (neueste), **Android SDK Platform-Tools**, **Android SDK Command-line Tools (latest)** anhaken.
   - **Google USB Driver** ist nur für Pixel-Geräte – für Samsung siehe Abschnitt 2.
4. Projekt öffnen: **Open** → Ordner `C:\Projekte\Rezepteverwaltung_Essenplan\android` (nicht das ganze Repository). Der erste Gradle-Abgleich lädt einige hundert MB.

Ergebnis (Standardpfade):

| Was | Pfad |
|---|---|
| SDK | `C:\Users\<Name>\AppData\Local\Android\Sdk` |
| Java (mitgeliefert) | `C:\Program Files\Android\Android Studio\jbr` |
| adb | `…\Android\Sdk\platform-tools\adb.exe` |

Bauen ohne Android Studio (Git Bash):

```bash
export JAVA_HOME="/c/Program Files/Android/Android Studio/jbr"
cd /c/Projekte/Rezepteverwaltung_Essenplan/android
./gradlew assembleDebug     # APK: app/build/outputs/apk/debug/app-debug.apk
./gradlew installDebug      # auf das per USB verbundene Handy installieren
```

Oberflächentests ohne Gerät (Robolectric) mit Bildschirmfotos unter `app/build/outputs/roborazzi/`:

```bash
./gradlew recordRoborazziDebug
```

**Release-Build vor dem Veröffentlichen auf einem Gerät oder Emulator prüfen** – am einfachsten mit der Variante `qa` (`./gradlew assembleQa`): wie Release mit R8, aber mit Debug-Schlüssel signiert und mit `http` zum Testserver (`http://10.0.2.2:<Port>` aus dem Emulator; Debug und QA erlauben das, Release nur `https`). Nur er wird von R8 verkleinert; Fehler durch fehlende R8-Regeln (z. B. beim QR-Scanner in 0.2.0/0.2.1) zeigen weder Debug-Build noch Unit-Tests. Eigene Regeln stehen in `app/proguard-rules.pro`. Einen verschleierten Stacktrace übersetzt `retrace` mit `app/build/outputs/mapping/release/mapping.txt`.

## 2. Galaxy A54 per USB verbinden

1. **Samsung-USB-Treiber** für Windows installieren: <https://developer.samsung.com/android-usb-driver> (der Google-Treiber passt nicht).
2. **Entwickleroptionen freischalten:** Einstellungen → **Telefoninfo** → **Softwareinformationen** → **7 × auf „Buildnummer“ tippen** → PIN eingeben. Meldung: „Entwicklermodus aktiviert“.
3. **USB-Debugging einschalten:** Einstellungen → **Entwickleroptionen** (ganz unten) → **USB-Debugging** an.
4. **Automatische Sperre (Auto Blocker) ausschalten:** Einstellungen → **Sicherheit und Datenschutz** → **Automatische Sperre** → aus. Sonst blockiert One UI Befehle über USB und die Installation von Apps außerhalb des Play Store (auch über Obtainium).
5. Handy mit einem **Datenkabel** an den PC anschließen (reine Ladekabel funktionieren nicht). Auf dem Handy erscheint **„USB-Debugging zulassen?“** mit dem Fingerabdruck des PCs → **„Von diesem Computer immer zulassen“** → Zulassen.
6. Prüfen:

   ```bash
   "$LOCALAPPDATA/Android/Sdk/platform-tools/adb.exe" devices
   # erwartet: R5CW…   device   (bei "unauthorized": Abfrage auf dem Handy bestätigen)
   ```

In Android Studio erscheint das Handy oben in der Geräteauswahl; **Run ▶** installiert und startet die Debug-Version („Rezeptplaner“ mit Endung `.debug`, läuft neben der Release-Version).

Alternative ohne Kabel: Entwickleroptionen → **Wireless-Debugging** (gleiches WLAN), in Android Studio *Pair Devices Using Wi-Fi*.

## 3. Signaturschlüssel anlegen (einmalig)

Android installiert ein Update nur, wenn es mit **demselben Schlüssel** signiert ist wie die installierte App. Der Schlüssel entsteht einmal auf deinem PC und wird als geheimer Wert bei GitHub hinterlegt; die GitHub Action signiert damit jede Release-APK.

**Selbst in Git Bash ausführen** (keytool fragt nach Passwörtern; nicht über Claude):

```bash
cd ~
"/c/Program Files/Android/Android Studio/jbr/bin/keytool" -genkeypair -v \
  -keystore rezeptplaner-release.jks -storetype PKCS12 \
  -alias rezeptplaner -keyalg RSA -keysize 4096 -validity 36500 \
  -dname "CN=Rezeptplaner"
```

- Passwort zweimal eingeben (ein langes, zufälliges; z. B. aus dem Passwortmanager). Bei PKCS12 gilt es für Speicher **und** Schlüssel.
- Ergebnis: `C:\Users\<Name>\rezeptplaner-release.jks`.

Fingerabdruck notieren (zum späteren Abgleich):

```bash
"/c/Program Files/Android/Android Studio/jbr/bin/keytool" -list -v -keystore ~/rezeptplaner-release.jks | grep SHA256
```

### Als geheime Werte bei GitHub hinterlegen

1. Datei base64-kodiert in die Zwischenablage kopieren:

   ```bash
   base64 -w0 ~/rezeptplaner-release.jks | clip
   ```

2. GitHub → Repository → **Settings → Secrets and variables → Actions → New repository secret**, vier Einträge:

   | Name | Wert |
   |---|---|
   | `ANDROID_KEYSTORE_BASE64` | Inhalt der Zwischenablage (Strg+V) |
   | `ANDROID_KEYSTORE_PASSWORD` | das Passwort |
   | `ANDROID_KEY_ALIAS` | `rezeptplaner` |
   | `ANDROID_KEY_PASSWORD` | dasselbe Passwort |

GitHub zeigt geheime Werte nie wieder an und gibt sie nicht an Pull Requests aus Forks weiter.

### ⚠ Schlüssel sichern

**Ohne diese Datei und das Passwort lassen sich keine Updates mehr installieren** – die App müsste deinstalliert (lokale Daten weg) und neu eingerichtet werden, auf jedem Gerät.

- `rezeptplaner-release.jks` **und** Passwort an zwei Orten sichern, z. B. im Passwortmanager (Datei als Anhang) **und** auf einem USB-Stick bzw. im verschlüsselten Backup.
- **Nie** ins Repository legen (`*.jks` steht in `.gitignore`), nicht per Mail oder Chat verschicken.
- Lokal signieren ist optional: Datei `android/keystore.properties` (wird nicht eingecheckt) mit `storeFile=C:/Users/<Name>/rezeptplaner-release.jks`, `storePassword=…`, `keyAlias=rezeptplaner`, `keyPassword=…`.

## 4. Release veröffentlichen

1. `appVersionName` in `android/gradle.properties` erhöhen (z. B. `0.2.0`).
2. Committen, Tag `android-v0.2.0` setzen und pushen.
3. Die Action **Android** baut die signierte APK, prüft die Signatur und legt ein GitHub-Release „Android-App 0.2.0“ mit `rezeptplaner-0.2.0.apk` an.

Tags `v…` (ohne `android-`) bleiben der Home-Assistant-App vorbehalten.

## 5. Installation und Updates mit Obtainium

Am einfachsten über die Weboberfläche: **Einstellungen → App installieren → Schritt 1**. Dort stehen QR-Codes und die Anleitung für das A54. Die Links baut der Server (`GET /api/v1/app-install`, `server/internal/appinstall`) aus der Option **app_repository** der Home-Assistant-App (Standard: dieses Repository).

| QR-Code | Ziel |
|---|---|
| Obtainium herunterladen | `https://github.com/ImranR98/Obtainium/releases/latest/download/app-arm64-v8a-release.apk` (64-Bit-ARM, ohne `fdroid`; passt für das A54) |
| Rezeptplaner in Obtainium eintragen | `https://apps.obtainium.imranr.dev/redirect?r=obtainium://app/<Konfiguration>` – öffnet Obtainium mit fertiger Konfiguration (Repository; Version aus dem Tag `android-v1.2.3` per `versionExtractionRegEx`), dort bestätigen und **Installieren** |
| APK direkt (ohne Obtainium) | `…/releases/latest/download/rezeptplaner.apk` – immer die neueste APK; die GitHub Action legt sie in jedem Release unter diesem Namen ab |

Filter für Release-Titel oder APK braucht Obtainium nicht: Releases gibt es nur für die Android-App (Tags `v…` der Home-Assistant-App bekommen keine), und jedes enthält genau eine APK. **Dabei bleiben:** keine Releases für die Home-Assistant-App anlegen, sonst zeigt `releases/latest` nicht mehr auf die App.

Auf dem Galaxy A54 (Android 16):

1. **Automatische Sperre** aus (Abschnitt 2, Schritt 4) – sonst blockiert One UI die Installation und auch spätere Updates über Obtainium.
2. **„Unbekannte Apps installieren“** erlauben: Android fragt beim ersten Mal für den Browser bzw. für Obtainium.
3. **Play Protect:** Bei „Unbekannte App“ → *Weitere Details* → *Trotzdem installieren*. Das Senden zur Prüfung kann man ablehnen.
4. Updates: Obtainium prüft regelmäßig und meldet neue Versionen; „Aktualisieren“ installiert sie. Akku-Optimierung für Obtainium ggf. ausschalten, damit die Prüfung im Hintergrund läuft.

## 6. Googles Entwickler-Verifizierung (Stand 2026-10-23)

Google verlangt schrittweise, dass auf zertifizierten Android-Geräten nur noch Apps **registrierter, verifizierter Entwickler** installiert werden – auch außerhalb des Play Store.

- **Seit 30.09.2026** nur in Brasilien, Indonesien, Singapur und Thailand und nur für Installationen aus „teilnehmenden Stores“. Direkt geladene APKs (Browser, Obtainium) sind laut FAQ noch nicht betroffen.
- **2027:** weltweite Ausweitung auf alle Apps angekündigt, ohne genaues Datum für Deutschland/EU.
- **Ausnahmen:** Installation per **ADB** bleibt ohne Verifizierung möglich; für Power-User gibt es den **„erweiterten Ablauf“** (einmalig: Entwicklermodus, Neustart, 24 Stunden Wartezeit, dann Bestätigung; danach lassen sich unregistrierte Apps für 7 Tage oder dauerhaft installieren).
- **Für uns vorgesehen:** ein kostenloses **Konto für eingeschränkten Vertrieb** (Studierende/Hobby, ohne Ausweis, ohne Gebühr, bis 20 Geräte) in der Android Developer Console, dort den Paketnamen `io.github.thomas1607haja.rezeptplaner` mit dem Signaturzertifikat registrieren. Rechtzeitig vor der Ausweitung 2027 erledigen; an App und Action ändert sich dafür nichts. **Der Signaturschlüssel ist dafür Voraussetzung** – ohne ihn lässt sich das Paket nicht registrieren.

Quellen: [Android Developers Blog](https://developer.android.com/blog/posts/android-developer-verification-rolling-out-to-all-developers-on-play-console-and-android-developer-console), [Developer Verification](https://developer.android.com/developer-verification), [FAQ](https://developer.android.com/developer-verification/guides/faq).
