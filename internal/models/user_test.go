package models_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"lifelink/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestHashPasswordAndCheck(t *testing.T) {
	plain := "SecretPass123!"

	hash, err := models.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == plain {
		t.Fatalf("Hashed password must not equal plain text password")
	}

	// Verify correct password matches
	if !models.CheckPasswordHash(plain, hash) {
		t.Errorf("CheckPasswordHash failed for matching password")
	}

	// Verify wrong password does not match
	if models.CheckPasswordHash("WrongPassword", hash) {
		t.Errorf("CheckPasswordHash should return false for incorrect password")
	}
}

func TestUser_JSONDoesNotExposePassword(t *testing.T) {
	user := models.User{
		ID:        bson.NewObjectID(),
		Name:      "Test User",
		Email:     "test@example.com",
		Password:  "$2a$10$somesecretbcryptstringthatmustneverbeleaked",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "somesecretbcryptstringthatmustneverbeleaked") {
		t.Errorf("JSON output must NOT contain password hash! Got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"password"`) {
		t.Errorf("JSON output must NOT contain 'password' key! Got: %s", jsonStr)
	}
}

func TestRegisterRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     models.RegisterRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: models.RegisterRequest{
				Name:     "Test User",
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: models.RegisterRequest{
				Name:     "",
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "missing email",
			req: models.RegisterRequest{
				Name:     "Test User",
				Email:    "",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			req: models.RegisterRequest{
				Name:     "Test User",
				Email:    "not-an-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "password too short (<6 chars)",
			req: models.RegisterRequest{
				Name:     "Test User",
				Email:    "test@example.com",
				Password: "123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     models.LoginRequest
		wantErr bool
	}{
		{
			name: "valid login",
			req: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing email",
			req: models.LoginRequest{
				Email:    "",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			req: models.LoginRequest{
				Email:    "bad-email",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			req: models.LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
