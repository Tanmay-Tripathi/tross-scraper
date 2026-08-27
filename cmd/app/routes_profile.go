package app

import "github.com/gin-gonic/gin"

// addProfileRoutes exposes the scrape endpoint. It is public by design; the
// account is protected by the cache and the daily scrape budget, not by auth.
func (app *App) addProfileRoutes(router *gin.Engine) {
	controller := app.controllers.Profile

	public := router.Group(PublicApiV1)
	{
		public.POST("/profile", controller.Scrape)
	}

	// Same handler on the authenticated group, ready for when auth lands.
	protected := router.Group(ProtectedApiV1)
	{
		protected.POST("/profile", controller.Scrape)
	}
}
