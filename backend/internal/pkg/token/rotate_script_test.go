package token

import (
	"context"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// Go'dagi sessiya tekshiruvi va rotateScript orasida sessiya bekor qilinsa
// (RevokeAllUserSessions poygasi), skript sessiya kalitini TIRILTIRMASLIGI kerak.
func TestRotateScript_DoesNotResurrectRevokedSession(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6399"
	}
	cli := goredis.NewClient(&goredis.Options{Addr: addr})
	ctx := context.Background()
	pctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	if err := cli.Ping(pctx).Err(); err != nil {
		t.Skipf("Redis mavjud emas: %v", err)
	}
	keys := []string{"rs:old", "rs:new", "rs:grace", "rs:sess"}
	cli.Del(ctx, keys...)
	defer cli.Del(ctx, keys...)
	require.NoError(t, cli.Set(ctx, "rs:old", "sid", time.Minute).Err())
	// Sessiya kaliti YO'Q (bekor qilingan).

	got, err := rotateScript.Run(ctx, cli, keys, "sid", int64(60), int64(1000), "{}", "user").Int64()
	require.NoError(t, err)
	require.Equal(t, int64(-1), got)

	n, _ := cli.Exists(ctx, "rs:sess", "rs:new", "rs:grace").Result()
	require.Zero(t, n, "bekor qilingan sessiya qayta tiklanmasligi kerak")
	require.Equal(t, int64(1), cli.Exists(ctx, "rs:old").Val(), "eski JTI tegilmaydi")

	// Sessiya bor bo'lsa oddiy rotatsiya ishlaydi.
	require.NoError(t, cli.Set(ctx, "rs:sess", "user", time.Minute).Err())
	got, err = rotateScript.Run(ctx, cli, keys, "sid", int64(60), int64(1000), "{}", "user").Int64()
	require.NoError(t, err)
	require.Equal(t, int64(1), got)
}
