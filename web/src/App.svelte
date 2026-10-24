<script>
  import { currentPath, match } from './lib/router.svelte.js';
  import Icon from './components/Icon.svelte';
  import MealPlanPage from './pages/MealPlanPage.svelte';
  import RecipesPage from './pages/RecipesPage.svelte';
  import RecipePage from './pages/RecipePage.svelte';
  import ShoppingPage from './pages/ShoppingPage.svelte';
  import SettingsPage from './pages/SettingsPage.svelte';
  import StatisticsPage from './pages/StatisticsPage.svelte';
  import NotFoundPage from './pages/NotFoundPage.svelte';

  // Die Hauptbereiche (docs/PLANUNG.md, Abschnitt 5).
  const sections = [
    { path: '/', label: 'Essensplan', icon: 'calendar' },
    { path: '/recipes', label: 'Rezepte', icon: 'book' },
    { path: '/shopping', label: 'Einkauf', icon: 'cart' },
    { path: '/statistics', label: 'Statistik', icon: 'chart' },
    { path: '/settings', label: 'Einstellungen', icon: 'gear' },
  ];

  // Die Bearbeiten-Seite samt Drag-and-Drop-Bibliothek wird erst geladen,
  // wenn sie gebraucht wird: import(...) als Funktion veranlasst Vite, sie in
  // eine eigene Datei zu legen. Übersicht und Kochansicht bleiben so schlank.
  const loadRecipeEditPage = () => import('./pages/RecipeEditPage.svelte');
  const loadRecipePickerPage = () => import('./pages/RecipePickerPage.svelte');
  const loadImportFilePage = () => import('./pages/ImportFilePage.svelte');

  // Alle Seiten. section gibt an, welcher Bereich in der Navigation
  // hervorgehoben wird. page ist eine Komponente, load eine Funktion, die
  // sie nachlädt.
  const routes = [
    { pattern: '/', section: '/', page: MealPlanPage },
    { pattern: '/recipes', section: '/recipes', page: RecipesPage },
    // "/recipes/new" vor "/recipes/:id", sonst gälte "new" als ID.
    { pattern: '/recipes/new', section: '/recipes', load: loadRecipeEditPage },
    { pattern: '/recipes/import', section: '/recipes', load: loadImportFilePage },
    { pattern: '/recipes/:id', section: '/recipes', page: RecipePage },
    { pattern: '/recipes/:id/edit', section: '/recipes', load: loadRecipeEditPage },
    // Kochansicht aus dem Essensplan: Portionen kommen aus dem Eintrag.
    { pattern: '/recipes/:id/entry/:entryId', section: '/', page: RecipePage },
    // Rezeptauswahl beim Planen bzw. Ersetzen
    { pattern: '/plan/add/:date', section: '/', load: loadRecipePickerPage },
    { pattern: '/plan/replace/:entryId', section: '/', load: loadRecipePickerPage },
    { pattern: '/shopping', section: '/shopping', page: ShoppingPage },
    { pattern: '/statistics', section: '/statistics', page: StatisticsPage },
    { pattern: '/settings', section: '/settings', page: SettingsPage },
  ];

  /** Sucht die Seite zu path samt Platzhalterwerten. */
  function resolve(path) {
    for (const route of routes) {
      const params = match(route.pattern, path);
      if (params) return { route, params };
    }
    return { route: { section: null, page: NotFoundPage }, params: {} };
  }

  // $derived berechnet den Wert neu, sobald sich currentPath() ändert.
  const current = $derived(resolve(currentPath()));
  const Page = $derived(current.route.page);
</script>

<div class="layout">
  <nav class="nav" aria-label="Hauptbereiche">
    {#each sections as section (section.path)}
      <a
        href={'#' + section.path}
        class="nav-item"
        aria-current={section.path === current.route.section ? 'page' : undefined}
      >
        <Icon name={section.icon} />
        <span>{section.label}</span>
      </a>
    {/each}
  </nav>

  <main class="content">
    <!-- {#key} baut die Seite neu auf, wenn sich die Adresse ändert, z. B.
         beim Wechsel von Rezept 3 zu Rezept 7. -->
    {#key currentPath()}
      {#if current.route.load}
        {#await current.route.load() then module}
          <module.default {...current.params} />
        {:catch}
          <p>Seite konnte nicht geladen werden. Bitte neu laden.</p>
        {/await}
      {:else}
        <Page {...current.params} />
      {/if}
    {/key}
  </main>
</div>

<style>
  .layout {
    min-height: 100dvh;
  }

  .content {
    max-width: 72rem;
    margin: 0 auto;
    padding: 1rem 1rem calc(var(--nav-height) + 1rem);
  }

  /* Handy: Leiste unten */
  .nav {
    position: fixed;
    inset: auto 0 0 0;
    z-index: 10;
    display: flex;
    height: var(--nav-height);
    padding-bottom: env(safe-area-inset-bottom);
    background: var(--surface);
    border-top: 1px solid var(--border);
  }

  .nav-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.15rem;
    font-size: 0.75rem;
    color: var(--text-muted);
    text-decoration: none;
  }

  .nav-item[aria-current='page'] {
    color: var(--accent);
    font-weight: 600;
  }

  /* Großer Bildschirm: Seitenleiste links */
  @media (min-width: 50rem) {
    .layout {
      display: grid;
      grid-template-columns: 13rem 1fr;
    }

    .nav {
      position: sticky;
      top: 0;
      flex-direction: column;
      height: 100dvh;
      padding: 1rem 0.5rem;
      gap: 0.25rem;
      border-top: none;
      border-right: 1px solid var(--border);
    }

    .nav-item {
      flex: none;
      flex-direction: row;
      justify-content: flex-start;
      gap: 0.75rem;
      padding: 0.6rem 0.75rem;
      border-radius: var(--radius);
      font-size: 0.95rem;
    }

    .nav-item:hover {
      background: var(--surface-hover);
    }

    .nav-item[aria-current='page'] {
      background: var(--accent-soft);
    }

    .content {
      width: 100%;
      padding: 1.5rem 2rem;
    }
  }
</style>
