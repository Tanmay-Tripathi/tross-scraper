package controllers

import (
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/services"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// ControllerAccess is the shared dependency set every controller receives.
type ControllerAccess struct {
	Cfg      *config.Config
	Logger   log.Logger
	Services *services.Services
}
