<script>
  // Platzhalter: Die Kochansicht folgt in Schritt 4.
  let { id } = $props();
</script>

<p><a href="#/recipes">← Zur Übersicht</a></p>
<h1>Rezept {id}</h1>
<p class="placeholder">Die Kochansicht folgt in Schritt 4.</p>

<style>
  .placeholder {
    color: var(--text-muted);
  }
</style>
