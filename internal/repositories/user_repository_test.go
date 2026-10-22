package repositories_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"lifelink/internal/config"
	"lifelink/internal/database"
	"lifelink/internal/models"
	"lifelink/internal/repositories"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestUserRepository_InvalidID(t *testing.T) {
	repo := repositories.NewUserRepository(nil)
	ctx := context.Background()

	invalidID := "not-a-valid-hex-id"

	_, err := repo.FindByID(ctx, invalidID)
	if !errors.Is(err, repositories.ErrInvalidUserID) {
		t.Errorf("Expected ErrInvalidUserID on FindByID, got: %v", err)
	}
}

func TestUserRepository_CRUD_LiveOrSkip(t *testing.T) {
	cfg := &config.Config{
		MongoURI:    "mongodb://localhost:27017",
		MongoDBName: "lifelink_test_user_db",
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("Skipping live database user repository test: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := db.Database.Collection("users")
	_ = coll.Drop(ctx)
	defer func() {
		_ = coll.Drop(ctx)
	}()

	repo := repositories.NewUserRepository(db.Database)

	user := &models.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "hashedpassword123",
	}

	// 1. Create
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	if user.ID.IsZero() {
		t.Errorf("Expected user ID to be assigned an ObjectID")
	}

	// 2. Prevent duplicate email creation
	dupUser := &models.User{
		Name:     "Another Name",
		Email:    "TEST@example.com", // testing case-insensitivity
		Password: "hashedpassword456",
	}
	if err := repo.Create(ctx, dupUser); !errors.Is(err, repositories.ErrUserAlreadyExists) {
		t.Errorf("Expected ErrUserAlreadyExists on duplicate email, got: %v", err)
	}

	// 3. FindByEmail
	foundByEmail, err := repo.FindByEmail(ctx, "Test@Example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if foundByEmail.Name != "Test User" {
		t.Errorf("Expected user name 'Test User', got '%s'", foundByEmail.Name)
	}

	// 4. FindByID
	foundByID, err := repo.FindByID(ctx, user.ID.Hex())
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if foundByID.Email != "test@example.com" {
		t.Errorf("Expected user email 'test@example.com', got '%s'", foundByID.Email)
	}

	// 5. Non-existent email & ID
	_, err = repo.FindByEmail(ctx, "nonexistent@example.com")
	if !errors.Is(err, repositories.ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got: %v", err)
	}

	_, err = repo.FindByID(ctx, bson.NewObjectID().Hex())
	if !errors.Is(err, repositories.ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got: %v", err)
	}
}
