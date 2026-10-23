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

1. Obtainium installieren: <https://github.com/ImranR98/Obtainium/releases> → `app-arm64-v8a-release.apk` auf dem Handy herunterladen und öffnen. Android fragt einmalig, ob der Browser Apps installieren darf → erlauben.
2. In Obtainium **App hinzufügen**: URL `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
   - **Nur Releases mit diesem Tag-Muster** (Filter für Release-Titel/Tags): `^android-v`
   - **APK-Filter**: `rezeptplaner-.*\.apk`
3. **Installieren** – Obtainium fragt einmalig nach der Berechtigung „Unbekannte Apps installieren“ → erlauben.
4. Updates: Obtainium prüft regelmäßig und meldet neue Versionen; „Aktualisieren“ installiert sie. Akku-Optimierung für Obtainium ggf. ausschalten, damit die Prüfung im Hintergrund läuft.

Die Automatische Sperre (Abschnitt 2, Schritt 4) muss dafür aus sein.
