package services

import (
	"context"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
)

// ServiceHealthMethods reports the service's own readiness.
type ServiceHealthMethods interface {
	// Check probes every dependency and returns the aggregate status. A
	// degraded dependency is reported in the status, not as an error — the
	// controller decides what HTTP code that deserves.
	Check(ctx context.Context) models.HealthStatus
}

type ServiceHealth struct {
	Access *ServiceAccess
}

func NewServiceHealth(access *ServiceAccess) ServiceHealthMethods {
	return &ServiceHealth{Access: access}
}

func (s *ServiceHealth) Check(ctx context.Context) models.HealthStatus {
	logger := s.Access.Logger.With(ctx)
	repo := s.Access.Repositories.Health

	status := models.HealthStatus{
		Service:     s.Access.Cfg.AppName,
		Version:     s.Access.Cfg.AppVersion,
		Environment: s.Access.Cfg.Environment,
		Database:    models.Up(),
		Cache:       models.Up(),
	}

	if err := repo.PingDatabase(ctx); err != nil {
		logger.Errorf("health check: database unreachable: %v", err)
		status.Database = models.Down(err)
	}

	if err := repo.PingCache(ctx); err != nil {
		logger.Errorf("health check: cache unreachable: %v", err)
		status.Cache = models.Down(err)
	}

	return status
}
