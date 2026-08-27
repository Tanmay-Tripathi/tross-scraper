package controllers

import (
	"github.com/gin-gonic/gin"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/mapper"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/utils"
)

// ControllerHealthMethods exposes the liveness and readiness handlers.
type ControllerHealthMethods interface {
	// Live answers as long as the process is running. Container orchestrators
	// use it to decide whether to restart the pod.
	Live(ctx *gin.Context)
	// Ready probes the downstream dependencies and fails the request when any
	// of them is unreachable, so a broken instance is pulled out of rotation.
	Ready(ctx *gin.Context)
}

type ControllerHealth struct {
	Access *ControllerAccess
}

func NewControllerHealth(access *ControllerAccess) ControllerHealthMethods {
	return &ControllerHealth{Access: access}
}

func (c *ControllerHealth) Live(ctx *gin.Context) {
	utils.SendApiResponseV2(ctx, gin.H{
		"service":     c.Access.Cfg.AppName,
		"version":     c.Access.Cfg.AppVersion,
		"environment": c.Access.Cfg.Environment,
	}, nil, nil)
}

func (c *ControllerHealth) Ready(ctx *gin.Context) {
	status := c.Access.Services.Health.Check(ctx.Request.Context())
	body := mapper.ToHealthResponse(status)

	if !status.IsHealthy() {
		utils.SendApiResponseV2(ctx, body, nil, exceptions.New(exceptions.HealthDependencyDown))
		return
	}

	utils.SendApiResponseV2(ctx, body, nil, nil)
}
