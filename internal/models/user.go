package models

import (
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

// User represents an authenticated application user stored in MongoDB.
// Notice that the Password field has json:"-" so plain-text or hashed passwords
// are NEVER returned in JSON API responses.
type User struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string        `bson:"name" json:"name"`
	Email     string        `bson:"email" json:"email"`
	Password  string        `bson:"password" json:"-"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

// Validate checks for required user fields.
func (u *User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(u.Email) == "" {
		return errors.New("email is required")
	}
	if !IsValidEmail(u.Email) {
		return errors.New("invalid email format")
	}
	if strings.TrimSpace(u.Password) == "" {
		return errors.New("password hash is required")
	}
	return nil
}

// RegisterRequest holds the JSON payload sent by clients to register a new user account.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks that all required fields are present and conform to basic rules.
func (r *RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(r.Email) == "" {
		return errors.New("email is required")
	}
	if !IsValidEmail(r.Email) {
		return errors.New("invalid email format")
	}
	if len(r.Password) < 6 {
		return errors.New("password must be at least 6 characters long")
	}
	return nil
}

// LoginRequest holds the JSON payload sent by clients to log into an existing account.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks that login credentials are provided.
func (l *LoginRequest) Validate() error {
	if strings.TrimSpace(l.Email) == "" {
		return errors.New("email is required")
	}
	if !IsValidEmail(l.Email) {
		return errors.New("invalid email format")
	}
	if strings.TrimSpace(l.Password) == "" {
		return errors.New("password is required")
	}
	return nil
}

// HashPassword hashes a plain-text password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash verifies a plain-text password against a bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
