package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strings"

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

	// Default urgency to "normal" if omitted
	if strings.TrimSpace(req.Urgency) == "" {
		req.Urgency = models.UrgencyNormal
	} else {
		req.Urgency = models.NormalizeUrgency(req.Urgency)
	}

	// Validate required fields, supported blood group, allowed status, and valid coordinates
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
// Supports optional filtering by urgency (?urgency=critical) and sorting (?sort=urgency).
func (h *BloodRequestHandler) ListBloodRequests(c *gin.Context) {
	requests, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve blood requests",
		})
		return
	}

	// Filter by urgency if requested
	urgencyFilter := strings.ToLower(strings.TrimSpace(c.Query("urgency")))
	if urgencyFilter != "" && models.IsValidUrgency(urgencyFilter) {
		filtered := make([]models.BloodRequest, 0)
		for _, r := range requests {
			if strings.EqualFold(r.Urgency, urgencyFilter) {
				filtered = append(filtered, r)
			}
		}
		requests = filtered
	}

	// Sort by urgency if requested (Critical > Urgent > Normal)
	sortParam := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	if sortParam == "" {
		sortParam = strings.ToLower(strings.TrimSpace(c.Query("sortBy")))
	}

	if sortParam == "urgency" {
		urgencyWeight := func(u string) int {
			switch strings.ToLower(strings.TrimSpace(u)) {
			case models.UrgencyCritical:
				return 3
			case models.UrgencyUrgent:
				return 2
			default:
				return 1
			}
		}

		sort.SliceStable(requests, func(i, j int) bool {
			wi := urgencyWeight(requests[i].Urgency)
			wj := urgencyWeight(requests[j].Urgency)
			if wi != wj {
				return wi > wj // higher urgency first
			}
			return requests[i].CreatedAt.After(requests[j].CreatedAt)
		})
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
