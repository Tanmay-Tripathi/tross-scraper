// Package app is the composition root: it builds every dependency, serves HTTP,
// and tears everything down on shutdown.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tanmay-Tripathi/tross-scraper/cmd/app/middlewares"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/clients"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/controllers"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/repositories"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/services"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/db"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/telemetry"
)

const (
	requestTimeout  = 60 * time.Second
	shutdownTimeout = 15 * time.Second
)

// App holds every long-lived dependency for one running instance.
type App struct {
	cfg    *config.Config
	logger log.Logger

	cache     db.CacheStoreMethods
	telemetry telemetry.Methods

	repos       *repositories.Repositories
	clients     *clients.Clients
	services    *services.Services
	controllers *controllers.Controllers
	middlewares *middlewares.Middlewares

	router *gin.Engine
	http   *http.Server
}

// New builds a fully wired application; the caller must call Run. Wiring order
// mirrors the dependency direction, from stores out to the router.
func New(ctx context.Context, cfg *config.Config, logger log.Logger) (*App, error) {
	app := &App{cfg: cfg, logger: logger}

	if err := app.initCache(ctx); err != nil {
		return nil, err
	}

	app.telemetry = telemetry.New(telemetry.Config{
		Logger:      logger,
		ServiceName: cfg.AppName,
		AppEnv:      cfg.Environment,
		AppVersion:  cfg.AppVersion,
		ExporterURL: cfg.OtlpExporterUrl,
	})

	// No Postgres: the service keeps no relational state, only the Redis cache.
	// The *db.Store seam stays wired so adding one is a change here and nowhere else.
	var store *db.Store

	app.clients = clients.NewClients(cfg, logger, app.cache)
	app.repos = repositories.NewRepositories(store, app.cache, logger)
	app.services = services.NewServices(cfg, store, app.repos, app.cache, logger, app.clients)
	app.controllers = controllers.NewControllers(cfg, logger, app.services)
	app.middlewares = middlewares.NewMiddlewares(cfg, store, app.repos, app.cache, logger)

	app.router = app.newRouter()
	app.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      app.router,
		ReadTimeout:  requestTimeout,
		WriteTimeout: requestTimeout,
		IdleTimeout:  requestTimeout,
	}

	return app, nil
}

func (app *App) initCache(ctx context.Context) error {
	cache, err := db.NewRedisClient(ctx, db.CacheConfig{
		Host:       app.cfg.Redis.Host,
		Port:       strconv.Itoa(app.cfg.Redis.Port),
		Username:   app.cfg.Redis.Username,
		Password:   app.cfg.Redis.Password,
		TlsEnabled: app.cfg.Redis.TlsEnabled,
		PoolSize:   app.cfg.Redis.PoolSize,
	}, app.logger)
	if err != nil {
		return fmt.Errorf("cache initialization failed: %w", err)
	}

	app.cache = cache
	return nil
}

func (app *App) newRouter() *gin.Engine {
	if app.cfg.Env().IsProdLike() {
		app.logger.Infof("setting gin to release mode")
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(app.middlewares.Cors.Handler())
	app.telemetry.EnableGinTracing(router)
	app.addRoutes(router)

	return router
}

// Run serves HTTP until SIGINT or SIGTERM, then drains and releases everything.
func (app *App) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		app.logger.Infof("%s listening on port %d", app.cfg.AppName, app.cfg.ServerPort)
		if err := app.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		app.shutdown()
		return err
	case <-ctx.Done():
		app.logger.Infof("shutdown signal received, draining in-flight requests")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := app.http.Shutdown(shutdownCtx)
	app.shutdown()
	return err
}

// shutdown releases dependencies in reverse construction order.
func (app *App) shutdown() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if app.telemetry != nil {
		if err := app.telemetry.Shutdown(shutdownCtx); err != nil {
			app.logger.Errorf("failed to flush telemetry: %v", err)
		}
	}
	if app.cache != nil {
		if err := app.cache.Close(); err != nil {
			app.logger.Errorf("failed to close cache: %v", err)
		}
	}

	app.logger.Infof("shutdown complete")
}
