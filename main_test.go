package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateManifest(t *testing.T) {
	manifests = make(map[string]Manifest)
	mux := setupRoutes()

	tests := []struct {
		name         string
		method       string
		payload      string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Manifiesto válido",
			method:       http.MethodPost,
			payload:      `{"metadata":{"name":"test1"},"spec":{"source":{"image":"nginx:latest"}}}`,
			expectedCode: http.StatusOK,
			expectedBody: `"status":"ok"`,
		},
		{
			name:         "JSON inválido",
			method:       http.MethodPost,
			payload:      `{malformed-json}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "JSON inválido",
		},
		{
			name:         "Nombre vacío",
			method:       http.MethodPost,
			payload:      `{"metadata":{"name":""},"spec":{"source":{"image":"nginx:latest"}}}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "el campo 'metadata.name' es obligatorio",
		},
		{
			name:         "Imagen vacía",
			method:       http.MethodPost,
			payload:      `{"metadata":{"name":"test2"},"spec":{"source":{"image":""}}}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "el campo 'spec.source.image' es obligatorio",
		},
		{
			name:         "Método no permitido en manifests",
			method:       http.MethodGet,
			payload:      "",
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "Método no permitido",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.payload != "" {
				req = httptest.NewRequest(tt.method, "/api/v1/manifests", strings.NewReader(tt.payload))
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/manifests", nil)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedCode {
				t.Errorf("Código de estado incorrecto: obtuve %v, esperaba %v", status, tt.expectedCode)
			}

			if !strings.Contains(rr.Body.String(), tt.expectedBody) {
				t.Errorf("Cuerpo de respuesta inesperado: %v", rr.Body.String())
			}
		})
	}
}

func TestGetStatus(t *testing.T) {
	manifests = map[string]Manifest{
		"test1": {
			Metadata: struct{ Name string `json:"name"` }{Name: "test1"},
			Spec: struct{ Source struct{ Image string `json:"image"` } `json:"source"` }{
				Source: struct{ Image string `json:"image"` }{Image: "nginx:latest"},
			},
		},
	}

	mux := setupRoutes()

	t.Run("GET válido", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Código de estado incorrecto: obtuve %v, esperaba %v", status, http.StatusOK)
		}

		var response map[string]Manifest
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("No se pudo decodificar la respuesta: %v", err)
		}

		if len(response) != 1 || response["test1"].Spec.Source.Image != "nginx:latest" {
			t.Errorf("Respuesta inesperada: %v", response)
		}
	})

	t.Run("Método no permitido", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/status", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("Código de estado incorrecto: obtuve %v, esperaba %v", status, http.StatusMethodNotAllowed)
		}

		if !strings.Contains(rr.Body.String(), "Método no permitido") {
			t.Errorf("Cuerpo de respuesta inesperado: %v", rr.Body.String())
		}
	})
}
