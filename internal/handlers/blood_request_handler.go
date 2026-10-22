package handlers

import (
	"errors"
	"net/http"

	"lifelink/internal/models"
	"lifelink/internal/repositories"

	"github.com/gin-gonic/gin"
)

// BloodRequestHandler defines HTTP handlers for blood request operations.
type BloodRequestHandler struct {
	repo repositories.BloodRequestRepository
}

// NewBloodRequestHandler creates a new BloodRequestHandler instance with the given repository.
func NewBloodRequestHandler(repo repositories.BloodRequestRepository) *BloodRequestHandler {
	return &BloodRequestHandler{
		repo: repo,
	}
}

// CreateBloodRequest handles POST /api/requests to create a new patient blood request.
func (h *BloodRequestHandler) CreateBloodRequest(c *gin.Context) {
	var req models.BloodRequest

	// Bind request JSON payload to the BloodRequest struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload: " + err.Error(),
		})
		return
	}

	// Validate required fields, supported blood group, and allowed status
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Persist the blood request record using the repository
	if err := h.repo.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create blood request",
		})
		return
	}

	c.JSON(http.StatusCreated, req)
}

// CreateRequest is an alias for CreateBloodRequest.
func (h *BloodRequestHandler) CreateRequest(c *gin.Context) {
	h.CreateBloodRequest(c)
}

// ListBloodRequests handles GET /api/requests to retrieve all blood requests.
func (h *BloodRequestHandler) ListBloodRequests(c *gin.Context) {
	requests, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve blood requests",
		})
		return
	}

	c.JSON(http.StatusOK, requests)
}

// ListRequests is an alias for ListBloodRequests.
func (h *BloodRequestHandler) ListRequests(c *gin.Context) {
	h.ListBloodRequests(c)
}

// GetBloodRequestByID handles GET /api/requests/:id to retrieve a specific blood request by ObjectID.
func (h *BloodRequestHandler) GetBloodRequestByID(c *gin.Context) {
	id := c.Param("id")

	req, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrInvalidBloodRequestID) || errors.Is(err, repositories.ErrInvalidID) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid blood request ID format",
			})
			return
		}
		if errors.Is(err, repositories.ErrBloodRequestNotFound) || errors.Is(err, repositories.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Blood request not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve blood request",
		})
		return
	}

	c.JSON(http.StatusOK, req)
}

// GetRequestByID is an alias for GetBloodRequestByID.
func (h *BloodRequestHandler) GetRequestByID(c *gin.Context) {
	h.GetBloodRequestByID(c)
}
