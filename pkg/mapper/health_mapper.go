// Package mapper converts between transport DTOs (internal/requests,
// internal/response) and domain models (internal/models). Controllers own the
// conversion; services only ever see models.
package mapper

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/response"
)

const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
)

// ToHealthResponse converts the aggregate health model into its API shape,
// dropping the operator-only error details on each component.
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
		},
	}
}
