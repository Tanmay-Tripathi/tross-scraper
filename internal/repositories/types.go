package repositories

import (
	"errors"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// ErrDatabaseNotConfigured means no Postgres store was wired. The service keeps
// no relational state, so this is a normal deployment, not a failure.
var ErrDatabaseNotConfigured = errors.New("database not configured")

// RepositoryAccess is the shared dependency set every repository receives.
// Db is optional and nil unless a Postgres store is wired in cmd/app.
type RepositoryAccess struct {
	Db     *db.Store
	Cache  db.CacheStoreMethods
	Logger log.Logger
}
