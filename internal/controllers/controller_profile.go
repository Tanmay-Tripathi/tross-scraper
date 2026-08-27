package controllers

import (
	"github.com/gin-gonic/gin"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/requests"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/services"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/utils"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/validation"
)

// ControllerProfileMethods exposes the scrape endpoint.
type ControllerProfileMethods interface {
	// Scrape turns a LinkedIn profile URL into structured JSON.
	Scrape(ctx *gin.Context)
}

type ControllerProfile struct {
	Access *ControllerAccess
}

func NewControllerProfile(access *ControllerAccess) ControllerProfileMethods {
	return &ControllerProfile{Access: access}
}

func (c *ControllerProfile) Scrape(ctx *gin.Context) {
	logger := c.Access.Logger.With(ctx.Request.Context())

	var body requests.RequestScrapeProfile
	if err := ctx.ShouldBindJSON(&body); err != nil {
		logger.Warnf("rejected malformed scrape request: %v", err)
		utils.SendApiResponseV2[any](ctx, nil, nil, exceptions.New(exceptions.InvalidRequest))
		return
	}

	publicID, appErr := validation.ParseProfileURL(body.ProfileURL)
	if appErr != nil {
		// Distinguish a company page from a non-URL, so the log says which.
		if validation.IsNonProfileLinkedInURL(body.ProfileURL) {
			logger.Warnf("rejected non-member linkedin url")
		}
		utils.SendApiResponseV2[any](ctx, nil, nil, appErr)
		return
	}

	sections, appErr := validation.MergeSections(c.Access.Cfg.Sections, body.Sections)
	if appErr != nil {
		utils.SendApiResponseV2[any](ctx, nil, nil, appErr)
		return
	}

	result, appErr := c.Access.Services.Profile.Scrape(ctx.Request.Context(), services.ScrapeRequest{
		PublicID: publicID,
		Sections: sections,
		Refresh:  body.Refresh,
	})
	if appErr != nil {
		utils.SendApiResponseV2[any](ctx, nil, nil, appErr)
		return
	}

	utils.SendApiResponseV2(ctx, result, nil, nil)
}

// compile-time guard that the controller keeps satisfying its interface.
var _ ControllerProfileMethods = (*ControllerProfile)(nil)
