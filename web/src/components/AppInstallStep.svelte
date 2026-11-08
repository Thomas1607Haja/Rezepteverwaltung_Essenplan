<script>
  import { api } from '../lib/api.js';
  import { t, errorText } from '../lib/i18n.svelte.js';
  import QrCode from './QrCode.svelte';

  // Schritt 1 der Einrichtung eines Handys: App installieren. Die Links
  // baut der Server aus der Option app_repository der Home-Assistant-App
  // (GET /api/v1/app-install), damit hier keine Adresse fest im Code steht.
  let links = $state(null);
  let error = $state('');
  api
    .appInstall()
    .then((l) => (links = l))
    .catch((e) => (error = t('devices.install.load_failed', { message: errorText(e) })));

  /**
   * Zerlegt einen übersetzten Text mit **fett** und *kursiv* in Stücke,
   * damit die Hervorhebungen ohne {@html} erhalten bleiben.
   */
  function rich(text) {
    return text
      .split(/(\*\*[^*]+\*\*|\*[^*]+\*)/)
      .filter(Boolean)
      .map((part) =>
        part.startsWith('**')
          ? { bold: part.slice(2, -2) }
          : part.startsWith('*') && part.length > 2
            ? { em: part.slice(1, -1) }
            : { text: part },
      );
  }
</script>

{#snippet richText(text)}{#each rich(text) as part, i (i)}{#if part.bold}<strong>{part.bold}</strong>{:else if part.em}<em
      >{part.em}</em
    >{:else}{part.text}{/if}{/each}{/snippet}

<h3>{t('devices.install.step1')}</h3>

{#if error}
  <p class="error">{error}</p>
{:else if !links}
  <p>{t('common.loading')}</p>
{:else}
  <p class="hint">{@render richText(t('devices.install.recommended'))}</p>
  <div class="codes">
    <figure>
      <a href={links.obtainium_download_url} target="_blank" rel="noopener noreferrer">
        <QrCode text={links.obtainium_download_url} size={150} label={t('devices.install.qr_obtainium')} />
      </a>
      <figcaption>
        <strong>a)</strong>
        {t('devices.install.obtainium_download')}<br /><small
          >{t('devices.install.loads_directly')} <code>app-arm64-v8a-release.apk</code></small
        >
      </figcaption>
    </figure>
    <figure>
      <a href={links.obtainium_url} target="_blank" rel="noopener noreferrer">
        <QrCode text={links.obtainium_url} size={200} label={t('devices.install.qr_obtainium_add')} />
      </a>
      <figcaption>
        <strong>b)</strong>
        {t('devices.install.obtainium_add')}<br /><small>{t('devices.install.obtainium_add_hint')}</small>
      </figcaption>
    </figure>
  </div>

  <details>
    <summary>{t('devices.install.apk_summary')}</summary>
    <div class="codes">
      <figure>
        <a href={links.apk_url}>
          <QrCode text={links.apk_url} size={150} label={t('devices.install.qr_apk')} />
        </a>
        <figcaption>{t('devices.install.apk_latest')}<br /><small>{t('devices.install.apk_updates')}</small></figcaption>
      </figure>
    </div>
  </details>

  <details>
    <summary>{t('devices.install.guide_summary')}</summary>
    <ol class="steps">
      <li>{@render richText(t('devices.install.guide_auto_blocker'))}</li>
      <li>{@render richText(t('devices.install.guide_unknown_apps'))}</li>
      <li>{@render richText(t('devices.install.guide_play_protect'))}</li>
      <li>{t('devices.install.guide_open')}</li>
    </ol>
  </details>
  <p class="hint">
    {t('devices.install.source')}
    <a href={links.releases_url} target="_blank" rel="noopener noreferrer">{links.repository}</a>
    ({t('devices.install.releases')})
  </p>

  <h3>{t('devices.install.skill_title')}</h3>
  <p class="hint">{t('devices.install.skill_text')}</p>
  <p><a class="download" href={links.skill_url}>{t('devices.install.skill_download')}</a></p>
  <details>
    <summary>{t('devices.install.skill_summary')}</summary>
    <ol class="steps">
      <li>{@render richText(t('devices.install.skill_step1'))}</li>
      <li>{@render richText(t('devices.install.skill_step2'))}</li>
      <li>{t('devices.install.skill_step3')}</li>
    </ol>
    <p class="hint">{t('devices.install.skill_free')}</p>
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
