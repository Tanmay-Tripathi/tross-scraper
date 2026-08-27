// Package config loads and validates the service configuration. Every value the
// service needs comes from here — nothing reads os.Getenv at the point of use.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
)

// RedisConfig describes the cache connection.
type RedisConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	TlsEnabled bool   `yaml:"tls_enabled"`
	PoolSize   int    `yaml:"pool_size"`
}

// DatabaseConfig describes the Postgres connections. An empty slave DSN falls
// back to the master, which is the normal single-instance setup.
type DatabaseConfig struct {
	MasterDatabaseDsn string `yaml:"master_database_dsn"`
	SlaveDatabaseDsn  string `yaml:"slave_database_dsn"`
	MaxOpenConns      int    `yaml:"max_open_connections"`
	MaxIdleConns      int    `yaml:"max_idle_connections"`
	SkipMigrations    bool   `yaml:"skip_migrations"`
}

// StringList is a []string that also accepts a comma-separated scalar. That
// second form is what makes a list configurable from a single environment
// variable, which is all a container platform can inject.
type StringList []string

func (l *StringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		var items []string
		if err := node.Decode(&items); err != nil {
			return err
		}
		*l = trimAndCompact(items)
		return nil

	case yaml.ScalarNode:
		var raw string
		if err := node.Decode(&raw); err != nil {
			return err
		}
		*l = trimAndCompact(strings.Split(raw, ","))
		return nil

	default:
		return fmt.Errorf("expected a list or a comma-separated string, got %v", node.Kind)
	}
}

func trimAndCompact(items []string) StringList {
	var result StringList
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// CorsConfig lists the browser origins allowed to call this API. An empty list
// disables CORS entirely, which is right for a service with no browser client.
// "*" allows any origin — acceptable for a public read-only API, never for one
// that trusts cookies.
type CorsConfig struct {
	AllowedOrigins StringList `yaml:"allowed_origins"`
}

// SQSConfig describes the AWS messaging setup.
type SQSConfig struct {
	Enabled bool   `yaml:"enabled"`
	Region  string `yaml:"region"`
	// Endpoint overrides the AWS endpoint; point it at LocalStack
	// (http://localhost:4566) for local development.
	Endpoint string `yaml:"endpoint"`
}

// Config is the fully resolved application configuration.
type Config struct {
	ServerPort      int    `yaml:"ServerPort"`
	AppName         string `yaml:"AppName"`
	AppVersion      string `yaml:"AppVersion"`
	BaseUrl         string `yaml:"BaseUrl"`
	Environment     string `yaml:"Environment"`
	LogLevel        string `yaml:"LogLevel"`
	OtlpExporterUrl string `yaml:"OtlpExporterUrl"`

	Redis    RedisConfig    `yaml:"Redis"`
	Database DatabaseConfig `yaml:"Database"`
	Cors     CorsConfig     `yaml:"Cors"`
	SQS      SQSConfig      `yaml:"SQS"`
}

// Load reads the YAML file at path, expands ${VAR} and ${VAR:-default}
// references against the environment, applies defaults and validates the result.
//
// Expanding environment variables is what keeps secrets out of the repository:
// the committed file names them, the deployment supplies their values.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(expandEnv(string(raw))), &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", path, err)
	}

	return &cfg, nil
}

// expandEnv resolves ${VAR} and ${VAR:-default} references. The default form
// matters for numeric and boolean YAML fields: an unset ${VAR} would expand to
// nothing and leave the document with a bare key, so every such field in a
// committed config supplies a fallback.
func expandEnv(raw string) string {
	return os.Expand(raw, func(reference string) string {
		name, fallback, hasFallback := strings.Cut(reference, ":-")

		if value, ok := os.LookupEnv(name); ok && value != "" {
			return value
		}
		if hasFallback {
			return fallback
		}
		return ""
	})
}

func (c *Config) applyDefaults() {
	if c.ServerPort == 0 {
		c.ServerPort = 4201
	}
	if c.Environment == "" {
		c.Environment = string(global.LocalEnv)
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.AppVersion == "" {
		c.AppVersion = "0.0.0"
	}
	if c.Redis.Port == 0 {
		c.Redis.Port = 6379
	}
	if strings.TrimSpace(c.Database.SlaveDatabaseDsn) == "" {
		c.Database.SlaveDatabaseDsn = c.Database.MasterDatabaseDsn
	}
}

// validate fails fast at startup rather than letting a misconfigured instance
// serve traffic and fail on the first request.
func (c *Config) validate() error {
	var problems []error

	if strings.TrimSpace(c.AppName) == "" {
		problems = append(problems, errors.New("AppName is required"))
	}
	if strings.TrimSpace(c.Database.MasterDatabaseDsn) == "" {
		problems = append(problems, errors.New("Database.master_database_dsn is required"))
	}
	if strings.TrimSpace(c.Redis.Host) == "" {
		problems = append(problems, errors.New("Redis.host is required"))
	}
	if !global.Environment(c.Environment).IsValid() {
		problems = append(problems, fmt.Errorf("Environment %q must be one of local, stg, uat, prd", c.Environment))
	}
	if c.SQS.Enabled && strings.TrimSpace(c.SQS.Region) == "" {
		problems = append(problems, errors.New("SQS.region is required when SQS.enabled is true"))
	}

	return errors.Join(problems...)
}

// Env returns the typed deployment environment.
func (c *Config) Env() global.Environment {
	return global.Environment(c.Environment)
}
