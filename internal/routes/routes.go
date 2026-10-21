package routes

import (
	"lifelink/internal/config"
	"lifelink/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the Gin engine, attaches middleware, and registers all API routes.
func SetupRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode based on the environment configuration
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// gin.Default() comes with standard Logger and Recovery middleware
	router := gin.Default()

	// API route group with consistent /api prefix
	api := router.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)
	}

	return router
}
