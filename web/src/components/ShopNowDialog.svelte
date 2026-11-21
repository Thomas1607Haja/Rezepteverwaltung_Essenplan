<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { t, errorText } from '../lib/i18n.svelte.js';
  import PlanPreview from './PlanPreview.svelte';

  // „Jetzt einkaufen“: die Zutaten eines Rezepts spontan an Bring senden,
  // ohne es einzuplanen. Erst die Kurzvorschau (Portionen, Zutaten
  // abwählen), danach die Bestätigung mit „Auch im Essensplan eintragen“.
  //   recipe      das Rezept der Kochansicht
  //   onplan({ servings })  „Auch im Essensplan eintragen“ gewählt
  //   onclose()
  let { recipe, onplan, onclose } = $props();

  let list = $state(null); // { entity_id, name }
  let busy = $state(false);
  let message = $state('');
  let result = $state(null); // Antwort des Servers nach dem Senden
  let servings = $state(null); // gesendete Portionen (für „Auch eintragen“)

  api
    .shoppingList()
    .then((l) => (list = l))
    .catch((e) => (message = errorText(e)));

  const noList = $derived(list != null && !list.entity_id);
  const label = $derived(
    busy
      ? t('recipe.shop_now.sending')
      : list?.entity_id
        ? t('recipe.shop_now.send', { list: list.name })
        : t('recipe.shop_now.send_plain'),
  );

  async function send(choice) {
    busy = true;
    message = '';
    try {
      result = await api.sendRecipe(recipe.id, choice.servings, choice.excluded);
      servings = choice.servings;
    } catch (e) {
      message = errorText(e);
    } finally {
      busy = false;
    }
  }

  let dialog = $state();
  // Der Ergebnisdialog öffnet sich, sobald er im DOM steht.
  $effect(() => {
    if (result && dialog && !dialog.open) dialog.showModal();
  });

  const merged = $derived(result?.sent.filter((s) => s.merged).length ?? 0);
  const listName = $derived(list?.name || result?.list || '');
</script>

{#if !result}
  <PlanPreview
    recipeId={recipe.id}
    shop
    {busy}
    confirmLabel={label}
    confirmDisabled={noList || list == null}
    message={noList ? t('recipe.shop_now.no_list') : message}
    onconfirm={send}
    {onclose}
  />
{:else}
  <dialog
    bind:this={dialog}
    oncancel={onclose}
    onclick={(e) => e.target === dialog && onclose()}
    aria-labelledby="shop-now-title"
  >
    <div class="content">
      <h2 id="shop-now-title">{t('recipe.shop_now.title')}</h2>
      <p class="done">{t('recipe.shop_now.sent', { count: result.sent.length, list: listName })}</p>
      {#if merged > 0}<p class="hint">{t('recipe.shop_now.merged', { count: merged })}</p>{/if}
      {#if result.failed.length > 0}
        <p class="error">
          {t('recipe.shop_now.failed', { items: result.failed.map((f) => f.name).join(', ') })}
        </p>
      {/if}
      <p class="also">
        <button type="button" class="link" onclick={() => onplan({ servings })}>{t('recipe.shop_now.also_plan')}</button>
        <small>{t('recipe.shop_now.also_plan_hint')}</small>
      </p>
      <footer>
        <button type="button" class="primary" onclick={onclose}>{t('common.close')}</button>
      </footer>
    </div>
  </dialog>
{/if}

<style>
  dialog {
    width: min(30rem, calc(100vw - 2rem));
    padding: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
  }

  dialog::backdrop {
    background: rgb(0 0 0 / 0.4);
  }

  .content {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    padding: 1rem 1.2rem;
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  p {
    margin: 0;
  }

  .done {
    font-weight: 600;
  }

  .hint,
  small {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  .also {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.15rem;
  }

  .error {
    color: var(--danger);
  }

  .link {
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    font: inherit;
    text-decoration: underline;
    cursor: pointer;
  }

  footer {
    display: flex;
    justify-content: flex-end;
  }

  footer button {
    padding: 0.5rem 1rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--on-accent);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
  }
</style>
