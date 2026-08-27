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

// CacheStoreMethods is the cache surface the rest of the service depends on.
// Keeping it an interface lets services and middlewares be unit-tested with a
// fake instead of a live Redis.
type CacheStoreMethods interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// SetNX stores value only when key is absent and reports whether it won.
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	// SetJSON marshals value to JSON before storing it.
	SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error
	// Get returns the value at key, or defaultValue when the key is missing.
	Get(ctx context.Context, key, defaultValue string) (string, error)
	// GetJSON unmarshals the value at key into dest and reports whether the key existed.
	GetJSON(ctx context.Context, key string, dest any) (bool, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, keys ...string) error
	// DeletePattern removes every key matching a Redis glob pattern, scanning in batches.
	DeletePattern(ctx context.Context, pattern string) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
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
	// PoolSize caps the socket pool. A zero value uses defaultPoolSize.
	PoolSize int
}

type cacheStore struct {
	client *redis.Client
}

// NewRedisClient dials Redis, verifies the connection and instruments the
// client with OpenTelemetry tracing and metrics.
//
// Managed Redis offerings (ElastiCache, MemoryDB, Upstash) drop idle
// connections after roughly 20 minutes, so the pool keeps its own idle window
// comfortably below that.
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

func (c *cacheStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, ttl).Result()
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

func (c *cacheStore) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	return count > 0, err
}

func (c *cacheStore) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Unlink(ctx, keys...).Err()
}

func (c *cacheStore) DeletePattern(ctx context.Context, pattern string) error {
	const batchSize = 500

	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := c.Delete(ctx, keys...); err != nil {
				return err
			}
		}

		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func (c *cacheStore) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

func (c *cacheStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *cacheStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

func (c *cacheStore) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *cacheStore) Close() error {
	return c.client.Close()
}
