// Package clients holds the service's outbound integrations. Services depend on
// the interfaces declared here, never on an SDK or an HTTP call directly.
package clients

import (
	"context"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/network"
)

// Clients aggregates every external integration.
type Clients struct {
	// Sqs is nil when messaging is disabled in the configuration, or when the
	// AWS client could not be built. Callers must check it before use.
	Sqs ClientSqsMethods
}

// NewClients wires the integration layer. A client whose dependency is
// unreachable is logged and left nil rather than aborting startup, so a
// degraded downstream cannot take the API down with it.
func NewClients(
	ctx context.Context,
	cfg *config.Config,
	logger log.Logger,
	cache db.CacheStoreMethods,
	networkOps network.NetworkOpsMethods,
) *Clients {
	access := &clientAccess{
		cfg:        cfg,
		logger:     logger,
		cache:      cache,
		networkOps: networkOps,
	}

	sqsClient, err := NewClientSqs(ctx, access)
	if err != nil {
		logger.Errorf("failed to initialize sqs client, messaging is unavailable: %v", err)
	}

	return &Clients{
		Sqs: sqsClient,
	}
}

// Close releases every client that holds a long-lived resource.
func (c *Clients) Close() {
	if c.Sqs != nil {
		c.Sqs.Close()
	}
}
