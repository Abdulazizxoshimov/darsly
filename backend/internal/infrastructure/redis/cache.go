package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func (c *redisCache) Client() *redis.Client { return c.client }

func (c *redisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// ─── Basic key-value ──────────────────────────────────────────────────────────

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, string(b), ttl).Err()
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *redisCache) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Incr atomik ravishda kalitni bittaga oshiradi; birinchi oshirishda TTL o'rnatadi.
// Rate-limit / brute-force hisoblagichlari uchun.
func (c *redisCache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		_ = c.client.Expire(ctx, key, ttl).Err()
	}
	return n, nil
}

func (c *redisCache) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, ttl).Result()
}

// ─── Key scanning ─────────────────────────────────────────────────────────────

func (c *redisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

func (c *redisCache) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return c.client.Scan(ctx, cursor, match, count).Result()
}

func (c *redisCache) ScanDel(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// ─── Batch ────────────────────────────────────────────────────────────────────

func (c *redisCache) MGet(ctx context.Context, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	res, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	out := make([]string, len(res))
	for i, v := range res {
		if v != nil {
			out[i] = v.(string)
		}
	}
	return out, nil
}

// ─── Hash ─────────────────────────────────────────────────────────────────────

func (c *redisCache) HSet(ctx context.Context, key string, values map[string]any, ttl time.Duration) error {
	if err := c.client.HSet(ctx, key, values).Err(); err != nil {
		return err
	}
	if ttl > 0 {
		return c.client.Expire(ctx, key, ttl).Err()
	}
	return nil
}

func (c *redisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	res, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("redis: key not found: %s", key)
	}
	return res, nil
}

// ─── Pub/Sub ──────────────────────────────────────────────────────────────────

func (c *redisCache) Publish(ctx context.Context, channel string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.client.Publish(ctx, channel, b).Err()
}

func (c *redisCache) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

// ─── Distributed lock ─────────────────────────────────────────────────────────

func (c *redisCache) AcquireLock(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, ttl).Result()
}

// ReleaseLock deletes the lock only if the stored value matches — prevents
// releasing a lock that was re-acquired by another caller after TTL expiry.
var releaseLockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	end
	return 0
`)

func (c *redisCache) ReleaseLock(ctx context.Context, key, value string) error {
	return releaseLockScript.Run(ctx, c.client, []string{key}, value).Err()
}

// ─── Lua script ───────────────────────────────────────────────────────────────

func (c *redisCache) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return c.client.Eval(ctx, script, keys, args...).Result()
}
