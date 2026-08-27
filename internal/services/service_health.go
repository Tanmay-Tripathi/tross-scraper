package services

import (
	"context"
	"errors"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/repositories"
)

// ServiceHealthMethods reports the service's own readiness.
type ServiceHealthMethods interface {
	// Check probes every dependency. A degraded one is reported in the status,
	// not as an error; the controller decides the HTTP code.
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
		LinkedIn:    models.Disabled(),
	}

	// Running without Postgres is deliberate, not a failure.
	switch err := repo.PingDatabase(ctx); {
	case errors.Is(err, repositories.ErrDatabaseNotConfigured):
		status.Database = models.Disabled()
	case err != nil:
		logger.Errorf("health check: database unreachable: %v", err)
		status.Database = models.Down(err)
	}

	if err := repo.PingCache(ctx); err != nil {
		logger.Errorf("health check: cache unreachable: %v", err)
		status.Cache = models.Down(err)
	}

	// Running without credentials is deliberate, not a failure.
	if client := s.Access.Clients.LinkedIn; client != nil {
		status.LinkedIn = models.Up()
		if err := client.SessionValid(ctx); err != nil {
			logger.Errorf("health check: linkedin session invalid: %v", err)
			status.LinkedIn = models.Down(err)
		}
	}

	return status
}
