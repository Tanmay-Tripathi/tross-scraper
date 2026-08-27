package clients

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// clientAccess is the shared dependency set every client receives. It is
// unexported because clients are only ever built through NewClients.
type clientAccess struct {
	cfg    *config.Config
	logger log.Logger
	cache  db.CacheStoreMethods
}
