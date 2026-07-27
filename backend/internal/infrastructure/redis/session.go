package redis

import (
	"context"
	"time"
)

// StoreSession saves a session payload under the given sessionID key.
func (c *redisCache) StoreSession(ctx context.Context, sessionID, payload string, ttl time.Duration) error {
	return c.client.Set(ctx, "sess:"+sessionID, payload, ttl).Err()
}

// GetSession retrieves the session payload for the given sessionID.
// Returns ("", redis.Nil) if the session does not exist.
func (c *redisCache) GetSession(ctx context.Context, sessionID string) (string, error) {
	return c.client.Get(ctx, "sess:"+sessionID).Result()
}

// DeleteSession removes the session key, effectively logging the user out.
func (c *redisCache) DeleteSession(ctx context.Context, sessionID string) error {
	return c.client.Del(ctx, "sess:"+sessionID).Err()
}

// ExtendSession resets the TTL on an existing session key.
func (c *redisCache) ExtendSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	return c.client.Expire(ctx, "sess:"+sessionID, ttl).Err()
}
