// Rezeptplaner-Server: stellt die REST-API für Rezepte, Essensplan und
// Einkaufsliste bereit.
package main

import (
	"log"
	"net/http"
)

// listenAddr ist die Adresse, auf der der Server lauscht. Vorerst fest auf
// localhost, damit der Server nur vom eigenen Rechner aus erreichbar ist.
const listenAddr = "127.0.0.1:8080"

func main() {
	mux := http.NewServeMux()

	// Lebenszeichen: antwortet immer mit "ok", solange der Server läuft.
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("Server lauscht auf http://%s", listenAddr)
	// ListenAndServe läuft, bis ein Fehler auftritt (z. B. Port belegt).
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}
