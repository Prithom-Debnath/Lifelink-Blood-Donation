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
)

// BloodRequestRepository defines data access methods for patient blood requests.
type BloodRequestRepository interface {
	Create(ctx context.Context, req *models.BloodRequest) error
	List(ctx context.Context) ([]models.BloodRequest, error)
	FindByID(ctx context.Context, id string) (*models.BloodRequest, error)
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
