package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"lifelink/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	// ErrBloodRequestNotFound is returned when a requested blood request is not found in MongoDB.
	ErrBloodRequestNotFound = errors.New("blood request not found")

	// ErrInvalidBloodRequestID is returned when an ID cannot be parsed as a valid MongoDB ObjectID hex string.
	ErrInvalidBloodRequestID = errors.New("invalid blood request id format")

	// ErrBloodRequestNotOpen is returned when a closed request is changed again.
	ErrBloodRequestNotOpen       = errors.New("blood request is no longer open")
	ErrBloodRequestNotDeletable  = errors.New("blood request cannot be deleted")
	ErrBloodRequestForbidden     = errors.New("blood request is owned by another account")
	ErrInvalidBloodRequestStatus = errors.New("invalid blood request status")
)

type OwnedBloodRequestRepository interface {
	UpdateOwned(ctx context.Context, id, ownerUserID string, req *models.BloodRequest) error
	UpdateStatusOwned(ctx context.Context, id, ownerUserID, status string) error
	DeleteOwned(ctx context.Context, id, ownerUserID string) error
}

// BloodRequestRepository defines data access methods for patient blood requests.
type BloodRequestRepository interface {
	Create(ctx context.Context, req *models.BloodRequest) error
	List(ctx context.Context) ([]models.BloodRequest, error)
	FindByID(ctx context.Context, id string) (*models.BloodRequest, error)
	Update(ctx context.Context, id string, req *models.BloodRequest) error
	UpdateStatus(ctx context.Context, id, status string) error
}

// MongoBloodRequestRepository implements BloodRequestRepository using MongoDB.
type MongoBloodRequestRepository struct {
	collection *mongo.Collection
}

// NewBloodRequestRepository creates a new MongoBloodRequestRepository for the "blood_requests" collection.
func NewBloodRequestRepository(db *mongo.Database) *MongoBloodRequestRepository {
	var coll *mongo.Collection
	if db != nil {
		coll = db.Collection("blood_requests")
	}
	return &MongoBloodRequestRepository{
		collection: coll,
	}
}

// Create inserts a new blood request into MongoDB.
// Automatically generates an ObjectID, defaults status to "open" if unspecified, and sets UTC timestamps.
func (r *MongoBloodRequestRepository) Create(ctx context.Context, req *models.BloodRequest) error {
	if req.ID.IsZero() {
		req.ID = bson.NewObjectID()
	}

	if req.Status == "" {
		req.Status = models.StatusOpen
	}
	if !models.IsValidStatus(req.Status) {
		return ErrInvalidBloodRequestStatus
	}

	if req.Urgency == "" {
		req.Urgency = models.UrgencyNormal
	} else {
		req.Urgency = models.NormalizeUrgency(req.Urgency)
	}

	now := time.Now().UTC()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	req.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to insert blood request into database: %w", err)
	}

	return nil
}

// List retrieves all blood requests from MongoDB.
// Guarantees an initialized empty slice `[]` rather than `null` when no records exist.
func (r *MongoBloodRequestRepository) List(ctx context.Context) ([]models.BloodRequest, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query blood requests: %w", err)
	}
	defer cursor.Close(ctx)

	requests := make([]models.BloodRequest, 0)
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, fmt.Errorf("failed to decode blood requests cursor: %w", err)
	}

	return requests, nil
}

// FindByID retrieves a single blood request by its hex ObjectID.
// Returns ErrInvalidBloodRequestID if the hex string is invalid, or ErrBloodRequestNotFound if not found.
func (r *MongoBloodRequestRepository) FindByID(ctx context.Context, id string) (*models.BloodRequest, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidBloodRequestID
	}

	var req models.BloodRequest
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&req)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrBloodRequestNotFound
		}
		return nil, fmt.Errorf("failed to find blood request by ID: %w", err)
	}

	return &req, nil
}

// Update edits the details of an open request without changing its status or creation time.
func (r *MongoBloodRequestRepository) Update(ctx context.Context, id string, req *models.BloodRequest) error {
	return r.update(ctx, id, "", req)
}

func (r *MongoBloodRequestRepository) UpdateOwned(ctx context.Context, id, ownerUserID string, req *models.BloodRequest) error {
	return r.update(ctx, id, ownerUserID, req)
}

func (r *MongoBloodRequestRepository) update(ctx context.Context, id, ownerUserID string, req *models.BloodRequest) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidBloodRequestID
	}

	now := time.Now().UTC()
	requestFilter := bson.M{"_id": objID, "status": models.StatusOpen}
	if ownerUserID != "" {
		requestFilter["owner_user_id"] = ownerUserID
	}
	result, err := r.collection.UpdateOne(ctx,
		requestFilter,
		bson.M{"$set": bson.M{
			"patient_name":  req.PatientName,
			"blood_group":   req.BloodGroup,
			"urgency":       req.Urgency,
			"hospital":      req.Hospital,
			"city":          req.City,
			"contact_phone": req.ContactPhone,
			"updated_at":    now,
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update blood request: %w", err)
	}
	if result.MatchedCount == 0 {
		return r.ensureOwnedRequestIsOpen(ctx, objID, ownerUserID)
	}
	return nil
}

// UpdateStatus closes an open request as fulfilled.
func (r *MongoBloodRequestRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.updateStatus(ctx, id, "", status)
}

func (r *MongoBloodRequestRepository) UpdateStatusOwned(ctx context.Context, id, ownerUserID, status string) error {
	return r.updateStatus(ctx, id, ownerUserID, status)
}

func (r *MongoBloodRequestRepository) updateStatus(ctx context.Context, id, ownerUserID, status string) error {
	if status != models.StatusFulfilled {
		return ErrInvalidBloodRequestStatus
	}
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidBloodRequestID
	}

	requestFilter := bson.M{"_id": objID, "status": models.StatusOpen}
	if ownerUserID != "" {
		requestFilter["owner_user_id"] = ownerUserID
	}
	result, err := r.collection.UpdateOne(ctx,
		requestFilter,
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now().UTC()}},
	)
	if err != nil {
		return fmt.Errorf("failed to update blood request status: %w", err)
	}
	if result.MatchedCount == 0 {
		return r.ensureOwnedRequestIsOpen(ctx, objID, ownerUserID)
	}
	return nil
}

// DeleteOwned deletes an owner's open or fulfilled blood request.
func (r *MongoBloodRequestRepository) DeleteOwned(ctx context.Context, id, ownerUserID string) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidBloodRequestID
	}
	result, err := r.collection.DeleteOne(ctx, bson.M{
		"_id":           objID,
		"owner_user_id": ownerUserID,
		"status":        bson.M{"$in": []string{models.StatusOpen, models.StatusFulfilled}},
	})
	if err != nil {
		return fmt.Errorf("failed to delete blood request: %w", err)
	}
	if result.DeletedCount > 0 {
		return nil
	}

	var req models.BloodRequest
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&req)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrBloodRequestNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check blood request ownership: %w", err)
	}
	if req.OwnerUserID != ownerUserID {
		return ErrBloodRequestForbidden
	}
	return ErrBloodRequestNotDeletable
}

func (r *MongoBloodRequestRepository) ensureRequestIsOpen(ctx context.Context, id bson.ObjectID) error {
	return r.ensureOwnedRequestIsOpen(ctx, id, "")
}

func (r *MongoBloodRequestRepository) ensureOwnedRequestIsOpen(ctx context.Context, id bson.ObjectID, ownerUserID string) error {
	var req models.BloodRequest
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&req)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrBloodRequestNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check blood request status: %w", err)
	}
	if ownerUserID != "" && req.OwnerUserID != ownerUserID {
		return ErrBloodRequestForbidden
	}
	return ErrBloodRequestNotOpen
}
