package handlers

import (
	"math"
	"net/http"
	"sort"
	"strconv"
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

// FindMatches handles GET /api/matches?bloodGroup=B+&city=Dhaka&lat=23.81&lng=90.41&radius=25.
// It retrieves available donors whose blood group is compatible with the requested recipient blood group.
// Supports geospatial radius filtering (in kilometers) when coordinates are provided,
// and falls back gracefully to city-based filtering when coordinates are omitted.
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

	// Optional location and search radius parameters
	city := strings.TrimSpace(c.Query("city"))

	latStr := c.Query("lat")
	if latStr == "" {
		latStr = c.Query("latitude")
	}
	lngStr := c.Query("lng")
	if lngStr == "" {
		lngStr = c.Query("longitude")
	}

	var hasCoords bool
	var searchLat, searchLng float64

	if latStr != "" && lngStr != "" {
		var errLat, errLng error
		searchLat, errLat = strconv.ParseFloat(strings.TrimSpace(latStr), 64)
		searchLng, errLng = strconv.ParseFloat(strings.TrimSpace(lngStr), 64)
		if errLat != nil || errLng != nil || !models.IsValidCoordinates(&searchLat, &searchLng) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid coordinates: lat must be between -90 and 90, lng between -180 and 180",
			})
			return
		}
		hasCoords = true
	}

	// Default radius is 25 km if coordinates are supplied
	radiusKm := 25.0
	radiusStr := c.Query("radius")
	if radiusStr == "" {
		radiusStr = c.Query("radius_km")
	}
	if radiusStr == "" {
		radiusStr = c.Query("radiusKm")
	}
	if radiusStr != "" {
		parsedRadius, err := strconv.ParseFloat(strings.TrimSpace(radiusStr), 64)
		if err != nil || math.IsNaN(parsedRadius) || math.IsInf(parsedRadius, 0) || parsedRadius < 0 || parsedRadius > 50 || math.Mod(parsedRadius, 5) != 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "radius must be between 0 and 50 km in steps of 5 km",
			})
			return
		}
		radiusKm = parsedRadius
	}

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

		if hasCoords {
			// When search coordinates are provided, filter by distance
			if donor.Latitude != nil && donor.Longitude != nil {
				dist := models.HaversineDistanceKm(searchLat, searchLng, *donor.Latitude, *donor.Longitude)
				if dist <= radiusKm {
					donorCopy := donor
					rounded := math.Round(dist*100) / 100 // round to 2 decimals
					donorCopy.DistanceKm = &rounded
					matches = append(matches, donorCopy)
				}
			} else if city != "" && strings.Contains(strings.ToLower(strings.TrimSpace(donor.City)), strings.ToLower(city)) {
				// Donors without coordinates in the same city are included as fallback
				matches = append(matches, donor)
			}
		} else {
			// Fallback: If no coordinates provided, use city filter if present
			if city != "" && !strings.Contains(strings.ToLower(strings.TrimSpace(donor.City)), strings.ToLower(city)) {
				continue
			}
			matches = append(matches, donor)
		}
	}

	// If coordinates were searched, sort matches by distance (closest first)
	if hasCoords {
		sort.SliceStable(matches, func(i, j int) bool {
			if matches[i].DistanceKm == nil {
				return false
			}
			if matches[j].DistanceKm == nil {
				return true
			}
			return *matches[i].DistanceKm < *matches[j].DistanceKm
		})
	}

	c.JSON(http.StatusOK, matches)
}

// GetMatches is an alias for FindMatches.
func (h *MatchHandler) GetMatches(c *gin.Context) {
	h.FindMatches(c)
}
