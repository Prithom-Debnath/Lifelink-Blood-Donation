package handlers_test

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
	"lifelink/internal/handlers"
	"lifelink/internal/middleware"
	"lifelink/internal/models"
	"lifelink/internal/repositories"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// mockUserRepo is a thread-safe in-memory implementation of repositories.UserRepository for fast testing.
type mockUserRepo struct {
	mu    sync.RWMutex
	users map[string]models.User // keyed by ID hex
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]models.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	normalizedEmail := strings.ToLower(strings.TrimSpace(user.Email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == normalizedEmail {
			return repositories.ErrUserAlreadyExists
		}
	}

	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.Email = normalizedEmail

	m.users[user.ID.Hex()] = *user
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == normalizedEmail {
			userCopy := u
			return &userCopy, nil
		}
	}
	return nil, repositories.ErrUserNotFound
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*models.User, error) {
	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return nil, repositories.ErrInvalidUserID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.users[id]
	if !exists {
		return nil, repositories.ErrUserNotFound
	}
	userCopy := u
	return &userCopy, nil
}

func setupAuthTestRouter(repo repositories.UserRepository, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	authHandler := handlers.NewAuthHandler(repo, cfg)
	api := router.Group("/api/auth")
	{
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)
		api.GET("/me", middleware.AuthMiddleware(cfg.JWTSecret), authHandler.GetMe)
	}

	return router
}

func TestAuthHandler_Register_Success(t *testing.T) {
	repo := newMockUserRepo()
	cfg := &config.Config{JWTSecret: "testsecretkey123"}
	router := setupAuthTestRouter(repo, cfg)

	payload := map[string]string{
		"name":     "Dr. Amina Rahman",
		"email":    "amina@example.com",
		"password": "securepassword123",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message string      `json:"message"`
		Token   string      `json:"token"`
		User    models.User `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if resp.Token == "" {
		t.Errorf("Expected non-empty JWT token in response")
	}
	if resp.User.Name != "Dr. Amina Rahman" {
		t.Errorf("Expected user name 'Dr. Amina Rahman', got '%s'", resp.User.Name)
	}
	if resp.User.Email != "amina@example.com" {
		t.Errorf("Expected user email 'amina@example.com', got '%s'", resp.User.Email)
	}

	// Verify plain text password is NEVER leaked in JSON response
	rawResp := w.Body.String()
	if strings.Contains(rawResp, "securepassword123") {
		t.Errorf("Response must not contain plain text password! Body: %s", rawResp)
	}
	if strings.Contains(rawResp, `"password"`) {
		t.Errorf("Response must not contain password key! Body: %s", rawResp)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	cfg := &config.Config{JWTSecret: "testsecretkey123"}
	router := setupAuthTestRouter(repo, cfg)

	// Pre-populate user
	hashed, _ := models.HashPassword("existingpass123")
	_ = repo.Create(context.Background(), &models.User{
		Name:     "Existing User",
		Email:    "existing@example.com",
		Password: hashed,
	})

	payload := map[string]string{
		"name":     "Duplicate Attempt",
		"email":    "EXISTING@example.com", // case-insensitive duplicate check
		"password": "password999",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict, got %d", w.Code)
	}
}

func TestAuthHandler_Register_ValidationErrors(t *testing.T) {
	repo := newMockUserRepo()
	cfg := &config.Config{JWTSecret: "testsecretkey123"}
	router := setupAuthTestRouter(repo, cfg)

	tests := []struct {
		name    string
		payload map[string]string
	}{
		{"missing name", map[string]string{"name": "", "email": "valid@example.com", "password": "password123"}},
		{"invalid email", map[string]string{"name": "Name", "email": "invalid-email", "password": "password123"}},
		{"short password", map[string]string{"name": "Name", "email": "valid@example.com", "password": "123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("[%s] Expected status 400 Bad Request, got %d", tt.name, w.Code)
			}
		})
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	cfg := &config.Config{JWTSecret: "testsecretkey123"}
	router := setupAuthTestRouter(repo, cfg)

	// Pre-create user with hashed password
	hashed, _ := models.HashPassword("mySecretPassword!")
	_ = repo.Create(context.Background(), &models.User{
		Name:     "Kazi Nazrul",
		Email:    "nazrul@example.com",
		Password: hashed,
	})

	payload := map[string]string{
		"email":    "nazrul@example.com",
		"password": "mySecretPassword!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Token == "" {
		t.Errorf("Expected token to be returned")
	}
	if resp.User.Email != "nazrul@example.com" {
		t.Errorf("Expected email 'nazrul@example.com', got '%s'", resp.User.Email)
	}

	// Validate the returned token
	claims, err := middleware.ValidateToken(resp.Token, cfg.JWTSecret)
	if err != nil {
		t.Fatalf("Returned token validation failed: %v", err)
	}
	if claims.Email != "nazrul@example.com" {
		t.Errorf("Expected claims email 'nazrul@example.com', got '%s'", claims.Email)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	repo := newMockUserRepo()
	cfg := &config.Config{JWTSecret: "testsecretkey123"}
	router := setupAuthTestRouter(repo, cfg)

	hashed, _ := models.HashPassword("correctPassword123")
	_ = repo.Create(context.Background(), &models.User{
		Name:     "Test User",
		Email:    "user@example.com",
		Password: hashed,
	})

	// Case 1: Wrong password
	{
		body, _ := json.Marshal(map[string]string{
			"email":    "user@example.com",
			"password": "wrongPassword!",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for wrong password, got %d", w.Code)
		}
	}

	// Case 2: Non-existent email
	{
		body, _ := json.Marshal(map[string]string{
			"email":    "nonexistent@example.com",
			"password": "somePassword123",
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for unknown email, got %d", w.Code)
		}
	}
}

func TestAuthHandler_GetMe(t *testing.T) {
	repo := newMockUserRepo()
	cfg := &config.Config{JWTSecret: "testsecretkey123"}
	router := setupAuthTestRouter(repo, cfg)

	hashed, _ := models.HashPassword("password123")
	user := &models.User{
		Name:     "Current User",
		Email:    "current@example.com",
		Password: hashed,
	}
	_ = repo.Create(context.Background(), user)

	token, _ := middleware.GenerateToken(user.ID.Hex(), user.Email, cfg.JWTSecret, time.Hour)

	// Case 1: Success with valid token
	{
		req, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 OK, got %d", w.Code)
		}

		var profile models.User
		_ = json.Unmarshal(w.Body.Bytes(), &profile)
		if profile.Email != "current@example.com" {
			t.Errorf("Expected email 'current@example.com', got '%s'", profile.Email)
		}
		if profile.Name != "Current User" {
			t.Errorf("Expected name 'Current User', got '%s'", profile.Name)
		}

		// Ensure password was not returned
		raw := w.Body.String()
		if strings.Contains(raw, "password") {
			t.Errorf("Password must not be present in /api/auth/me response! Got: %s", raw)
		}
	}

	// Case 2: Missing authorization header -> 401
	{
		req, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 without auth header, got %d", w.Code)
		}
	}
}
