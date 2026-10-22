package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lifelink/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test_secret_key_123"
	userID := "60d5ec49f1b24b2b8c8e1234"
	email := "donor@example.com"

	// 1. Generate valid token with 1 hour duration
	tokenStr, err := middleware.GenerateToken(userID, email, secret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if tokenStr == "" {
		t.Fatalf("Expected non-empty token string")
	}

	// 2. Validate token
	claims, err := middleware.ValidateToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	tokenStr, err := middleware.GenerateToken("user123", "user@example.com", "correct_secret", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = middleware.ValidateToken(tokenStr, "wrong_secret")
	if err == nil {
		t.Errorf("Expected error when validating token with wrong secret, got nil")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	// Generate an already-expired token (-1 minute)
	tokenStr, err := middleware.GenerateToken("user123", "user@example.com", "secret", -time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = middleware.ValidateToken(tokenStr, "secret")
	if err == nil {
		t.Errorf("Expected error for expired token, got nil")
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "jwt_test_secret"

	router := gin.New()
	router.GET("/protected", middleware.AuthMiddleware(secret), func(c *gin.Context) {
		userID, _ := c.Get("userID")
		email, _ := c.Get("userEmail")
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   email,
		})
	})

	// Case 1: Missing Authorization header -> 401
	{
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for missing header, got %d", w.Code)
		}
	}

	// Case 2: Malformed Authorization header -> 401
	{
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "InvalidHeaderFormat")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for malformed header, got %d", w.Code)
		}
	}

	// Case 3: Valid Bearer token -> 200
	{
		token, _ := middleware.GenerateToken("60d5ec49f1b24b2b8c8e1234", "user@example.com", secret, time.Hour)
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for valid token, got %d (body: %s)", w.Code, w.Body.String())
		}

		var resp map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["user_id"] != "60d5ec49f1b24b2b8c8e1234" {
			t.Errorf("Expected user_id '60d5ec49f1b24b2b8c8e1234', got '%s'", resp["user_id"])
		}
	}
}
