// Command server is the service entry point: it loads configuration, builds the
// logger, and hands control to the application.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Tanmay-Tripathi/tross-scraper/cmd/app"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// Version is stamped in at build time via -ldflags.
var Version = "dev"

func main() {
	configPath := flag.String("config", "./config/local.yml", "path to the config file")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	logger := log.New(log.LogConfig{
		ServiceName: cfg.AppName,
		AppEnv:      cfg.Environment,
		AppVersion:  fmt.Sprintf("%s(%s)", cfg.AppVersion, Version),
		Level:       cfg.LogLevel,
	})

	ctx := context.Background()

	application, err := app.New(ctx, cfg, logger)
	if err != nil {
		return err
	}

	return application.Run(ctx)
}
