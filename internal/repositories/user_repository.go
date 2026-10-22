package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"lifelink/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var (
	// ErrUserNotFound is returned when a requested user account does not exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when trying to register an email already in use.
	ErrUserAlreadyExists = errors.New("user with this email already exists")

	// ErrInvalidUserID is returned when an ID cannot be parsed as a valid MongoDB ObjectID.
	ErrInvalidUserID = errors.New("invalid user id format")
)

// UserRepository defines the data access contract for user accounts.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
}

// MongoUserRepository implements UserRepository using MongoDB.
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a new MongoUserRepository for the "users" collection.
func NewUserRepository(db *mongo.Database) *MongoUserRepository {
	var coll *mongo.Collection
	if db != nil {
		coll = db.Collection("users")
	}
	return &MongoUserRepository{
		collection: coll,
	}
}

// Create inserts a new user record into MongoDB after checking for duplicate email.
func (r *MongoUserRepository) Create(ctx context.Context, user *models.User) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(user.Email))

	// Check if a user with this email already exists
	var existing models.User
	err := r.collection.FindOne(ctx, bson.M{"email": normalizedEmail}).Decode(&existing)
	if err == nil {
		return ErrUserAlreadyExists
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if user.ID.IsZero() {
		user.ID = bson.NewObjectID()
	}

	user.Email = normalizedEmail
	user.Name = strings.TrimSpace(user.Name)

	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	_, err = r.collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to insert user into database: %w", err)
	}

	return nil
}

// FindByEmail searches for a user record matching the given email (case-insensitive).
func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": normalizedEmail}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return &user, nil
}

// FindByID retrieves a user record by their hex ObjectID.
func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	var user models.User
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	return &user, nil
}
