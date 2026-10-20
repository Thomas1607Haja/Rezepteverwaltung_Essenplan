import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],

  // Relative Pfade in der gebauten index.html: Home Assistant liefert die
  // Oberfläche über Ingress unter einem wechselnden Pfad aus
  // (/api/hassio_ingress/<token>/), absolute Pfade wie /assets/... gingen
  // dort ins Leere.
  base: './',

  build: {
    // go:embed kann nur Dateien innerhalb des Go-Moduls einbetten, daher
    // landet der Build direkt im Server.
    outDir: '../server/internal/web/dist',
    emptyOutDir: true,
  },

  server: {
    // In der Entwicklung (npm run dev) gehen API-Aufrufe an den lokal
    // laufenden Go-Server.
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
});
