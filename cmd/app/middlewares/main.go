// Package middlewares holds the HTTP middleware chain. Middlewares may read
// config, cache and repositories, but must never contain business logic — that
// belongs in a service.
package middlewares

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/clients"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/repositories"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// Middlewares aggregates every middleware the router can apply.
type Middlewares struct {
	Cors        MiddlewareCorsMethods
	Idempotency MiddlewareIdempotencyMethods
}

// NewMiddlewares wires the middleware layer.
func NewMiddlewares(
	cfg *config.Config,
	store *db.Store,
	repos *repositories.Repositories,
	cache db.CacheStoreMethods,
	logger log.Logger,
	clients *clients.Clients,
) *Middlewares {
	access := &MiddlewareAccess{
		Cfg:          cfg,
		Db:           store,
		Cache:        cache,
		Logger:       logger,
		Repositories: repos,
		Clients:      clients,
	}

	return &Middlewares{
		Cors:        NewMiddlewareCors(access),
		Idempotency: NewMiddlewareIdempotency(access),
	}
}
