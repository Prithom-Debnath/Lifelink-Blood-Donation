package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lifelink/internal/config"
	"lifelink/internal/routes"
)

func TestHealthCheckEndpoint(t *testing.T) {
	// 1. Arrange: setup configuration and router
	cfg := &config.Config{
		Port:   "8080",
		AppEnv: "test",
	}
	router := routes.SetupRouter(cfg)

	// 2. Act: create an HTTP GET request to /api/health
	req, err := http.NewRequest(http.MethodGet, "/api/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 3. Assert: check response status code and JSON payload
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}
