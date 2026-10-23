<script>
  import { api } from '../lib/api.js';
  import QrCode from './QrCode.svelte';

  // Schritt 1 der Einrichtung eines Handys: App installieren. Die Links
  // baut der Server aus der Option app_repository der Home-Assistant-App
  // (GET /api/v1/app-install), damit hier keine Adresse fest im Code steht.
  let links = $state(null);
  let error = $state('');
  api
    .appInstall()
    .then((l) => (links = l))
    .catch((e) => (error = `Links nicht geladen: ${e.message}`));
</script>

<h3>Schritt 1 – App installieren</h3>

{#if error}
  <p class="error">{error}</p>
{:else if !links}
  <p>Lädt …</p>
{:else}
  <p class="hint">
    Empfohlen über <strong>Obtainium</strong>: Es installiert den Rezeptplaner und meldet später jedes Update. Die
    Codes mit der Kamera des Handys scannen.
  </p>
  <div class="codes">
    <figure>
      <a href={links.obtainium_download_url} target="_blank" rel="noopener noreferrer">
        <QrCode text={links.obtainium_download_url} size={150} label="QR-Code: Obtainium herunterladen" />
      </a>
      <figcaption>
        <strong>a)</strong> Obtainium herunterladen<br /><small>lädt direkt <code>app-arm64-v8a-release.apk</code></small>
      </figcaption>
    </figure>
    <figure>
      <a href={links.obtainium_url} target="_blank" rel="noopener noreferrer">
        <QrCode text={links.obtainium_url} size={200} label="QR-Code: Rezeptplaner in Obtainium eintragen" />
      </a>
      <figcaption>
        <strong>b)</strong> Rezeptplaner in Obtainium eintragen<br /><small>dann dort „Installieren“</small>
      </figcaption>
    </figure>
  </div>

  <details>
    <summary>Ohne Obtainium: APK direkt herunterladen</summary>
    <div class="codes">
      <figure>
        <a href={links.apk_url}>
          <QrCode text={links.apk_url} size={150} label="QR-Code: neueste APK herunterladen" />
        </a>
        <figcaption>Immer die neueste Version.<br /><small>Updates dann von Hand.</small></figcaption>
      </figure>
    </div>
  </details>

  <details>
    <summary>Anleitung für das Galaxy A54 (Android 16)</summary>
    <ol class="steps">
      <li>
        <strong>Automatische Sperre ausschalten:</strong> Einstellungen → Sicherheit und Datenschutz → Automatische Sperre →
        Aus. Sie blockiert Apps, die nicht aus dem Play Store oder Galaxy Store kommen – auch Updates über Obtainium.
      </li>
      <li>
        <strong>„Unbekannte Apps installieren“ erlauben:</strong> Beim ersten Öffnen der heruntergeladenen Datei fragt
        Android, ob der Browser (bzw. später Obtainium) Apps installieren darf → <em>Einstellungen</em> → Schalter an →
        zurück.
      </li>
      <li>
        <strong>Play-Protect-Hinweis bestätigen:</strong> Meldet Google Play Protect eine „unbekannte App“, auf
        <em>Weitere Details</em> → <em>Trotzdem installieren</em> tippen. Die Frage, ob die App zur Prüfung gesendet
        werden soll, kann man ablehnen.
      </li>
      <li>Rezeptplaner öffnen und mit Schritt 2 verbinden.</li>
    </ol>
  </details>
  <p class="hint">
    Quelle: <a href={links.releases_url} target="_blank" rel="noopener noreferrer">{links.repository}</a> (GitHub-Releases)
  </p>

  <h3>Claude-Skill (optional)</h3>
  <p class="hint">
    Mit der Skill „rezeptplaner-import“ erfasst Claude Rezepte aus Fotos (Rezeptkarten, Kochbuchseiten) und erzeugt eine
    Importdatei, die du per Teilen an die App schickst oder unter Rezepte → Datei importieren hochlädst.
  </p>
  <p><a class="download" href={links.skill_url}>rezeptplaner-import-skill.zip herunterladen</a></p>
  <details>
    <summary>So kommt die Skill in Claude</summary>
    <ol class="steps">
      <li>
        In Claude (claude.ai oder App) unter <em>Einstellungen → Funktionen</em> „Code execution and file creation“
        (Codeausführung und Dateierstellung) einschalten.
      </li>
      <li><em>Customize → Skills</em> → Skill hochladen → die heruntergeladene ZIP-Datei wählen (nicht entpacken).</li>
      <li>
        Fotos der Rezepte in einen Chat schicken, z. B. „Erfasse diese Rezepte für den Rezeptplaner“. Claude liefert
        eine ZIP-Datei zum Teilen.
      </li>
    </ol>
    <p class="hint">
      Funktioniert auch mit der kostenlosen Claude-Version; größere Stapel von Fotos stoßen dort aber schneller an die
      Nutzungsgrenzen. Bei einer neuen Version der Skill die ZIP erneut herunterladen und in Claude ersetzen.
    </p>
  </details>
{/if}

<style>
  h3 {
    margin: 0 0 0.5rem;
    font-size: 1rem;
  }

  p {
    margin: 0 0 0.75rem;
  }

  .hint,
  small {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .codes {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    margin-bottom: 0.75rem;
  }

  figure {
    margin: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.35rem;
    max-width: 15rem;
  }

  figcaption {
    font-size: 0.9rem;
  }

  details {
    margin-bottom: 0.75rem;
  }

  summary {
    cursor: pointer;
    color: var(--accent);
  }

  details[open] summary {
    margin-bottom: 0.5rem;
  }

  .steps {
    margin: 0;
    padding-left: 1.3rem;
    font-size: 0.9rem;
  }

  .steps li {
    margin-bottom: 0.4rem;
  }

  .download {
    font-weight: 600;
  }

  .error {
    color: var(--danger);
  }
</style>
