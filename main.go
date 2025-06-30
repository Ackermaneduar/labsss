package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

// Estructura del manifiesto
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

// Ejecuta `docker run` con nombre de contenedor igual a manifest.Metadata.Name
func runDockerContainer(manifest Manifest) error {
	// Primero detener y eliminar contenedor existente con ese nombre
	stopAndRemoveContainer(manifest.Metadata.Name)

	// Ejecutar nuevo contenedor en modo detached (-d)
	// Mapea puerto 80 del contenedor al puerto 8080 + un offset para evitar choques (opcional)
	// Aquí solo mapeamos puerto 80 al 8080 para que puedas ajustar según la imagen
	args := []string{
		"run", "-d",
		"--name", manifest.Metadata.Name,
		"-p", "8081:80",
		manifest.Spec.Source.Image,
	}

	return runCommand("docker", args...)
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

		if err := pullDockerImage(manifest.Spec.Source.Image); err != nil {
			log.Printf("Error al hacer pull: %v", err)
			http.Error(w, "Error al descargar la imagen con Docker", http.StatusInternalServerError)
			return
		}

		if err := runDockerContainer(manifest); err != nil {
			log.Printf("Error al correr contenedor: %v", err)
			http.Error(w, "Error al iniciar el contenedor Docker", http.StatusInternalServerError)
			return
		}

		mu.Lock()
		manifests[manifest.Metadata.Name] = manifest
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "Imagen " + manifest.Spec.Source.Image + " registrada y contenedor iniciado con nombre " + manifest.Metadata.Name,
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
