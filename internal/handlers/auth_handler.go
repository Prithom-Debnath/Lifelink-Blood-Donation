package handlers

import (
	"errors"
	"net/http"
	"time"

	"lifelink/internal/config"
	"lifelink/internal/middleware"
	"lifelink/internal/models"
	"lifelink/internal/repositories"

	"github.com/gin-gonic/gin"
)

// AuthHandler provides HTTP handlers for user authentication and profile management.
type AuthHandler struct {
	userRepo  repositories.UserRepository
	cfg       *config.Config
	tokenLife time.Duration
}

// NewAuthHandler creates a new AuthHandler with the provided repository and configuration.
func NewAuthHandler(userRepo repositories.UserRepository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		userRepo:  userRepo,
		cfg:       cfg,
		tokenLife: 24 * time.Hour, // Standard time-limited session duration
	}
}

// Register handles POST /api/auth/register to create a new user account.
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload: " + err.Error(),
		})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Securely hash plain-text password with bcrypt before storing
	hashedPassword, err := models.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to hash password",
		})
		return
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		if errors.Is(err, repositories.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "A user with this email already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user account",
		})
		return
	}

	// Sign a time-limited JWT login token using the configured JWT_SECRET
	token, err := middleware.GenerateToken(user.ID.Hex(), user.Email, h.cfg.JWTSecret, h.tokenLife)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Account created, but failed to generate authentication token",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"token":   token,
		"user":    user,
	})
}

// Login handles POST /api/auth/login to authenticate an existing user.
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload: " + err.Error(),
		})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.userRepo.FindByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify credentials",
		})
		return
	}

	// Verify password hash with bcrypt
	if !models.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	// Sign a time-limited JWT login token
	token, err := middleware.GenerateToken(user.ID.Hex(), user.Email, h.cfg.JWTSecret, h.tokenLife)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user":    user,
	})
}

// GetMe handles protected GET /api/auth/me to return the profile of the signed-in user.
func (h *AuthHandler) GetMe(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User authentication context missing",
		})
		return
	}

	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user identity",
		})
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) || errors.Is(err, repositories.ErrInvalidUserID) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User account not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user profile",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
