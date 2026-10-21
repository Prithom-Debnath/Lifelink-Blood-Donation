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

func TestBloodRequestRepository_InvalidID(t *testing.T) {
	repo := repositories.NewBloodRequestRepository(nil)
	ctx := context.Background()

	invalidID := "not-a-valid-24-char-hex-id"

	_, err := repo.FindByID(ctx, invalidID)
	if !errors.Is(err, repositories.ErrInvalidBloodRequestID) {
		t.Errorf("Expected ErrInvalidBloodRequestID, got: %v", err)
	}
}

func TestBloodRequestRepository_LiveOrSkip(t *testing.T) {
	cfg := &config.Config{
		MongoURI:    "mongodb://localhost:27017",
		MongoDBName: "lifelink_test_blood_requests_db",
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("Skipping live blood request repository test: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := db.Database.Collection("blood_requests")
	defer coll.Drop(ctx)
	_ = coll.Drop(ctx)

	repo := repositories.NewBloodRequestRepository(db.Database)

	// 1. Test Create with status defaulting to "open"
	req1 := &models.BloodRequest{
		PatientName:  "Zubair Ahmed",
		BloodGroup:   "O-",
		Hospital:     "Kurmitola General Hospital",
		City:         "Dhaka",
		ContactPhone: "+8801511223344",
	}

	err = repo.Create(ctx, req1)
	if err != nil {
		t.Fatalf("Failed to create blood request: %v", err)
	}

	if req1.ID.IsZero() {
		t.Fatal("Expected ObjectID to be generated")
	}
	if req1.Status != models.StatusOpen {
		t.Errorf("Expected status to default to '%s', got '%s'", models.StatusOpen, req1.Status)
	}
	if req1.CreatedAt.IsZero() || req1.UpdatedAt.IsZero() {
		t.Fatal("Expected timestamps CreatedAt and UpdatedAt to be set")
	}

	reqID := req1.ID.Hex()

	// 2. Test FindByID
	found, err := repo.FindByID(ctx, reqID)
	if err != nil {
		t.Fatalf("Failed to find blood request by ID: %v", err)
	}
	if found.PatientName != "Zubair Ahmed" || found.Hospital != "Kurmitola General Hospital" {
		t.Errorf("Found blood request data mismatch: %+v", found)
	}

	// 3. Test FindByID with non-existent ObjectID -> ErrBloodRequestNotFound
	nonExistentHex := bson.NewObjectID().Hex()
	_, err = repo.FindByID(ctx, nonExistentHex)
	if !errors.Is(err, repositories.ErrBloodRequestNotFound) {
		t.Errorf("Expected ErrBloodRequestNotFound for non-existent ID, got: %v", err)
	}

	// 4. Test Create a second request with explicit status "fulfilled"
	req2 := &models.BloodRequest{
		PatientName:  "Sadia Rahman",
		BloodGroup:   "B+",
		Hospital:     "Square Hospital",
		City:         "Dhaka",
		ContactPhone: "+8801719887766",
		Status:       models.StatusFulfilled,
	}
	err = repo.Create(ctx, req2)
	if err != nil {
		t.Fatalf("Failed to create second blood request: %v", err)
	}
	if req2.Status != models.StatusFulfilled {
		t.Errorf("Expected status '%s', got '%s'", models.StatusFulfilled, req2.Status)
	}

	// 5. Test List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list blood requests: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 blood requests in list, got %d", len(list))
	}
}
