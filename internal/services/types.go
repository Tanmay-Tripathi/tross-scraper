package services

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/clients"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/repositories"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// ServiceAccess is the shared dependency set every service receives.
type ServiceAccess struct {
	Cfg          *config.Config
	Db           *db.Store
	Cache        db.CacheStoreMethods
	Logger       log.Logger
	Clients      *clients.Clients
	Repositories *repositories.Repositories
}
