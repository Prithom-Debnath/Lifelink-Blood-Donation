package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"lifelink/internal/models"
	"lifelink/internal/repositories"

	"github.com/gin-gonic/gin"
)

// DonorHandler defines HTTP handlers for donor CRUD operations.
type DonorHandler struct {
	repo repositories.DonorRepository
}

// NewDonorHandler creates a new DonorHandler instance with the given repository.
func NewDonorHandler(repo repositories.DonorRepository) *DonorHandler {
	return &DonorHandler{
		repo: repo,
	}
}

// CreateDonor handles POST /api/donors to register a new blood donor.
func (h *DonorHandler) CreateDonor(c *gin.Context) {
	var donor models.Donor

	// Bind request JSON payload to the Donor struct
	if err := c.ShouldBindJSON(&donor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload: " + err.Error(),
		})
		return
	}
	if owner, ok := c.Get("userID"); ok {
		donor.OwnerUserID, _ = owner.(string)
	}

	// Validate required fields, email format, and supported blood group
	if err := donor.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Persist the donor record using the repository
	if err := h.repo.Create(c.Request.Context(), &donor); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create donor record",
		})
		return
	}

	c.JSON(http.StatusCreated, donor)
}

// ListDonors handles GET /api/donors to retrieve all registered donors.
func (h *DonorHandler) ListDonors(c *gin.Context) {
	donors, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve donors",
		})
		return
	}

	c.JSON(http.StatusOK, donors)
}

// GetDonorByID handles GET /api/donors/:id to retrieve a specific donor by their ObjectID.
func (h *DonorHandler) GetDonorByID(c *gin.Context) {
	id := c.Param("id")

	donor, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrInvalidID) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid donor ID format",
			})
			return
		}
		if errors.Is(err, repositories.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Donor not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve donor",
		})
		return
	}

	c.JSON(http.StatusOK, donor)
}

// UpdateDonor handles PUT /api/donors/:id to update an existing donor record.
func (h *DonorHandler) UpdateDonor(c *gin.Context) {
	id := c.Param("id")

	var donor models.Donor
	if err := c.ShouldBindJSON(&donor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload: " + err.Error(),
		})
		return
	}

	// Validate required fields, email format, and supported blood group
	if err := donor.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.repo.Update(c.Request.Context(), id, &donor); err != nil {
		if errors.Is(err, repositories.ErrInvalidID) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid donor ID format",
			})
			return
		}
		if errors.Is(err, repositories.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Donor not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update donor",
		})
		return
	}

	// Fetch and return the updated document
	updated, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Donor updated successfully",
		})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// RecordDonation handles POST /api/donors/:id/donations.
func (h *DonorHandler) RecordDonation(c *gin.Context) {
	var payload struct {
		DonationDate string `json:"donation_date"`
		Location     string `json:"location"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	dateText := strings.TrimSpace(payload.DonationDate)
	donationDate, err := time.Parse("2006-01-02", dateText)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "donation_date must use YYYY-MM-DD format"})
		return
	}
	bdTime := time.Now().In(time.FixedZone("Asia/Dhaka", 6*60*60))
	today := bdTime.Format("2006-01-02")
	if dateText > today {
		c.JSON(http.StatusBadRequest, gin.H{"error": "donation_date cannot be in the future"})
		return
	}

	donationDate = donationDate.UTC()
	donation := models.DonationRecord{
		DonationDate: donationDate,
		Location:     strings.TrimSpace(payload.Location),
	}
	ownerID, _ := c.Get("userID")
	ownedRepo, ok := h.repo.(repositories.OwnedDonorRepository)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ownership checks are unavailable"})
		return
	}
	if err := ownedRepo.RecordDonationOwned(c.Request.Context(), c.Param("id"), ownerID.(string), donation); err != nil {
		switch {
		case errors.Is(err, repositories.ErrInvalidID):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid donor ID format"})
		case errors.Is(err, repositories.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Donor not found"})
		case errors.Is(err, repositories.ErrDonorForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "Only the account that registered this donor can record donations"})
		case errors.Is(err, repositories.ErrDuplicateDonationDate):
			c.JSON(http.StatusConflict, gin.H{"error": "A donation is already recorded for this date"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record donation"})
		}
		return
	}

	nextReminder := donationDate.AddDate(0, 0, models.DonationReminderIntervalDays)
	c.JSON(http.StatusCreated, gin.H{
		"donation_date":      dateText,
		"next_reminder_date": nextReminder.Format("2006-01-02"),
	})
}

// DeleteDonor handles DELETE /api/donors/:id to delete an existing donor record.
func (h *DonorHandler) DeleteDonor(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repositories.ErrInvalidID) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid donor ID format",
			})
			return
		}
		if errors.Is(err, repositories.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Donor not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete donor",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Donor deleted successfully",
	})
}
