package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements InvalidationCache using Redis.
// It leverages Redis's native TTL for automatic expiration.
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache creates a new Redis invalidation cache from a Redis client and a key prefix.
// prefix typicallyends with a colon.
func NewRedisCache(client *redis.Client, keyPrefix string) (*RedisCache, error) {
	return &RedisCache{
		client: client,
		prefix: keyPrefix,
	}, nil
}

// RedisConfig contains configuration options for Redis.
type RedisConfig struct {
	// Addr is the Redis server address (e.g., "localhost:6379")
	Addr string

	// Password is the Redis password (empty for no auth)
	Password string

	// DB is the Redis database number (0-15)
	DB int

	// KeyPrefix is prepended to all keys (default: "heimdall:invalidated:")
	// typically ends with a colon.
	KeyPrefix string
}

// NewRedis creates a new Redis invalidation cache.
func NewRedisFromConfig(cfg RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: failed to connect: %w", err)
	}

	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "heimdall:invalidated:"
	}

	return &RedisCache{
		client: client,
		prefix: prefix,
	}, nil
}


// Set marks a session ID as invalidated with the given TTL.
func (c *RedisCache) Set(sessionID string, ttl time.Duration) error {
	ctx := context.Background()
	key := c.prefix + sessionID

	err := c.client.Set(ctx, key, "1", ttl).Err()
	if err != nil {
		return fmt.Errorf("redis: failed to set key: %w", err)
	}
	return nil
}

// Exists returns true if the session ID has been invalidated.
func (c *RedisCache) Exists(sessionID string) (bool, error) {
	ctx := context.Background()
	key := c.prefix + sessionID

	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis: failed to check key: %w", err)
	}
	return result > 0, nil
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Delete removes an invalidation entry (useful for testing).
func (c *RedisCache) Delete(sessionID string) error {
	ctx := context.Background()
	key := c.prefix + sessionID

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis: failed to delete key: %w", err)
	}
	return nil
}
