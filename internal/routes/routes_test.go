package routes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"lifelink/internal/config"
	"lifelink/internal/models"
	"lifelink/internal/repositories"
	"lifelink/internal/routes"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type mockUserRepoForRoutes struct {
	mu    sync.RWMutex
	users map[string]models.User
}

func newMockUserRepoForRoutes() *mockUserRepoForRoutes {
	return &mockUserRepoForRoutes{
		users: make(map[string]models.User),
	}
}

func (m *mockUserRepoForRoutes) Create(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	norm := strings.ToLower(strings.TrimSpace(user.Email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == norm {
			return repositories.ErrUserAlreadyExists
		}
	}
	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.Email = norm
	m.users[user.ID.Hex()] = *user
	return nil
}

func (m *mockUserRepoForRoutes) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	norm := strings.ToLower(strings.TrimSpace(email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == norm {
			uCopy := u
			return &uCopy, nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

func (m *mockUserRepoForRoutes) FindByID(ctx context.Context, id string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return nil, repositories.ErrUserNotFound
	}
	uCopy := u
	return &uCopy, nil
}

func TestHealthCheckEndpoint(t *testing.T) {
	// 1. Arrange: setup configuration and router
	cfg := &config.Config{
		Port:      "8080",
		AppEnv:    "test",
		JWTSecret: "testsecret123",
	}
	router := routes.SetupRouter(cfg, nil, nil, nil)

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

func TestFrontendIndexServing(t *testing.T) {
	cfg := &config.Config{
		Port:      "8080",
		AppEnv:    "test",
		JWTSecret: "testsecret123",
	}
	router := routes.SetupRouter(cfg, nil, nil, nil)

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for GET /, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if len(contentType) == 0 {
		t.Errorf("Expected Content-Type header on GET /, got empty")
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Errorf("Expected non-empty body for frontend index.html")
	}
}

func TestAuthRoutesIntegration(t *testing.T) {
	cfg := &config.Config{
		Port:      "8080",
		AppEnv:    "test",
		JWTSecret: "supersecrettestjwtkey",
	}
	userRepo := newMockUserRepoForRoutes()
	router := routes.SetupRouter(cfg, nil, nil, userRepo)

	// 1. Register a new user account
	regPayload := []byte(`{
		"name": "Integration User",
		"email": "integration@example.com",
		"password": "mypassword123"
	}`)
	regReq, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(regPayload))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	router.ServeHTTP(regW, regReq)

	if regW.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created on register, got %d: %s", regW.Code, regW.Body.String())
	}

	var regResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(regW.Body.Bytes(), &regResp)
	if regResp.Token == "" {
		t.Fatalf("Expected non-empty token on register")
	}

	// 2. Login with registered credentials
	loginPayload := []byte(`{
		"email": "integration@example.com",
		"password": "mypassword123"
	}`)
	loginReq, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(loginPayload))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on login, got %d: %s", loginW.Code, loginW.Body.String())
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(loginW.Body.Bytes(), &loginResp)

	// 3. Access protected GET /api/auth/me with login token
	meReq, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	meW := httptest.NewRecorder()
	router.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on /api/auth/me, got %d: %s", meW.Code, meW.Body.String())
	}

	var user models.User
	_ = json.Unmarshal(meW.Body.Bytes(), &user)
	if user.Email != "integration@example.com" {
		t.Errorf("Expected email 'integration@example.com', got '%s'", user.Email)
	}
	if user.Name != "Integration User" {
		t.Errorf("Expected name 'Integration User', got '%s'", user.Name)
	}

	// 4. Access protected GET /api/auth/me without token -> 401
	unauthReq, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	unauthW := httptest.NewRecorder()
	router.ServeHTTP(unauthW, unauthReq)

	if unauthW.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized on /api/auth/me without token, got %d", unauthW.Code)
	}
}
