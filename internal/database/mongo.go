package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"lifelink/internal/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// DB encapsulates the MongoDB client and database reference.
type DB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// Connect initializes the MongoDB client, sets a timeout, and verifies reachability with Ping.
// It fails clearly with an informative error if the database server cannot be reached.
func Connect(cfg *config.Config) (*DB, error) {
	log.Printf("Connecting to MongoDB at %s...", cfg.MongoURI)

	// Set a 5-second server selection timeout to avoid hanging indefinitely if MongoDB is offline
	clientOptions := options.Client().
		ApplyURI(cfg.MongoURI).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MongoDB client: %w", err)
	}

	// Ping the MongoDB server using a 5-second deadline context
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		// Clean up the client resources if the ping fails
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping MongoDB at %s: %w", cfg.MongoURI, err)
	}

	log.Printf("Successfully connected to MongoDB database '%s' at %s", cfg.MongoDBName, cfg.MongoURI)

	return &DB{
		Client:   client,
		Database: client.Database(cfg.MongoDBName),
	}, nil
}

// Close gracefully closes the MongoDB connection using a timeout context.
func (db *DB) Close() error {
	if db == nil || db.Client == nil {
		return nil
	}

	log.Println("Closing MongoDB connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to cleanly disconnect MongoDB: %w", err)
	}

	log.Println("MongoDB connection closed cleanly.")
	return nil
}
