package handlers

import (
	"errors"
	"net/http"

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
