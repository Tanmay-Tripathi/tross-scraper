package services

import (
	"context"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/response"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/mapper"
)

// ScrapeRequest is one profile lookup, already validated.
type ScrapeRequest struct {
	PublicID string
	// Sections is the default merged with any per-request override.
	Sections config.SectionSet
	// Refresh bypasses the cache.
	Refresh bool
}

// ServiceProfileMethods is the profile use case.
type ServiceProfileMethods interface {
	// Scrape renders the profile against the requested sections, using the cache when it can.
	Scrape(ctx context.Context, req ScrapeRequest) (response.ProfileResult, *exceptions.ApplicationError)
}

type ServiceProfile struct {
	Access *ServiceAccess
}

func NewServiceProfile(access *ServiceAccess) ServiceProfileMethods {
	return &ServiceProfile{Access: access}
}

func (s *ServiceProfile) Scrape(ctx context.Context, req ScrapeRequest) (response.ProfileResult, *exceptions.ApplicationError) {
	logger := s.Access.Logger.With(ctx)
	repo := s.Access.Repositories.Profile

	if req.Refresh {
		if err := repo.Invalidate(ctx, req.PublicID); err != nil {
			logger.Warnf("could not invalidate cached profile %q: %v", req.PublicID, err)
		}
	} else if cached, found := repo.GetCached(ctx, req.PublicID); found {
		logger.Infof("serving profile %q from cache", req.PublicID)
		return mapper.ToProfileResult(cached, req.Sections, true), nil
	}

	profile, appErr := s.fetchLive(ctx, req.PublicID)
	if appErr != nil {
		return response.ProfileResult{}, appErr
	}

	// The full profile is cached, not the filtered response.
	ttl := time.Duration(s.Access.Cfg.LinkedIn.CacheTTLMinutes) * time.Minute
	if err := repo.Cache(ctx, profile, ttl); err != nil {
		logger.Warnf("could not cache profile %q: %v", req.PublicID, err)
	}

	return mapper.ToProfileResult(profile, req.Sections, false), nil
}

// fetchLive performs the upstream call, guarded by the daily budget.
func (s *ServiceProfile) fetchLive(ctx context.Context, publicID string) (*models.Profile, *exceptions.ApplicationError) {
	logger := s.Access.Logger.With(ctx)

	client := s.Access.Clients.LinkedIn
	if client == nil {
		return nil, exceptions.Wrap(logger.Errorf, exceptions.SessionExpired,
			"profile %q requested but no linkedin client is configured", publicID)
	}

	// Checked only for live fetches, so cached reads stay free. This is what
	// protects the account when a public endpoint gets hammered.
	budget := s.Access.Cfg.LinkedIn.DailyScrapeBudget
	allowed, used, err := s.Access.Repositories.Profile.ConsumeScrapeBudget(ctx, budget)
	if err != nil {
		logger.Warnf("scrape budget check failed, allowing request: %v", err)
	}
	if !allowed {
		return nil, exceptions.Wrap(logger.Errorf, exceptions.ScrapeBudgetExhausted,
			"daily scrape budget spent (%d used of %d)", used, budget)
	}

	logger.Infof("fetching profile %q from linkedin (%d/%d scrapes used today)", publicID, used, budget)
	return client.FetchProfile(ctx, publicID)
}
