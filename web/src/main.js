// Einstiegspunkt: lädt die Sprache und hängt dann die Anwendung in das
// Element #app der index.html.
import { mount } from 'svelte';
import App from './App.svelte';
import { api } from './lib/api.js';
import { applyLanguage } from './lib/i18n.svelte.js';
import './app.css';

// Erst die Sprache, dann die Oberfläche – sonst blitzt kurz Englisch auf.
// Ist der Server nicht erreichbar, gilt die Sprache des Browsers.
api
  .language()
  .catch(() => null)
  .then(applyLanguage)
  .catch(() => {})
  .finally(() => mount(App, { target: document.getElementById('app') }));
