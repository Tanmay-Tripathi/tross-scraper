package app

import "github.com/gin-gonic/gin"

// Route group prefixes: public is unauthenticated, protected sits behind
// end-user auth, private is cluster-internal only.
const (
	PublicApiV1  = "/public/v1"
	PrivateApiV1 = "/private/v1"
)

// addRoutes binds every route group. Each feature owns a routes_<feature>.go and
// registers here; a route never holds logic of its own.
func (app *App) addRoutes(router *gin.Engine) {
	app.addHealthRoutes(router)
	app.addProfileRoutes(router)
}
