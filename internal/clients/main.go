// Package clients holds the outbound integrations. Services depend on the
// interfaces here, never on an SDK directly. Add one as client_<name>.go.
package clients

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// Clients aggregates every external integration.
type Clients struct {
	// LinkedIn is nil when unconfigured or unbuildable; callers must check it.
	LinkedIn ClientLinkedInMethods
}

// NewClients wires the integration layer. An unreachable dependency is logged and
// left nil, so a missing session fails one endpoint rather than the whole boot.
func NewClients(cfg *config.Config, logger log.Logger, cache db.CacheStoreMethods) *Clients {
	access := &clientAccess{cfg: cfg, logger: logger, cache: cache}

	linkedIn, err := NewClientLinkedIn(access)
	if err != nil {
		logger.Errorf("linkedin client unavailable, profile scraping is disabled: %v", err)
	}

	return &Clients{LinkedIn: linkedIn}
}
