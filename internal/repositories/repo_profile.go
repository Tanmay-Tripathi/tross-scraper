package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
)

const (
	profileCachePrefix = "profile:v1:"
	budgetKeyPrefix    = "scrape-budget:"
)

// RepositoryProfileMethods caches scraped profiles and tracks the scrape budget.
// The full profile is cached, so two callers wanting different sections cost one scrape.
type RepositoryProfileMethods interface {
	// GetCached returns a stored profile, and whether one was found.
	GetCached(ctx context.Context, publicID string) (*models.Profile, bool)
	// Cache stores a profile for ttl.
	Cache(ctx context.Context, profile *models.Profile, ttl time.Duration) error
	// Invalidate drops a cached profile, used by a refresh request.
	Invalidate(ctx context.Context, publicID string) error
	// ConsumeScrapeBudget increments today's counter; a budget of zero is unlimited.
	ConsumeScrapeBudget(ctx context.Context, budget int) (allowed bool, used int64, err error)
}

type RepositoryProfile struct {
	access *RepositoryAccess
}

func NewRepositoryProfile(access *RepositoryAccess) RepositoryProfileMethods {
	return &RepositoryProfile{access: access}
}

func (r *RepositoryProfile) GetCached(ctx context.Context, publicID string) (*models.Profile, bool) {
	var profile models.Profile
	found, err := r.access.Cache.GetJSON(ctx, profileCacheKey(publicID), &profile)
	if err != nil {
		// A cache outage must not fail the request.
		r.access.Logger.With(ctx).Warnf("profile cache read failed for %q: %v", publicID, err)
		return nil, false
	}
	if !found {
		return nil, false
	}
	return &profile, true
}

func (r *RepositoryProfile) Cache(ctx context.Context, profile *models.Profile, ttl time.Duration) error {
	return r.access.Cache.SetJSON(ctx, profileCacheKey(profile.PublicID), profile, ttl)
}

func (r *RepositoryProfile) Invalidate(ctx context.Context, publicID string) error {
	return r.access.Cache.Delete(ctx, profileCacheKey(publicID))
}

// ConsumeScrapeBudget uses a self-expiring per-day key, so the budget resets
// without any scheduled cleanup.
func (r *RepositoryProfile) ConsumeScrapeBudget(ctx context.Context, budget int) (bool, int64, error) {
	if budget <= 0 {
		return true, 0, nil
	}

	key := budgetKeyPrefix + time.Now().UTC().Format("2006-01-02")

	used, err := r.access.Cache.Incr(ctx, key)
	if err != nil {
		// The cache is a guard rail, not the service: allow rather than fail.
		r.access.Logger.With(ctx).Warnf("scrape budget counter unavailable, allowing request: %v", err)
		return true, 0, nil
	}

	// Give the counter a lifetime on first creation.
	if used == 1 {
		if err := r.access.Cache.Expire(ctx, key, 48*time.Hour); err != nil {
			r.access.Logger.With(ctx).Warnf("could not set scrape budget expiry: %v", err)
		}
	}

	return used <= int64(budget), used, nil
}

func profileCacheKey(publicID string) string {
	return fmt.Sprintf("%s%s", profileCachePrefix, publicID)
}
