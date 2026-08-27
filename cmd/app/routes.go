package app

import "github.com/gin-gonic/gin"

// Route group prefixes. Public routes are unauthenticated, protected routes sit
// behind end-user auth, and private routes are reachable only from inside the
// cluster.
const (
	PublicApiV1    = "/public/v1"
	ProtectedApiV1 = "/v1"
	PrivateApiV1   = "/private/v1"
)

// addRoutes binds every route group. Each feature owns one routes_<feature>.go
// file and registers itself here. Handlers come from app.controllers and
// middlewares from app.middlewares — a route must never hold logic of its own.
func (app *App) addRoutes(router *gin.Engine) {
	app.addHealthRoutes(router)
}
