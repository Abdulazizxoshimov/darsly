package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zoom/darsly/internal/pkg/config"
)

// Cache is the interface for all Redis operations used in darsly.
type Cache interface {
	// Basic key-value
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Ping(ctx context.Context) error
	Close() error

	// Key scanning
	Scan(ctx context.Context, cursor uint64, match string, count int64) (keys []string, next uint64, err error)
	ScanDel(ctx context.Context, pattern string) error

	// Hash
	HSet(ctx context.Context, key string, values map[string]any, ttl time.Duration) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error

	// Pub/Sub (real-time notifications via WebSocket)
	Publish(ctx context.Context, channel string, payload any) error
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub

	// Distributed lock
	AcquireLock(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key, value string) error

	// Raw client (use only when the interface doesn't cover a use case)
	Client() *redis.Client
}

type redisCache struct {
	client *redis.Client
}

// New creates a Redis client from config and verifies connectivity.
func New(cfg config.RedisConfig) (Cache, error) {
	// Pool/timeout sozlamalari ikki rejimda ham bir xil.
	const (
		dialTimeout  = 5 * time.Second
		readTimeout  = 3 * time.Second
		writeTimeout = 3 * time.Second
		poolTimeout  = 4 * time.Second
	)

	var client *redis.Client
	if len(cfg.SentinelAddrs) > 0 {
		// HA rejim (audit R2 P2): 2+ host bo'lganda Sentinel failover. Bitta-host
		// deployda REDIS_SENTINEL_ADDRS bo'sh — bu shox umuman ishlamaydi.
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.SentinelMaster,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
			PoolSize:      cfg.PoolSize,
			MinIdleConns:  cfg.MinIdleConns,
			DialTimeout:   dialTimeout,
			ReadTimeout:   readTimeout,
			WriteTimeout:  writeTimeout,
			PoolTimeout:   poolTimeout,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     cfg.Addr(),
			Password: cfg.Password,
			DB:       cfg.DB,
			// Pool: yuqori concurrency'da default yetmasligi mumkin — aniq beramiz
			// (0 bo'lsa go-redis default: PoolSize=10×GOMAXPROCS).
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			// Timeoutlar — osilgan Redis so'rovi request goroutine'ini abadiy ushlab qolmasin.
			DialTimeout:  dialTimeout,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			PoolTimeout:  poolTimeout,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping failed: %w", err)
	}

	return &redisCache{client: client}, nil
}
