package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kaanevranportfolio/Commercium/pkg/config"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
)

// Redis wraps redis.Client with additional functionality
type Redis struct {
	*redis.Client
	logger *logger.Logger
}

// NewRedis creates a new Redis client
func NewRedis(cfg config.RedisConfig, log *logger.Logger) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.PoolSize,
		PoolTimeout:  cfg.PoolTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Redis connection established",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"pool_size", cfg.PoolSize,
	)

	return &Redis{
		Client: client,
		logger: log,
	}, nil
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	r.logger.Info("Closing Redis connection")
	return r.Client.Close()
}

// HealthCheck performs a health check on Redis
func (r *Redis) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Ping(ctx).Err()
}

// SetWithExpiration sets a key-value pair with expiration
func (r *Redis) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Set(ctx, key, value, expiration).Err()
}

// GetString gets a string value by key
func (r *Redis) GetString(ctx context.Context, key string) (string, error) {
	return r.Get(ctx, key).Result()
}

// Exists checks if a key exists
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	result := r.Client.Exists(ctx, key)
	if result.Err() != nil {
		return false, result.Err()
	}
	return result.Val() > 0, nil
}

// DeleteKeys deletes multiple keys
func (r *Redis) DeleteKeys(ctx context.Context, keys ...string) error {
	return r.Del(ctx, keys...).Err()
}

// SetHash sets a hash field
func (r *Redis) SetHash(ctx context.Context, key, field string, value interface{}) error {
	return r.HSet(ctx, key, field, value).Err()
}

// GetHash gets a hash field
func (r *Redis) GetHash(ctx context.Context, key, field string) (string, error) {
	return r.HGet(ctx, key, field).Result()
}

// GetAllHash gets all fields and values in a hash
func (r *Redis) GetAllHash(ctx context.Context, key string) (map[string]string, error) {
	return r.HGetAll(ctx, key).Result()
}

// AddToSet adds a member to a set
func (r *Redis) AddToSet(ctx context.Context, key string, members ...interface{}) error {
	return r.SAdd(ctx, key, members...).Err()
}

// RemoveFromSet removes a member from a set
func (r *Redis) RemoveFromSet(ctx context.Context, key string, members ...interface{}) error {
	return r.SRem(ctx, key, members...).Err()
}

// IsMemberOfSet checks if a member exists in a set
func (r *Redis) IsMemberOfSet(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.SIsMember(ctx, key, member).Result()
}

// GetSetMembers gets all members of a set
func (r *Redis) GetSetMembers(ctx context.Context, key string) ([]string, error) {
	return r.SMembers(ctx, key).Result()
}
