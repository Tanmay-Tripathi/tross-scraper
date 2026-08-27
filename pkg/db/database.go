// Package db owns the service's persistence connections: the Postgres store
// (master/slave GORM handles plus migrations) and the Redis cache.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

const (
	idleTransactionTimeoutParam = "idle_in_transaction_session_timeout"
	migrationsDir               = "./migrations/postgres"
	migrationsSourceURL         = "file://" + migrationsDir
)

// DBConfig tunes a single connection pool. Zero values fall back to defaults.
type DBConfig struct {
	MaxOpenConns         int `yaml:"max_open_connections"`
	MaxIdleConns         int `yaml:"max_idle_connections"`
	MaxLifetimeInMinutes int `yaml:"max_lifetime_in_minutes"`
	MaxIdleTimeInMinutes int `yaml:"max_idletime_in_minutes"`
}

// StoreConfig describes the Postgres connections the store should open.
type StoreConfig struct {
	MasterDsn    string
	SlaveDsn     string
	MasterConfig DBConfig
	SlaveConfig  DBConfig
	AppName      string
	// SkipMigrations disables the automatic migration run on startup. Useful
	// for tests and for read-only replicas of the service.
	SkipMigrations bool
}

// Store holds the master (writes) and slave (reads) GORM handles.
type Store struct {
	MasterDB     *gorm.DB
	SlaveDB      *gorm.DB
	DatabaseName string
	Logger       log.Logger
}

// NewStore connects to Postgres, applies pending migrations and installs
// OpenTelemetry tracing on both handles.
func NewStore(logger log.Logger, cfg StoreConfig) (*Store, error) {
	if strings.TrimSpace(cfg.MasterDsn) == "" {
		return nil, errors.New("master database dsn is empty")
	}

	slaveDsn := cfg.SlaveDsn
	if strings.TrimSpace(slaveDsn) == "" {
		slaveDsn = cfg.MasterDsn
	}

	masterDB, err := connect(applyTimeoutConfig(cfg.MasterDsn), cfg.MasterConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to master database: %w", err)
	}

	slaveDB, err := connect(applyTimeoutConfig(slaveDsn), cfg.SlaveConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to slave database: %w", err)
	}

	databaseName, err := currentDatabaseName(masterDB)
	if err != nil {
		return nil, err
	}

	store := &Store{
		MasterDB:     masterDB,
		SlaveDB:      slaveDB,
		DatabaseName: databaseName,
		Logger:       logger,
	}

	if !cfg.SkipMigrations {
		if err := store.RunMigrations(); err != nil {
			return nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	store.setupTracing()

	logger.Infof("connected to postgres database %q", databaseName)
	return store, nil
}

func connect(dsn string, cfg DBConfig) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve *sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	sqlDB.SetMaxOpenConns(orDefault(cfg.MaxOpenConns, 10))
	sqlDB.SetMaxIdleConns(orDefault(cfg.MaxIdleConns, 2))
	sqlDB.SetConnMaxLifetime(time.Duration(orDefault(cfg.MaxLifetimeInMinutes, 10)) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(orDefault(cfg.MaxIdleTimeInMinutes, 10)) * time.Minute)

	return gormDB, nil
}

func orDefault(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func currentDatabaseName(gormDB *gorm.DB) (string, error) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return "", fmt.Errorf("resolve *sql.DB: %w", err)
	}

	var name string
	if err := sqlDB.QueryRow("SELECT current_database()").Scan(&name); err != nil {
		return "", fmt.Errorf("resolve database name: %w", err)
	}
	return name, nil
}

// RunMigrations applies every pending migration in migrations/postgres. An
// empty migrations directory is not an error — a fresh service has no schema
// yet, and golang-migrate would otherwise refuse to open the source.
func (s *Store) RunMigrations() error {
	pending, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("scan migrations directory: %w", err)
	}
	if len(pending) == 0 {
		s.Logger.Infof("no migration files in %s, skipping", migrationsDir)
		return nil
	}

	sqlDB, err := s.MasterDB.DB()
	if err != nil {
		return fmt.Errorf("resolve *sql.DB from master: %w", err)
	}

	migrator, err := newMigrator(sqlDB, s.DatabaseName)
	if err != nil {
		return err
	}

	if err := migrator.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			s.Logger.Info("no migrations to apply")
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}

	s.Logger.Info("migrations applied successfully")
	return nil
}

func newMigrator(sqlDB *sql.DB, databaseName string) (*migrate.Migrate, error) {
	driver, err := migratePostgres.WithInstance(sqlDB, &migratePostgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("create migrate driver: %w", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(migrationsSourceURL, databaseName, driver)
	if err != nil {
		return nil, fmt.Errorf("initialize migrator: %w", err)
	}
	return migrator, nil
}

func (s *Store) setupTracing() {
	for role, handle := range map[string]*gorm.DB{"master": s.MasterDB, "slave": s.SlaveDB} {
		plugin := tracing.NewPlugin(tracing.WithAttributes(attribute.String("db.role", role)))
		if err := handle.Use(plugin); err != nil {
			s.Logger.Errorf("failed to enable tracing on %s database: %v", role, err)
		}
	}
}

// Begin opens a transaction on the master handle.
func (s *Store) Begin(ctx context.Context) (*gorm.DB, error) {
	tx := s.MasterDB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

// Rollback aborts tx, ignoring a nil transaction.
func (s *Store) Rollback(tx *gorm.DB) {
	if tx != nil {
		tx.Rollback()
	}
}

// Commit finalises tx.
func (s *Store) Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

// Ping verifies that both the master and slave handles are reachable.
func (s *Store) Ping(ctx context.Context) error {
	var errs []error
	for role, handle := range map[string]*gorm.DB{"master": s.MasterDB, "slave": s.SlaveDB} {
		sqlDB, err := handle.DB()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", role, err))
			continue
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", role, err))
		}
	}
	return errors.Join(errs...)
}

// Close releases both connection pools.
func (s *Store) Close() error {
	var errs []error
	for _, handle := range []*gorm.DB{s.MasterDB, s.SlaveDB} {
		if handle == nil {
			continue
		}
		sqlDB, err := handle.DB()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := sqlDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// applyTimeoutConfig makes sure a Postgres DSN caps idle-in-transaction time so
// a stuck handler cannot hold a connection open indefinitely.
func applyTimeoutConfig(dsn string) string {
	if dsn == "" || strings.Contains(dsn, idleTransactionTimeoutParam) {
		return dsn
	}

	param := idleTransactionTimeoutParam + "=30s"
	if strings.Contains(dsn, "?") {
		return dsn + "&" + param
	}
	return dsn + "?" + param
}
