# Glossar / Glossary

Verbindliche Begriffe für alle Übersetzungen (Weboberfläche, Android-App,
Integration, Dashboard-Karte, Dokumentation). Wer übersetzt, hält sich an
diese Tabelle; neue Begriffe kommen zuerst hierher.

Binding terms for every translation. Add new terms here first.

## Grundsätze / Principles

- Kurz und natürlich wie in einer gut gemachten App, nicht wörtlich.
  Short and natural, like a well-made app – not word for word.
- Anrede: Deutsch „du“, Englisch neutral (Imperativ), Französisch „vous“,
  Spanisch „tú“, Italienisch „tu“.
- Knöpfe als Verb im Infinitiv bzw. Imperativ: „Speichern“, „Save“,
  „Enregistrer“, „Guardar“, „Salva“.
- Der Name **Rezeptplaner** und **Bring** werden nicht übersetzt.
- Inhalte des Nutzers (Rezepte, Zutaten, Kategorien, Notizen) werden nie
  übersetzt – nur die Oberfläche.

## Bereiche / Sections

| Deutsch | English | Français | Español | Italiano |
|---|---|---|---|---|
| Essensplan | Meal plan | Planning des repas | Plan de comidas | Piano pasti |
| Rezepte | Recipes | Recettes | Recetas | Ricette |
| Einkauf | Shopping | Courses | Compra | Spesa |
| Statistik | Statistics | Statistiques | Estadísticas | Statistiche |
| Einstellungen | Settings | Paramètres | Ajustes | Impostazioni |

## Fachbegriffe / Domain terms

| Deutsch | English | Français | Español | Italiano |
|---|---|---|---|---|
| Rezept | recipe | recette | receta | ricetta |
| Zutat | ingredient | ingrédient | ingrediente | ingrediente |
| Zubereitung, Schritt | steps, step | étapes, étape | pasos, paso | passaggi, passaggio |
| Portion(en) | serving(s) | portion(s) | ración(es) | porzione/i |
| Arbeitszeit | prep time | préparation | preparación | preparazione |
| Gesamtzeit | total time | temps total | tiempo total | tempo totale |
| Variante | variant | variante | variante | variante |
| Kategorie | category | catégorie | categoría | categoria |
| Gruppe (von Kategorien) | group | groupe | grupo | gruppo |
| Gang | course | plat | plato | portata |
| Menü (mehrere Gänge) | menu | menu | menú | menù |
| Favorit | favorite | favori | favorito | preferito |
| Einkaufsliste | shopping list | liste de courses | lista de la compra | lista della spesa |
| einkaufsrelevant | on the shopping list | sur la liste de courses | en la lista de la compra | nella lista della spesa |
| Einkaufstag | shopping day | jour des courses | día de la compra | giorno della spesa |
| Vorrat, Vorratsliste | pantry, pantry list | garde-manger | despensa | dispensa |
| Synonym | alternative name | autre nom | otro nombre | altro nome |
| geplant | planned | prévu | previsto | previsto |
| Zutaten vorhanden | ingredients at home | ingrédients à la maison | ingredientes en casa | ingredienti in casa |
| an Bring übertragen | sent to Bring | envoyé à Bring | enviado a Bring | inviato a Bring |
| Übertragung zurücksetzen | undo sending | annuler l'envoi | deshacer el envío | annulla l'invio |
| einplanen | add to plan | planifier | añadir al plan | aggiungi al piano |
| tauschen | swap | échanger | intercambiar | scambia |
| verschieben | move | déplacer | mover | sposta |
| Haushalt | household | foyer | hogar | famiglia |
| Familiengröße | household size | taille du foyer | personas en casa | persone in casa |
| Kurzvorschau | quick preview | aperçu rapide | vista rápida | anteprima rapida |
| Verbundene Geräte | connected devices | appareils connectés | dispositivos conectados | dispositivi collegati |
| API-Schlüssel | API key | clé API | clave API | chiave API |
| Abgleich | sync | synchronisation | sincronización | sincronizzazione |
| Import, importieren | import | import, importer | importación, importar | importazione, importa |
| Unterstützen | Support | Soutenir | Apoyar | Sostieni |

## Tageszeiten (Slots) / Meal slots

Intern gespeichert als Schlüssel; nur die Anzeige wird übersetzt.

| Schlüssel | Deutsch | English | Français | Español | Italiano |
|---|---|---|---|---|---|
| `lunch` | Mittag | Lunch | Déjeuner | Comida | Pranzo |
| `coffee` | Kaffee & Kuchen | Afternoon tea | Goûter | Merienda | Merenda |
| `dinner` | Abend | Dinner | Dîner | Cena | Cena |

## Gänge (Startliste) / Courses (default list)

Schlüssel in `tags.course_key`. Die Namen sind Daten: Sie werden bei einer
Neuinstallation in der Sprache von Home Assistant angelegt und danach nie
übersetzt.

| Schlüssel | Deutsch | English | Français | Español | Italiano |
|---|---|---|---|---|---|
| `starter` | Vorspeise | Starter | Entrée | Entrante | Antipasto |
| `soup` | Suppe/Eintopf | Soup & stew | Soupe & ragoût | Sopa y guiso | Zuppa e stufato |
| `salad` | Salat | Salad | Salade | Ensalada | Insalata |
| `main` | Hauptgericht | Main course | Plat principal | Plato principal | Piatto principale |
| `side` | Beilage | Side dish | Accompagnement | Guarnición | Contorno |
| `dessert` | Nachspeise | Dessert | Dessert | Postre | Dolce |
| `cake` | Kuchen & Torten | Cakes & tarts | Gâteaux & tartes | Tartas y pasteles | Torte e crostate |
| `pastry` | Gebäck | Cookies & pastries | Biscuits & viennoiseries | Galletas y bollería | Biscotti e pasticcini |
| `bread` | Brot | Bread | Pain | Pan | Pane |
| `snack` | Snack | Snack | En-cas | Picoteo | Spuntino |

## Schnellauswahl / Quick picks

| Deutsch | English | Français | Español | Italiano |
|---|---|---|---|---|
| Essen gehen | Eating out | Restaurant | Comer fuera | Fuori a cena |
| Reste | Leftovers | Restes | Sobras | Avanzi |
| Pizza bestellen | Order pizza | Commander une pizza | Pedir pizza | Ordinare la pizza |

## Einheiten / Units

Einheiten in Rezepten sind Inhalt und bleiben, wie sie importiert wurden.
Der Parser erkennt sie in allen Sprachen und schreibt sie einheitlich
(z. B. „Esslöffel“ → `EL`, „tablespoon“ → `tbsp`), übersetzt sie aber nicht.

| Deutsch | English | Français | Español | Italiano |
|---|---|---|---|---|
| EL | tbsp | c. à s. | cda | cucchiaio |
| TL | tsp | c. à c. | cdta | cucchiaino |
| Prise | pinch | pincée | pizca | pizzico |
| Stück | piece | pièce | pieza | pezzo |
| Zehe | clove | gousse | diente | spicchio |
| Dose | can | boîte | lata | lattina |
| Bund | bunch | botte | manojo | mazzo |
