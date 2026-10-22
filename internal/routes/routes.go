package routes

import (
	"os"

	"lifelink/internal/config"
	"lifelink/internal/handlers"
	"lifelink/internal/middleware"
	"lifelink/internal/repositories"

	"github.com/gin-gonic/gin"
)

// SetupRouter initializes the Gin engine, attaches middleware, and registers all API routes.
func SetupRouter(
	cfg *config.Config,
	donorRepo repositories.DonorRepository,
	requestRepo repositories.BloodRequestRepository,
	userRepo repositories.UserRepository,
) *gin.Engine {
	// Set Gin mode based on the environment configuration
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// gin.Default() comes with standard Logger and Recovery middleware
	router := gin.Default()

	// Locate and serve web frontend assets
	webDir := ""
	if _, err := os.Stat("web/index.html"); err == nil {
		webDir = "web"
	} else if _, err := os.Stat("../../web/index.html"); err == nil {
		webDir = "../../web"
	}

	if webDir != "" {
		router.StaticFile("/", webDir+"/index.html")
		router.Static("/static", webDir)
	}

	// API route group with consistent /api prefix
	api := router.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)

		// Register user authentication and profile endpoints if a repository is provided
		if userRepo != nil {
			authHandler := handlers.NewAuthHandler(userRepo, cfg)
			auth := api.Group("/auth")
			{
				auth.POST("/register", authHandler.Register)
				auth.POST("/login", authHandler.Login)
				auth.GET("/me", middleware.AuthMiddleware(cfg.JWTSecret), authHandler.GetMe)
			}
		}

		// Register donor endpoints if a repository is provided
		if donorRepo != nil {
			donorHandler := handlers.NewDonorHandler(donorRepo)
			api.POST("/donors", donorHandler.CreateDonor)
			api.GET("/donors", donorHandler.ListDonors)
			api.GET("/donors/:id", donorHandler.GetDonorByID)
			api.PUT("/donors/:id", donorHandler.UpdateDonor)
			api.DELETE("/donors/:id", donorHandler.DeleteDonor)

			// Register donor matching endpoint
			matchHandler := handlers.NewMatchHandler(donorRepo)
			api.GET("/matches", matchHandler.FindMatches)
		}

		// Register blood request endpoints if a repository is provided
		if requestRepo != nil {
			requestHandler := handlers.NewBloodRequestHandler(requestRepo)
			api.POST("/requests", requestHandler.CreateBloodRequest)
			api.GET("/requests", requestHandler.ListBloodRequests)
			api.GET("/requests/:id", requestHandler.GetBloodRequestByID)
		}
	}

	return router
}
