package config

import (
	"path/filepath"
	"testing"
)

func TestLoadProductionConfig(t *testing.T) {
	path := filepath.Join("..", "..", "config", "production.yml")

	tests := []struct {
		name        string
		corsOrigins string
		want        []string
	}{
		{name: "unset expands to no origins", corsOrigins: "", want: nil},
		{name: "single origin", corsOrigins: "https://a.example.com", want: []string{"https://a.example.com"}},
		{
			name:        "comma separated origins",
			corsOrigins: "https://a.example.com, https://b.example.com",
			want:        []string{"https://a.example.com", "https://b.example.com"},
		},
		{name: "wildcard", corsOrigins: "*", want: []string{"*"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CORS_ALLOWED_ORIGINS", tc.corsOrigins)
			t.Setenv("DATABASE_URL", "postgres://user:pass@host:5432/db?sslmode=disable")
			t.Setenv("REDIS_HOST", "redis")

			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load() returned an error: %v", err)
			}

			if len(cfg.Cors.AllowedOrigins) != len(tc.want) {
				t.Fatalf("AllowedOrigins = %#v, want %#v", cfg.Cors.AllowedOrigins, tc.want)
			}
			for i, origin := range tc.want {
				if cfg.Cors.AllowedOrigins[i] != origin {
					t.Errorf("AllowedOrigins[%d] = %q, want %q", i, cfg.Cors.AllowedOrigins[i], origin)
				}
			}
		})
	}
}

// The committed production config relies on ${VAR:-default} for its numeric and
// boolean fields; a bare ${VAR} would leave them empty and silently zeroed.
func TestProductionConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@host:5432/db?sslmode=disable")
	t.Setenv("REDIS_HOST", "redis")

	cfg, err := Load(filepath.Join("..", "..", "config", "production.yml"))
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	if cfg.ServerPort != 4201 {
		t.Errorf("ServerPort = %d, want 4201", cfg.ServerPort)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d, want 6379", cfg.Redis.Port)
	}
	if cfg.Environment != "prd" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "prd")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	// An empty slave DSN must fall back to the master.
	if cfg.Database.SlaveDatabaseDsn != cfg.Database.MasterDatabaseDsn {
		t.Errorf("SlaveDatabaseDsn = %q, want it to fall back to the master DSN", cfg.Database.SlaveDatabaseDsn)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_HOST", "")

	if _, err := Load(filepath.Join("..", "..", "config", "production.yml")); err == nil {
		t.Fatal("Load() succeeded with a missing database DSN and redis host, want an error")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join("testdata", "does-not-exist.yml")); err == nil {
		t.Fatal("Load() succeeded for a missing file, want an error")
	}
}
