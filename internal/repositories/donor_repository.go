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
	// ErrNotFound is returned when a requested record does not exist in the database.
	ErrNotFound = errors.New("donor not found")

	// ErrInvalidID is returned when an ID cannot be parsed as a valid MongoDB ObjectID hex string.
	ErrInvalidID = errors.New("invalid donor id format")

	// ErrDuplicateDonationDate is returned when a donation is already recorded for that date.
	ErrDuplicateDonationDate = errors.New("donation is already recorded for this date")
	ErrDonorForbidden        = errors.New("donor is owned by another account")
)

// OwnedDonorRepository supports owner-checked donor mutations.
type OwnedDonorRepository interface {
	RecordDonationOwned(ctx context.Context, id, ownerUserID string, donation models.DonationRecord) error
}

// DonorRepository defines data access methods for donor records.
type DonorRepository interface {
	Create(ctx context.Context, donor *models.Donor) error
	List(ctx context.Context) ([]models.Donor, error)
	FindByID(ctx context.Context, id string) (*models.Donor, error)
	Update(ctx context.Context, id string, donor *models.Donor) error
	RecordDonation(ctx context.Context, id string, donation models.DonationRecord) error
	Delete(ctx context.Context, id string) error
}

// MongoDonorRepository implements DonorRepository using MongoDB.
type MongoDonorRepository struct {
	collection *mongo.Collection
}

// NewDonorRepository creates a new MongoDonorRepository for the "donors" collection.
func NewDonorRepository(db *mongo.Database) *MongoDonorRepository {
	var coll *mongo.Collection
	if db != nil {
		coll = db.Collection("donors")
	}
	return &MongoDonorRepository{
		collection: coll,
	}
}

// Create inserts a new donor into the database, generating an ObjectID and timestamps if not provided.
func (r *MongoDonorRepository) Create(ctx context.Context, donor *models.Donor) error {
	if donor.ID.IsZero() {
		donor.ID = bson.NewObjectID()
	}

	now := time.Now().UTC()
	if donor.CreatedAt.IsZero() {
		donor.CreatedAt = now
	}
	donor.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, donor)
	if err != nil {
		return fmt.Errorf("failed to insert donor into database: %w", err)
	}

	return nil
}

// List retrieves all donors from the database.
func (r *MongoDonorRepository) List(ctx context.Context) ([]models.Donor, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to query donors: %w", err)
	}
	defer cursor.Close(ctx)

	// Initialize as an empty slice so it serializes as [] in JSON instead of null
	donors := make([]models.Donor, 0)
	if err := cursor.All(ctx, &donors); err != nil {
		return nil, fmt.Errorf("failed to decode donors cursor: %w", err)
	}

	return donors, nil
}

// FindByID retrieves a single donor by their hex ObjectID.
// Returns ErrInvalidID if the ID format is invalid, or ErrNotFound if no record matches.
func (r *MongoDonorRepository) FindByID(ctx context.Context, id string) (*models.Donor, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	var donor models.Donor
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&donor)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to find donor by ID: %w", err)
	}

	return &donor, nil
}

// Update modifies an existing donor record's fields and updates the updated_at timestamp.
// Returns ErrInvalidID if the ID format is invalid, or ErrNotFound if no record was updated.
func (r *MongoDonorRepository) Update(ctx context.Context, id string, donor *models.Donor) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	donor.UpdatedAt = time.Now().UTC()

	update := bson.M{
		"$set": bson.M{
			"name":         donor.Name,
			"email":        donor.Email,
			"phone":        donor.Phone,
			"blood_group":  donor.BloodGroup,
			"city":         donor.City,
			"availability": donor.Availability,
			"latitude":     donor.Latitude,
			"longitude":    donor.Longitude,
			"updated_at":   donor.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return fmt.Errorf("failed to execute donor update: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

// RecordDonation appends a donor-reported donation date without replacing earlier history.
func (r *MongoDonorRepository) RecordDonation(ctx context.Context, id string, donation models.DonationRecord) error {
	return r.recordDonation(ctx, id, "", donation)
}

func (r *MongoDonorRepository) RecordDonationOwned(ctx context.Context, id, ownerUserID string, donation models.DonationRecord) error {
	return r.recordDonation(ctx, id, ownerUserID, donation)
}

func (r *MongoDonorRepository) recordDonation(ctx context.Context, id, ownerUserID string, donation models.DonationRecord) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	donation.DonationDate = donation.DonationDate.UTC()
	donation.RecordedAt = time.Now().UTC()
	filter := bson.M{"_id": objID, "donation_history.donation_date": bson.M{"$ne": donation.DonationDate}}
	if ownerUserID != "" {
		filter["owner_user_id"] = ownerUserID
	}
	result, err := r.collection.UpdateOne(ctx,
		filter,
		bson.M{
			"$push": bson.M{"donation_history": donation},
			"$set":  bson.M{"updated_at": donation.RecordedAt},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to record donor donation: %w", err)
	}
	if result.MatchedCount > 0 {
		return nil
	}

	donor, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if ownerUserID != "" && donor.OwnerUserID != ownerUserID {
		return ErrDonorForbidden
	}
	for _, existing := range donor.DonationHistory {
		if existing.DonationDate.Equal(donation.DonationDate) {
			return ErrDuplicateDonationDate
		}
	}
	return ErrNotFound
}

// Delete removes a donor record by their hex ObjectID.
// Returns ErrInvalidID if the ID format is invalid, or ErrNotFound if no record was deleted.
func (r *MongoDonorRepository) Delete(ctx context.Context, id string) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return fmt.Errorf("failed to execute donor delete: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrNotFound
	}

	return nil
}
