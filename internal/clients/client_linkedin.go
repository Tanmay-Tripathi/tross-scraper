package clients

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/linkedin/voyager"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/network"
)

// jitterMax bounds the pause before a live fetch; a regular rhythm is an easy tell.
const jitterMax = 400 * time.Millisecond

// ClientLinkedInMethods is the profile-fetching surface. It returns typed
// application errors, so no layer above reads an HTTP status from LinkedIn.
type ClientLinkedInMethods interface {
	// FetchProfile scrapes one profile by its public identifier.
	FetchProfile(ctx context.Context, publicID string) (*models.Profile, *exceptions.ApplicationError)
	// SessionValid reports whether the cookies still work; readiness uses it.
	SessionValid(ctx context.Context) error
}

type ClientLinkedIn struct {
	access  *clientAccess
	voyager *voyager.Client
}

// NewClientLinkedIn builds the Voyager client with one long-lived session.
func NewClientLinkedIn(access *clientAccess) (ClientLinkedInMethods, error) {
	cfg := access.cfg.LinkedIn
	if !cfg.Configured() {
		return nil, errors.New("no linkedin cookies configured (li_at and jsessionid)")
	}

	netOps, err := network.NewNetworkOps("linkedin", access.logger, network.Options{
		Timeout:         time.Duration(cfg.RequestTimeoutSeconds) * time.Second,
		EnableCookieJar: true,
		NoRedirects:     true,
	})
	if err != nil {
		return nil, err
	}

	client, err := voyager.NewClient(netOps, voyager.Credentials{
		LiAt:       cfg.LiAt,
		JSessionID: cfg.JSessionID,
		UserAgent:  cfg.UserAgent,
	}, access.logger)
	if err != nil {
		return nil, err
	}

	access.logger.Infof("linkedin client ready")
	return &ClientLinkedIn{access: access, voyager: client}, nil
}

func (c *ClientLinkedIn) FetchProfile(ctx context.Context, publicID string) (*models.Profile, *exceptions.ApplicationError) {
	logger := c.access.logger.With(ctx)

	c.jitter()

	bodies, failures := c.voyager.FetchAll(ctx, publicID)

	// Only the essential call failing stops us; a partial profile is still useful.
	// The name comes from the endpoint list rather than a literal, so retiring an
	// endpoint cannot leave this check pointing at a key nobody writes any more.
	essential := voyager.EssentialEndpointName()
	if _, ok := bodies[essential]; !ok {
		return nil, c.mapFailure(logger.Errorf, publicID, failures[essential])
	}

	for name, err := range failures {
		logger.Warnf("linkedin: section source %q unavailable for %q: %v", name, publicID, err)
	}

	profile, err := voyager.Assemble(publicID, bodies)
	if err != nil {
		return nil, exceptions.Wrap(logger.Errorf, exceptions.UpstreamShapeChanged,
			"assemble profile %q: %v", publicID, err)
	}

	if profile.Identity.Name == "" {
		// A parseable response with no person means blocked or deleted.
		return nil, exceptions.Wrap(logger.Errorf, exceptions.ProfileNotFound,
			"profile %q parsed but carried no identity", publicID)
	}

	return profile, nil
}

func (c *ClientLinkedIn) SessionValid(ctx context.Context) error {
	return c.voyager.SessionValid(ctx)
}

// mapFailure turns an upstream failure into the service's own error code.
func (c *ClientLinkedIn) mapFailure(logf func(string, ...any), publicID string, err error) *exceptions.ApplicationError {
	if err == nil {
		err = errors.New("the essential endpoint returned no body")
	}

	var status *voyager.StatusError
	if errors.As(err, &status) {
		switch {
		case status.SessionExpired():
			return exceptions.Wrap(logf, exceptions.SessionExpired,
				"linkedin rejected our session fetching %q: %v", publicID, err)
		case status.NotFound():
			return exceptions.Wrap(logf, exceptions.ProfileNotFound,
				"linkedin has no visible profile %q: %v", publicID, err)
		case status.RateLimited():
			return exceptions.Wrap(logf, exceptions.UpstreamRateLimited,
				"linkedin is throttling us fetching %q: %v", publicID, err)
		}
	}

	return exceptions.Wrap(logf, exceptions.UpstreamShapeChanged,
		"failed to fetch profile %q: %v", publicID, err)
}

// jitter pauses briefly so the request rhythm is not machine-regular.
func (c *ClientLinkedIn) jitter() {
	time.Sleep(time.Duration(rand.Int63n(int64(jitterMax))))
}
