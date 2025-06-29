package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"strings"
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

func validateManifest(m Manifest) error {
	if strings.TrimSpace(m.Metadata.Name) == "" {
		return errors.New("el campo 'metadata.name' es obligatorio")
	}
	if strings.TrimSpace(m.Spec.Source.Image) == "" {
		return errors.New("el campo 'spec.source.image' es obligatorio")
	}
	return nil
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/manifests", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Recibida %s %s", r.Method, r.URL.Path)

		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var manifest Manifest
		if err := json.NewDecoder(r.Body).Decode(&manifest); err != nil {
			log.Printf("Error decodificando JSON: %v", err)
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}

		if err := validateManifest(manifest); err != nil {
			log.Printf("Validación fallida: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
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

	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Recibida %s %s", r.Method, r.URL.Path)

		if r.Method != http.MethodGet {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		json.NewEncoder(w).Encode(manifests)
	})

	return mux
}

func main() {
	mux := setupRoutes()
	log.Println("Servidor escuchando en :42113")
	log.Fatal(http.ListenAndServe(":42113", mux))
}
