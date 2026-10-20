// Einstiegspunkt: hängt die Anwendung in das Element #app der index.html.
import { mount } from 'svelte';
import App from './App.svelte';
import './app.css';

mount(App, { target: document.getElementById('app') });
