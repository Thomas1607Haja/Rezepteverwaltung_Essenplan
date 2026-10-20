<script>
  import { currentPath } from './lib/router.svelte.js';
  import Icon from './components/Icon.svelte';
  import MealPlanPage from './pages/MealPlanPage.svelte';
  import RecipesPage from './pages/RecipesPage.svelte';
  import ShoppingPage from './pages/ShoppingPage.svelte';
  import SettingsPage from './pages/SettingsPage.svelte';
  import NotFoundPage from './pages/NotFoundPage.svelte';

  // Die vier Hauptbereiche (docs/PLANUNG.md, Abschnitt 5).
  const sections = [
    { path: '/', label: 'Essensplan', icon: 'calendar', page: MealPlanPage },
    { path: '/recipes', label: 'Rezepte', icon: 'book', page: RecipesPage },
    { path: '/shopping', label: 'Einkauf', icon: 'cart', page: ShoppingPage },
    { path: '/settings', label: 'Einstellungen', icon: 'gear', page: SettingsPage },
  ];

  /** Gehört path zum Bereich section? "/recipes/12" gehört zu "/recipes". */
  function inSection(section, path) {
    if (section.path === '/') return path === '/';
    return path === section.path || path.startsWith(section.path + '/');
  }

  // $derived berechnet den Wert neu, sobald sich currentPath() ändert.
  const active = $derived(sections.find((s) => inSection(s, currentPath())));
  const Page = $derived(active?.page ?? NotFoundPage);
</script>

<div class="layout">
  <nav class="nav" aria-label="Hauptbereiche">
    {#each sections as section (section.path)}
      <a
        href={'#' + section.path}
        class="nav-item"
        aria-current={section === active ? 'page' : undefined}
      >
        <Icon name={section.icon} />
        <span>{section.label}</span>
      </a>
    {/each}
  </nav>

  <main class="content">
    <Page />
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
