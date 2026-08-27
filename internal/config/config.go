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

// defaultUserAgent must look like a real browser; Go's default is an easy tell.
const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

// RedisConfig describes the cache connection.
type RedisConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	TlsEnabled bool   `yaml:"tls_enabled"`
	PoolSize   int    `yaml:"pool_size"`
}

// DatabaseConfig describes the Postgres connections. An empty slave DSN reuses the master.
type DatabaseConfig struct {
	MasterDatabaseDsn string `yaml:"master_database_dsn"`
	SlaveDatabaseDsn  string `yaml:"slave_database_dsn"`
	MaxOpenConns      int    `yaml:"max_open_connections"`
	MaxIdleConns      int    `yaml:"max_idle_connections"`
	SkipMigrations    bool   `yaml:"skip_migrations"`
}

// StringList is a []string that also accepts a comma-separated scalar, so a list
// can be set from one environment variable.
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

// LinkedInConfig holds the upstream credentials and tuning. The cookies are never
// committed: config names the env var, the deployment supplies the value.
type LinkedInConfig struct {
	// LiAt is the li_at session cookie.
	LiAt string `yaml:"li_at"`
	// JSessionID is the JSESSIONID value without its quotes. The client sends it
	// quoted in the Cookie header and bare as csrf-token; LinkedIn wants both.
	JSessionID string `yaml:"jsessionid"`
	// UserAgent is sent on every upstream call and must look like a real browser.
	UserAgent string `yaml:"user_agent"`
	// RequestTimeoutSeconds bounds one upstream call.
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds"`
	// CacheTTLMinutes is how long a scraped profile stays replayable from Redis.
	CacheTTLMinutes int `yaml:"cache_ttl_minutes"`
	// DailyScrapeBudget caps live scrapes per day — the guard that protects the
	// account when traffic spikes. Zero means unlimited; never deploy that.
	DailyScrapeBudget int `yaml:"daily_scrape_budget"`
}

// Configured reports whether both cookies are present.
func (l LinkedInConfig) Configured() bool {
	return strings.TrimSpace(l.LiAt) != "" && strings.TrimSpace(l.JSessionID) != ""
}

// CorsConfig lists the browser origins allowed to call this API. Empty disables
// CORS; "*" allows any origin, which is unsafe for anything trusting cookies.
type CorsConfig struct {
	AllowedOrigins StringList `yaml:"allowed_origins"`
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
	LinkedIn LinkedInConfig `yaml:"LinkedIn"`
	Sections SectionSet     `yaml:"Sections"`
}

// Load reads the YAML at path, expands ${VAR} and ${VAR:-default} from the
// environment, applies defaults and validates. Expansion keeps secrets out of the repo.
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

// expandEnv resolves ${VAR} and ${VAR:-default}. The fallback form matters for
// numeric and boolean fields, where an unset var would leave a bare YAML key.
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
	if c.LinkedIn.UserAgent == "" {
		c.LinkedIn.UserAgent = defaultUserAgent
	}
	if c.LinkedIn.RequestTimeoutSeconds == 0 {
		c.LinkedIn.RequestTimeoutSeconds = 20
	}
	if c.LinkedIn.CacheTTLMinutes == 0 {
		c.LinkedIn.CacheTTLMinutes = 360 // 6 hours
	}
	// All-on by default; switching a section off is the deliberate act.
	if len(c.Sections) == 0 {
		c.Sections = AllEnabled()
	}
	// DevTools shows JSESSIONID quoted and people paste it verbatim, so strip them.
	c.LinkedIn.JSessionID = strings.Trim(strings.TrimSpace(c.LinkedIn.JSessionID), `"`)
	c.LinkedIn.LiAt = strings.TrimSpace(c.LinkedIn.LiAt)
}

// validate fails fast at startup rather than on the first request.
func (c *Config) validate() error {
	var problems []error

	if strings.TrimSpace(c.AppName) == "" {
		problems = append(problems, errors.New("key AppName is required"))
	}
	if strings.TrimSpace(c.Database.MasterDatabaseDsn) == "" {
		problems = append(problems, errors.New("key Database.master_database_dsn is required"))
	}
	if strings.TrimSpace(c.Redis.Host) == "" {
		problems = append(problems, errors.New("key Redis.host is required"))
	}
	if !global.Environment(c.Environment).IsValid() {
		problems = append(problems, fmt.Errorf("key Environment is %q, want one of local, stg, uat, prd", c.Environment))
	}
	return errors.Join(problems...)
}

// Env returns the typed deployment environment.
func (c *Config) Env() global.Environment {
	return global.Environment(c.Environment)
}
