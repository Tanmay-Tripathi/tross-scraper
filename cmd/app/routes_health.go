package app

import "github.com/gin-gonic/gin"

// addHealthRoutes exposes liveness and readiness probes. The public one is for
// uptime checks; the private ones are for the orchestrator.
func (app *App) addHealthRoutes(router *gin.Engine) {
	controller := app.controllers.Health

	public := router.Group(PublicApiV1)
	{
		public.GET("/health", controller.Live)
	}

	private := router.Group(PrivateApiV1)
	{
		private.GET("/health/live", controller.Live)
		private.GET("/health/ready", controller.Ready)
	}
}
