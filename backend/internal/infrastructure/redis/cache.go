package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func (c *redisCache) Client() *redis.Client { return c.client }

// Close Redis ulanish pool'ini yopadi (graceful shutdown).
func (c *redisCache) Close() error { return c.client.Close() }

func (c *redisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// ─── Basic key-value ──────────────────────────────────────────────────────────

// Set — qiymatni saqlaydi. Struct/map/slice JSON'ga o'giriladi; SATR esa xom
// holda yoziladi.
//
// # Nega satr uchun istisno
//
// Avval hamma narsa `json.Marshal` dan o'tardi, `Get` esa xom satr qaytarardi —
// ya'ni `Set(k, "Ali")` + `Get(k)` juftligi `"Ali"` ni QO'SHTIRNOQ bilan
// qaytarardi. Bu jimgina buzuqlik edi: `joinlink.mentorName` keshidan o'qilgan
// ism foydalanuvchiga `"Ali"` ko'rinishida chiqardi (birinchi so'rovda to'g'ri,
// kesh-hit'da noto'g'ri — shuning uchun sinovda ham osongina o'tkazib
// yuborilardi).
//
// `testutil.FakeCache` boshidan shu — to'g'ri — xulqda edi, ya'ni testlar real
// Redis'da mavjud xatoni ko'rmasdi. Endi ikkalasi bir xil.
//
// Struct'lar uchun shartnoma o'zgarmadi: ularni oldindan marshal qilmang.
func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s, ok := value.(string); ok {
		return c.client.Set(ctx, key, s, ttl).Err()
	}
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

// incrExpireScript — INCR va TTL o'rnatishni ATOMIK bajaradi.
//
// Nega Lua (audit topilma #6 / infra H3): avval bu ikki alohida buyruq edi —
// `INCR` keyin faqat `n==1` da `Expire`. Ikkisi orasida timeout/crash bo'lsa
// yoki `Expire` xato bersa kalit TTL'siz qolardi va hisoblagich (`rl:login:*`,
// `loginfail:*`, `joinfail:*`) ABADIY o'sib, foydalanuvchi/IP doimiy 429/qulfda
// qolardi (faqat `redis-cli DEL` bilan chiqarib bo'lardi).
//
// Endi: har chaqiruvda INCR, va agar TTL o'rnatilmagan bo'lsa (PTTL < 0 →
// birinchi INCR yoki oldingi EXPIRE yiqilgani) qayta o'rnatiladi. Ya'ni TTL
// har doim kafolatlanadi.
var incrExpireScript = redis.NewScript(`
	local n = redis.call('INCR', KEYS[1])
	if redis.call('PTTL', KEYS[1]) < 0 then
		redis.call('PEXPIRE', KEYS[1], ARGV[1])
	end
	return n
`)

// Incr atomik ravishda kalitni bittaga oshiradi va TTL o'rnatilganini
// KAFOLATLAYDI (Lua). Rate-limit / brute-force hisoblagichlari uchun.
func (c *redisCache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	res, err := incrExpireScript.Run(ctx, c.client, []string{key}, ttl.Milliseconds()).Result()
	if err != nil {
		return 0, err
	}
	n, _ := res.(int64)
	return n, nil
}

func (c *redisCache) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, ttl).Result()
}

// ─── Key scanning ─────────────────────────────────────────────────────────────

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

// HDel hash'dan maydon(lar)ni o'chiradi. Mavjud bo'lmagan maydon xato EMAS
// (Redis 0 qaytaradi) — chaqiruvchi uchun "yo'q edi" va "o'chirildi" farqi yo'q.
func (c *redisCache) HDel(ctx context.Context, key string, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return c.client.HDel(ctx, key, fields...).Err()
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
