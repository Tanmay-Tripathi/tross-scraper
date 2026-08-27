package repositories

import (
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// RepositoryAccess is the shared dependency set every repository receives.
type RepositoryAccess struct {
	Db     *db.Store
	Cache  db.CacheStoreMethods
	Logger log.Logger
}
