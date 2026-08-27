package services

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/clients"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/repositories"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// Services aggregates every use-case service the controllers can call.
type Services struct {
	Health ServiceHealthMethods
}

// NewServices wires the business-logic layer.
func NewServices(
	cfg *config.Config,
	store *db.Store,
	repos *repositories.Repositories,
	cache db.CacheStoreMethods,
	logger log.Logger,
	clients *clients.Clients,
) *Services {
	access := &ServiceAccess{
		Cfg:          cfg,
		Db:           store,
		Cache:        cache,
		Logger:       logger,
		Repositories: repos,
		Clients:      clients,
	}

	return &Services{
		Health: NewServiceHealth(access),
	}
}
