package handlers

import (
	"net/http"
	"strings"

	"lifelink/internal/models"
	"lifelink/internal/repositories"

	"github.com/gin-gonic/gin"
)

// MatchHandler defines HTTP handlers for donor matching and blood compatibility operations.
type MatchHandler struct {
	donorRepo repositories.DonorRepository
}

// NewMatchHandler creates a new MatchHandler instance with the given donor repository.
func NewMatchHandler(donorRepo repositories.DonorRepository) *MatchHandler {
	return &MatchHandler{
		donorRepo: donorRepo,
	}
}

// FindMatches handles GET /api/matches?bloodGroup=B+&city=Dhaka.
// It retrieves available donors whose blood group is compatible with the requested recipient blood group.
// An optional city filter is supported.
//
// DISCLAIMER: This is a simplified educational matching feature for the CSE 250 course.
// It does NOT constitute medical guidance or clinical advice for actual blood transfusions.
func (h *MatchHandler) FindMatches(c *gin.Context) {
	bloodGroup := c.Query("bloodGroup")
	if bloodGroup == "" {
		bloodGroup = c.Query("blood_group")
	}

	// In URL query strings, an unencoded '+' is often decoded by HTTP parsers as a space ' '.
	// We handle this gracefully so both ?bloodGroup=B+ and ?bloodGroup=B%2B work seamlessly.
	if strings.HasSuffix(bloodGroup, " ") {
		prefix := strings.TrimSpace(bloodGroup)
		switch prefix {
		case "A", "B", "AB", "O":
			bloodGroup = prefix + "+"
		}
	}
	bloodGroup = strings.TrimSpace(bloodGroup)

	if bloodGroup == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bloodGroup query parameter is required",
		})
		return
	}

	if !models.IsValidBloodGroup(bloodGroup) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-",
		})
		return
	}

	city := strings.TrimSpace(c.Query("city"))

	donors, err := h.donorRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve donors for matching",
		})
		return
	}

	matches := make([]models.Donor, 0)
	for _, donor := range donors {
		// Only consider donors who are currently marked as available
		if !donor.Availability {
			continue
		}

		// Check red blood cell compatibility between donor and recipient
		if !models.IsBloodCompatible(donor.BloodGroup, bloodGroup) {
			continue
		}

		// If a city filter is provided, perform case-insensitive comparison
		if city != "" && !strings.EqualFold(strings.TrimSpace(donor.City), city) {
			continue
		}

		matches = append(matches, donor)
	}

	c.JSON(http.StatusOK, matches)
}

// GetMatches is an alias for FindMatches.
func (h *MatchHandler) GetMatches(c *gin.Context) {
	h.FindMatches(c)
}
