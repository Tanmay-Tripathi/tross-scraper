// Package mapper converts between transport DTOs and domain models. Controllers
// own the conversion; services only see models.
package mapper

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/response"
)

const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
)

// ToHealthResponse converts health into its API shape, dropping operator-only detail.
func ToHealthResponse(status models.HealthStatus) response.HealthResponse {
	overall := statusHealthy
	if !status.IsHealthy() {
		overall = statusUnhealthy
	}

	return response.HealthResponse{
		Status:      overall,
		Service:     status.Service,
		Version:     status.Version,
		Environment: status.Environment,
		Dependencies: map[string]response.ComponentResponse{
			"database": {Status: string(status.Database.State)},
			"cache":    {Status: string(status.Cache.State)},
			"linkedin": {Status: string(status.LinkedIn.State)},
		},
	}
}
