package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"lifelink/internal/config"
	"lifelink/internal/database"
	"lifelink/internal/models"
	"lifelink/internal/repositories"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestDonorRepository_InvalidID(t *testing.T) {
	// A repository instance with nil collection is sufficient because ID validation happens before database query
	repo := repositories.NewDonorRepository(nil)
	ctx := context.Background()

	invalidID := "123-not-a-valid-hex-object-id"

	// FindByID with invalid ID
	_, err := repo.FindByID(ctx, invalidID)
	if !errors.Is(err, repositories.ErrInvalidID) {
		t.Errorf("Expected ErrInvalidID on FindByID, got: %v", err)
	}

	// Update with invalid ID
	err = repo.Update(ctx, invalidID, &models.Donor{})
	if !errors.Is(err, repositories.ErrInvalidID) {
		t.Errorf("Expected ErrInvalidID on Update, got: %v", err)
	}

	// Delete with invalid ID
	err = repo.Delete(ctx, invalidID)
	if !errors.Is(err, repositories.ErrInvalidID) {
		t.Errorf("Expected ErrInvalidID on Delete, got: %v", err)
	}
}

func TestDonorRepository_CRUD_LiveOrSkip(t *testing.T) {
	cfg := &config.Config{
		MongoURI:    "mongodb://localhost:27017",
		MongoDBName: "lifelink_test_crud_db",
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("Skipping live database CRUD test: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Clean up any old test collection before and after
	coll := db.Database.Collection("donors")
	defer coll.Drop(ctx)
	_ = coll.Drop(ctx)

	repo := repositories.NewDonorRepository(db.Database)

	// 1. Create a donor
	newDonor := &models.Donor{
		Name:         "Rahim Ahmed",
		Email:        "rahim@example.com",
		Phone:        "+8801711000000",
		BloodGroup:   "O+",
		City:         "Dhaka",
		Availability: true,
	}

	err = repo.Create(ctx, newDonor)
	if err != nil {
		t.Fatalf("Failed to create donor: %v", err)
	}

	if newDonor.ID.IsZero() {
		t.Fatal("Expected donor ID to be populated with a new ObjectID")
	}
	if newDonor.CreatedAt.IsZero() || newDonor.UpdatedAt.IsZero() {
		t.Fatal("Expected timestamps CreatedAt and UpdatedAt to be set")
	}

	donorID := newDonor.ID.Hex()

	// 2. Find by ID
	found, err := repo.FindByID(ctx, donorID)
	if err != nil {
		t.Fatalf("Failed to find donor by ID: %v", err)
	}
	if found.Name != "Rahim Ahmed" || found.BloodGroup != "O+" {
		t.Errorf("Found donor data mismatch. Got: %+v", found)
	}

	// 3. Find by non-existent ObjectID -> ErrNotFound
	nonExistentHex := bson.NewObjectID().Hex()
	_, err = repo.FindByID(ctx, nonExistentHex)
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("Expected ErrNotFound for non-existent ID, got: %v", err)
	}

	// 4. List donors
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list donors: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 donor in list, got %d", len(list))
	}

	// 5. Update donor
	found.Name = "Rahim Uddin"
	found.Availability = false
	err = repo.Update(ctx, donorID, found)
	if err != nil {
		t.Fatalf("Failed to update donor: %v", err)
	}

	// Verify update took effect
	updated, err := repo.FindByID(ctx, donorID)
	if err != nil {
		t.Fatalf("Failed to find updated donor: %v", err)
	}
	if updated.Name != "Rahim Uddin" || updated.Availability != false {
		t.Errorf("Updated fields mismatch. Got: %+v", updated)
	}

	// Update non-existent record -> ErrNotFound
	err = repo.Update(ctx, nonExistentHex, found)
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("Expected ErrNotFound on updating non-existent record, got: %v", err)
	}

	// 6. Delete donor
	err = repo.Delete(ctx, donorID)
	if err != nil {
		t.Fatalf("Failed to delete donor: %v", err)
	}

	// Verify donor is no longer found
	_, err = repo.FindByID(ctx, donorID)
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}

	// Delete non-existent record -> ErrNotFound
	err = repo.Delete(ctx, nonExistentHex)
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("Expected ErrNotFound on deleting non-existent record, got: %v", err)
	}

	fmt.Println("All donor CRUD repository operations verified successfully!")
}
