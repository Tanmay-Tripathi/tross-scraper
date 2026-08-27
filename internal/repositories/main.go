package repositories

import (
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// Repositories aggregates every repository the service exposes.
type Repositories struct {
	Health RepositoryHealthMethods
}

// NewRepositories wires the persistence layer.
func NewRepositories(store *db.Store, cache db.CacheStoreMethods, logger log.Logger) *Repositories {
	access := &RepositoryAccess{
		Db:     store,
		Cache:  cache,
		Logger: logger,
	}

	return &Repositories{
		Health: NewRepositoryHealth(access),
	}
}
