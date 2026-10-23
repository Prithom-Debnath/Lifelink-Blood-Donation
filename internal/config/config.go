package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration values read from environment variables.
type Config struct {
	Port         string
	AppEnv       string
	MongoURI     string
	MongoDBName  string
	JWTSecret    string
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

// Load reads configuration from the environment and optional .env file.
// If an environment variable is missing, sensible fallback defaults are provided.
func Load() *Config {
	// Attempt to load .env file if it exists.
	// We do not fail if .env is missing because variables might be set directly in the OS environment.
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found or could not be loaded; reading from system environment variables.")
	}

	return &Config{
		Port:         getEnv("PORT", "8080"),
		AppEnv:       getEnv("APP_ENV", "development"),
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName:  getEnv("MONGO_DB_NAME", "lifelink_db"),
		JWTSecret:    getEnv("JWT_SECRET", "default_jwt_secret_lifelink"),
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
	}
}

// getEnv retrieves the value of an environment variable or returns the default fallback.
func getEnv(key string, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
