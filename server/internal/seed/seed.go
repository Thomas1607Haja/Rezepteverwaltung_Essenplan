// Package seed passt die Startdaten einer neuen Installation an die
// Sprache des Haushalts an.
//
// Die Migrationen legen die Startdaten auf Deutsch an (Kategorien, Gänge,
// Vorratsliste) – so ist es seit der ersten Version, und bestehende
// Installationen bleiben genau so. Nur wenn die Datenbank beim aktuellen
// Start neu angelegt wurde, benennt Localize diese Einträge in die Sprache
// von Home Assistant um. Danach sind es ganz normale Daten des Nutzers:
// Sie werden nie wieder übersetzt.
package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// names: deutscher Name aus den Migrationen → Name je Sprache (en, fr, es, it).
type names map[string][4]string

func (n names) in(lang, german string) string {
	i := map[string]int{"en": 0, "fr": 1, "es": 2, "it": 3}[lang]
	return n[german][i]
}

// groups sind die Kategoriegruppen der Startliste.
var groups = names{
	"Gang":          {"Course", "Plat", "Plato", "Portata"},
	"Hauptzutat":    {"Main ingredient", "Ingrédient principal", "Ingrediente principal", "Ingrediente principale"},
	"Geschmack":     {"Taste", "Saveur", "Sabor", "Sapore"},
	"Art":           {"Type", "Type", "Tipo", "Tipo"},
	"Eigenschaften": {"Features", "Caractéristiques", "Características", "Caratteristiche"},
}

// tags sind die Kategorien der Startliste, Gänge wie in docs/glossar.md.
var tags = names{
	"Vorspeise":       {"Starter", "Entrée", "Entrante", "Antipasto"},
	"Suppe/Eintopf":   {"Soup & stew", "Soupe & ragoût", "Sopa y guiso", "Zuppa e stufato"},
	"Salat":           {"Salad", "Salade", "Ensalada", "Insalata"},
	"Hauptgericht":    {"Main course", "Plat principal", "Plato principal", "Piatto principale"},
	"Beilage":         {"Side dish", "Accompagnement", "Guarnición", "Contorno"},
	"Nachspeise":      {"Dessert", "Dessert", "Postre", "Dolce"},
	"Kuchen & Torten": {"Cakes & tarts", "Gâteaux & tartes", "Tartas y pasteles", "Torte e crostate"},
	"Gebäck":          {"Cookies & pastries", "Biscuits & viennoiseries", "Galletas y bollería", "Biscotti e pasticcini"},
	"Brot":            {"Bread", "Pain", "Pan", "Pane"},
	"Snack":           {"Snack", "En-cas", "Picoteo", "Spuntino"},

	"Vegetarisch":   {"Vegetarian", "Végétarien", "Vegetariano", "Vegetariano"},
	"Vegan":         {"Vegan", "Végan", "Vegano", "Vegano"},
	"Rind":          {"Beef", "Bœuf", "Ternera", "Manzo"},
	"Schwein":       {"Pork", "Porc", "Cerdo", "Maiale"},
	"Geflügel":      {"Poultry", "Volaille", "Aves", "Pollame"},
	"Hackfleisch":   {"Minced meat", "Viande hachée", "Carne picada", "Carne macinata"},
	"Fisch":         {"Fish", "Poisson", "Pescado", "Pesce"},
	"Meeresfrüchte": {"Seafood", "Fruits de mer", "Marisco", "Frutti di mare"},

	"Herzhaft": {"Savory", "Salé", "Salado", "Salato"},
	"Süß":      {"Sweet", "Sucré", "Dulce", "Dolce"},

	"Pasta":        {"Pasta", "Pâtes", "Pasta", "Pasta"},
	"Reis":         {"Rice", "Riz", "Arroz", "Riso"},
	"Kartoffeln":   {"Potatoes", "Pommes de terre", "Patatas", "Patate"},
	"Auflauf/Ofen": {"Bakes & oven", "Gratins & four", "Al horno", "Al forno"},
	"Pfanne":       {"Pan", "Poêle", "Sartén", "Padella"},
	"Grill":        {"Grill", "Barbecue", "Parrilla", "Griglia"},

	"Schnell":           {"Quick", "Rapide", "Rápido", "Veloce"},
	"Kinderliebling":    {"Kids' favorite", "Les enfants adorent", "Favorito de los niños", "Piace ai bambini"},
	"Gut vorzubereiten": {"Make ahead", "À préparer à l'avance", "Para preparar antes", "Si prepara in anticipo"},
	"Resteverwertung":   {"Uses leftovers", "Anti-gaspi", "Aprovechar sobras", "Riciclo avanzi"},
	"Für Gäste":         {"For guests", "Pour les invités", "Para invitados", "Per ospiti"},
}

// pantryItem ist ein Eintrag der Vorratsliste mit Synonymen (kommagetrennt).
type pantryItem struct{ name, synonyms string }

// pantry ist die Vorratsliste der Startdaten je Sprache (Deutsch steht in
// Migration 004).
var pantry = map[string][]pantryItem{
	"en": {
		{"Salt", "sea salt, table salt, iodized salt, kosher salt"},
		{"Pepper", "black pepper, ground pepper, freshly ground pepper"},
		{"Oil", "olive oil, vegetable oil, sunflower oil, rapeseed oil, canola oil, cooking oil"},
		{"Sugar", ""},
		{"Water", ""},
	},
	"fr": {
		{"Sel", "sel fin, gros sel, sel de mer, fleur de sel"},
		{"Poivre", "poivre noir, poivre du moulin, poivre moulu"},
		{"Huile", "huile d'olive, huile de tournesol, huile de colza, huile végétale"},
		{"Sucre", "sucre en poudre"},
		{"Eau", ""},
	},
	"es": {
		{"Sal", "sal fina, sal gruesa, sal marina, sal yodada"},
		{"Pimienta", "pimienta negra, pimienta molida"},
		{"Aceite", "aceite de oliva, aceite de oliva virgen extra, aceite de girasol, aceite vegetal"},
		{"Azúcar", ""},
		{"Agua", ""},
	},
	"it": {
		{"Sale", "sale fino, sale grosso, sale marino"},
		{"Pepe", "pepe nero, pepe macinato"},
		{"Olio", "olio d'oliva, olio extravergine di oliva, olio di semi, olio di girasole"},
		{"Zucchero", ""},
		{"Acqua", ""},
	},
}

// Localize benennt die Startdaten einer frisch angelegten Datenbank in die
// Sprache lang um. Deutsch (die Sprache der Migrationen) und nicht
// unterstützte Sprachen lassen alles, wie es ist. Nur aufrufen, wenn die
// Datenbank in diesem Start neu entstanden ist!
func Localize(ctx context.Context, db *sql.DB, lang string) error {
	if _, ok := pantry[lang]; !ok {
		return nil
	}
	now := database.FormatTime(time.Now())
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("startdaten übersetzen: %w", err)
	}
	defer tx.Rollback()

	for german := range groups {
		if _, err := tx.ExecContext(ctx, "UPDATE tag_groups SET name = ?, updated_at = ? WHERE name = ?",
			groups.in(lang, german), now, german); err != nil {
			return fmt.Errorf("startdaten übersetzen: %w", err)
		}
	}
	for german := range tags {
		if _, err := tx.ExecContext(ctx, "UPDATE tags SET name = ?, updated_at = ? WHERE name = ?",
			tags.in(lang, german), now, german); err != nil {
			return fmt.Errorf("startdaten übersetzen: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM pantry_items"); err != nil {
		return fmt.Errorf("startdaten übersetzen: %w", err)
	}
	for _, it := range pantry[lang] {
		if _, err := tx.ExecContext(ctx, "INSERT INTO pantry_items (name, synonyms, updated_at) VALUES (?, ?, ?)",
			it.name, it.synonyms, now); err != nil {
			return fmt.Errorf("startdaten übersetzen: %w", err)
		}
	}
	return tx.Commit()
}
