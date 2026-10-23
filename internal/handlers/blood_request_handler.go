package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"lifelink/internal/models"
	"lifelink/internal/notifications"
	"lifelink/internal/repositories"

	"github.com/gin-gonic/gin"
)

// BloodRequestHandler defines HTTP handlers for blood request operations.
type BloodRequestHandler struct {
	repo          repositories.BloodRequestRepository
	donorRepo     repositories.DonorRepository
	emailNotifier *notifications.EmailNotifier
}

// NewBloodRequestHandler creates a new BloodRequestHandler instance with the given repository.
func NewBloodRequestHandler(repo repositories.BloodRequestRepository, donorRepo repositories.DonorRepository, emailNotifier *notifications.EmailNotifier) *BloodRequestHandler {
	return &BloodRequestHandler{
		repo:          repo,
		donorRepo:     donorRepo,
		emailNotifier: emailNotifier,
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
	if owner, ok := c.Get("userID"); ok {
		req.OwnerUserID, _ = owner.(string)
	}
	req.Status = models.StatusOpen

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
	sent := h.notifyMatchingDonors(c.Request.Context(), &req)
	c.Header("X-Notifications-Sent", fmt.Sprint(sent))

	c.JSON(http.StatusCreated, req)
}

func (h *BloodRequestHandler) notifyMatchingDonors(parent context.Context, req *models.BloodRequest) int {
	if (req.Urgency != models.UrgencyUrgent && req.Urgency != models.UrgencyCritical) ||
		h.donorRepo == nil || h.emailNotifier == nil || !h.emailNotifier.Enabled() {
		return 0
	}

	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	donors, err := h.donorRepo.List(ctx)
	if err != nil {
		log.Printf("LifeLink could not load donors for request notifications: %v", err)
		return 0
	}

	location := strings.ToLower(strings.TrimSpace(req.City))
	recipients := make([]string, 0)
	for _, donor := range donors {
		if !donor.Availability || !models.IsBloodCompatible(donor.BloodGroup, req.BloodGroup) {
			continue
		}
		if !strings.Contains(strings.ToLower(strings.TrimSpace(donor.City)), location) {
			continue
		}
		recipients = append(recipients, donor.Email)
	}
	if len(recipients) == 0 {
		return 0
	}

	sent, err := h.emailNotifier.SendBloodRequestAlert(ctx, recipients, req)
	if err != nil {
		log.Printf("LifeLink could not send blood request email alert: %v", err)
		return 0
	}
	return sent
}

// CreateRequest is an alias for CreateBloodRequest.
func (h *BloodRequestHandler) CreateRequest(c *gin.Context) {
	h.CreateBloodRequest(c)
}

// ListBloodRequests handles GET /api/requests to retrieve all blood requests.
// Supports optional filtering by urgency/status and sorting (?sort=urgency).
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

	statusFilter := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if statusFilter != "" && statusFilter != "all" && models.IsValidStatus(statusFilter) {
		filtered := make([]models.BloodRequest, 0)
		for _, r := range requests {
			if strings.EqualFold(r.Status, statusFilter) {
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

// UpdateBloodRequest handles PUT /api/requests/:id for editing an open request.
func (h *BloodRequestHandler) UpdateBloodRequest(c *gin.Context) {
	var req models.BloodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Urgency) == "" {
		req.Urgency = models.UrgencyNormal
	} else {
		req.Urgency = models.NormalizeUrgency(req.Urgency)
	}
	req.Status = ""
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ownerID, _ := c.Get("userID")
	ownedRepo, ok := h.repo.(repositories.OwnedBloodRequestRepository)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ownership checks are unavailable"})
		return
	}
	if err := ownedRepo.UpdateOwned(c.Request.Context(), c.Param("id"), ownerID.(string), &req); err != nil {
		h.respondToRequestMutationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Blood request updated successfully"})
}

// UpdateBloodRequestStatus handles PATCH /api/requests/:id/status to fulfill an open request.
func (h *BloodRequestHandler) UpdateBloodRequestStatus(c *gin.Context) {
	var payload struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	payload.Status = strings.ToLower(strings.TrimSpace(payload.Status))
	if payload.Status != models.StatusFulfilled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be fulfilled"})
		return
	}

	ownerID, _ := c.Get("userID")
	ownedRepo, ok := h.repo.(repositories.OwnedBloodRequestRepository)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ownership checks are unavailable"})
		return
	}
	if err := ownedRepo.UpdateStatusOwned(c.Request.Context(), c.Param("id"), ownerID.(string), payload.Status); err != nil {
		h.respondToRequestMutationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Blood request marked as " + payload.Status})
}

// DeleteBloodRequest handles DELETE /api/requests/:id for the request owner.
func (h *BloodRequestHandler) DeleteBloodRequest(c *gin.Context) {
	ownerID, _ := c.Get("userID")
	ownedRepo, ok := h.repo.(repositories.OwnedBloodRequestRepository)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ownership checks are unavailable"})
		return
	}
	if err := ownedRepo.DeleteOwned(c.Request.Context(), c.Param("id"), ownerID.(string)); err != nil {
		switch {
		case errors.Is(err, repositories.ErrInvalidBloodRequestID):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blood request ID format"})
		case errors.Is(err, repositories.ErrBloodRequestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Blood request not found"})
		case errors.Is(err, repositories.ErrBloodRequestForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "Only the account that submitted this request can delete it"})
		case errors.Is(err, repositories.ErrBloodRequestNotDeletable):
			c.JSON(http.StatusConflict, gin.H{"error": "Only open or fulfilled requests can be deleted"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete blood request"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Blood request deleted successfully"})
}

func (h *BloodRequestHandler) respondToRequestMutationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repositories.ErrInvalidBloodRequestID):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blood request ID format"})
	case errors.Is(err, repositories.ErrBloodRequestNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Blood request not found"})
	case errors.Is(err, repositories.ErrBloodRequestNotOpen):
		c.JSON(http.StatusConflict, gin.H{"error": "Only open requests can be edited or marked fulfilled"})
	case errors.Is(err, repositories.ErrBloodRequestForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the account that submitted this request can edit or close it"})
	case errors.Is(err, repositories.ErrInvalidBloodRequestStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be open or fulfilled"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blood request"})
	}
}
