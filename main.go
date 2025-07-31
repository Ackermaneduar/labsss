package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

// Estructura del manifiesto original
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

// Estructura extendida para guardar también el puerto usado
type StoredManifest struct {
	Manifest
	Port int `json:"port"`
}

var (
	manifests = make(map[string]StoredManifest)
	mu        sync.Mutex
	portBase  = 8081
)

// Validación de campos obligatorios
func validateManifest(m Manifest) error {
	if strings.TrimSpace(m.Metadata.Name) == "" {
		return errors.New("el campo 'metadata.name' es obligatorio")
	}
	if strings.TrimSpace(m.Spec.Source.Image) == "" {
		return errors.New("el campo 'spec.source.image' es obligatorio")
	}
	return nil
}

// Ejecuta un comando y loggea salida y errores
func runCommand(name string, args ...string) error {
	log.Printf("Ejecutando: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	log.Printf("Resultado: %s", string(output))
	return err
}

// Ejecuta `docker pull`
func pullDockerImage(image string) error {
	return runCommand("docker", "pull", image)
}

// Detiene y elimina contenedor si existe
func stopAndRemoveContainer(name string) {
	_ = runCommand("docker", "rm", "-f", name)
}

// Ejecuta `docker run` en el puerto asignado
func runDockerContainer(manifest Manifest, port int) error {
	stopAndRemoveContainer(manifest.Metadata.Name)
	args := []string{
		"run", "-d",
		"--name", manifest.Metadata.Name,
		"-p", fmt.Sprintf("%d:80", port),
		manifest.Spec.Source.Image,
	}
	return runCommand("docker", args...)
}

func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

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

		if err := pullDockerImage(manifest.Spec.Source.Image); err != nil {
			log.Printf("Error al hacer pull: %v", err)
			http.Error(w, "Error al descargar la imagen con Docker", http.StatusInternalServerError)
			return
		}

		mu.Lock()
		port := portBase + len(manifests) // asignación dinámica
		mu.Unlock()

		if err := runDockerContainer(manifest, port); err != nil {
			log.Printf("Error al correr contenedor: %v", err)
			http.Error(w, "Error al iniciar el contenedor Docker", http.StatusInternalServerError)
			return
		}

		mu.Lock()
		manifests[manifest.Metadata.Name] = StoredManifest{
			Manifest: manifest,
			Port:     port,
		}
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"message": "Imagen " + manifest.Spec.Source.Image + " registrada y contenedor iniciado con nombre " + manifest.Metadata.Name,
			"port":    port,
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

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<h2> Servidor de Manifiestos</h2>
			<p>Usa <code>POST /api/v1/manifests</code> para registrar una imagen</p>
			<p>Usa <code>GET /api/v1/status</code> para ver las registradas</p>
		`))
	})

	return mux
}

func main() {
	mux := setupRoutes()
	log.Println("Servidor escuchando en :42113")
	log.Fatal(http.ListenAndServe(":42113", mux))
}
