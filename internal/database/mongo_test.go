package database_test

import (
	"testing"

	"lifelink/internal/config"
	"lifelink/internal/database"
)

func TestConnect_InvalidURI(t *testing.T) {
	cfg := &config.Config{
		MongoURI:    "invalid://scheme:not-valid",
		MongoDBName: "lifelink_test",
	}

	_, err := database.Connect(cfg)
	if err == nil {
		t.Fatal("Expected error when connecting with an invalid MongoDB URI, got nil")
	}
}

func TestConnect_LiveOrSkip(t *testing.T) {
	cfg := &config.Config{
		MongoURI:    "mongodb://localhost:27017",
		MongoDBName: "lifelink_test_db",
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("Skipping live MongoDB test (server might not be accessible): %v", err)
	}

	if db == nil || db.Client == nil || db.Database == nil {
		t.Fatal("Expected non-nil DB, Client, and Database instances")
	}

	// Verify clean disconnect
	if err := db.Close(); err != nil {
		t.Fatalf("Expected clean disconnect, got error: %v", err)
	}
}
