package db

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

const defaultPoolSize = 20

// CacheStoreMethods is the cache surface the service depends on; an interface so
// callers can be tested against a fake.
type CacheStoreMethods interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// SetJSON marshals value to JSON before storing it.
	SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error
	// Get returns the value at key, or defaultValue when the key is missing.
	Get(ctx context.Context, key, defaultValue string) (string, error)
	// GetJSON unmarshals the value at key into dest and reports whether the key existed.
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	Delete(ctx context.Context, keys ...string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Ping(ctx context.Context) error
	Close() error
}

// CacheConfig describes how to reach Redis.
type CacheConfig struct {
	Host       string
	Port       string
	Username   string
	Password   string
	TlsEnabled bool
	// PoolSize caps the socket pool; zero uses defaultPoolSize.
	PoolSize int
}

type cacheStore struct {
	client *redis.Client
}

// NewRedisClient dials Redis, verifies it and adds OTel instrumentation. The idle
// window stays under the ~20 min managed providers use before dropping connections.
func NewRedisClient(ctx context.Context, cfg CacheConfig, logger log.Logger) (CacheStoreMethods, error) {
	poolSize := cfg.PoolSize
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}

	options := &redis.Options{
		Addr:            cfg.Host + ":" + cfg.Port,
		Username:        cfg.Username,
		Password:        cfg.Password,
		DB:              0,
		PoolSize:        poolSize,
		MinIdleConns:    min(2, poolSize),
		ConnMaxIdleTime: 4 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}

	if cfg.TlsEnabled {
		logger.Infof("connecting to redis with TLS enabled")
		options.TLSConfig = &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	}

	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	instrument(client, logger)
	logger.Infof("successfully connected to redis at %s", options.Addr)

	return &cacheStore{client: client}, nil
}

func instrument(client *redis.Client, logger log.Logger) {
	if err := redisotel.InstrumentTracing(client); err != nil {
		logger.Errorf("failed to instrument redis tracing: %v", err)
	}
	if err := redisotel.InstrumentMetrics(client); err != nil {
		logger.Errorf("failed to instrument redis metrics: %v", err)
	}
}

func (c *cacheStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *cacheStore) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, payload, ttl).Err()
}

func (c *cacheStore) Get(ctx context.Context, key, defaultValue string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

func (c *cacheStore) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	payload, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(payload, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *cacheStore) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Unlink(ctx, keys...).Err()
}

func (c *cacheStore) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

func (c *cacheStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *cacheStore) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *cacheStore) Close() error {
	return c.client.Close()
}
