package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

// Estructuras para el manifiesto
type Manifest struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Source struct {
			Image string `json:"image"`
		} `json:"source"`
	} `json:"spec"`
}

var (
	manifests = make(map[string]Manifest)
	mu        sync.Mutex
)

func main() {
	// Endpoint para crear manifiestos (POST)
	http.HandleFunc("/manifiesto", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var manifest Manifest
		if err := json.NewDecoder(r.Body).Decode(&manifest); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}

		mu.Lock()
		manifests[manifest.Metadata.Name] = manifest
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "Imagen " + manifest.Spec.Source.Image + " registrada como " + manifest.Metadata.Name,
		})
	})

	// Endpoint para ver estado (GET)
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(manifests)
	})

	log.Println("Servidor escuchando en :42113")
	log.Fatal(http.ListenAndServe(":42113", nil))
}