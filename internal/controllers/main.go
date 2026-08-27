package controllers

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/services"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// Controllers aggregates every HTTP controller the router can bind.
type Controllers struct {
	Health ControllerHealthMethods
}

// NewControllers wires the transport layer.
func NewControllers(cfg *config.Config, logger log.Logger, services *services.Services) *Controllers {
	access := &ControllerAccess{
		Cfg:      cfg,
		Logger:   logger,
		Services: services,
	}

	return &Controllers{
		Health: NewControllerHealth(access),
	}
}
