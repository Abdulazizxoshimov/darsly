package config_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/pkg/config"
)

func base() *config.Config {
	c := &config.Config{}
	c.JWT.Secret = "a-strong-production-secret-key-32chars!!"
	c.JWT.AccessTTL = 15 * time.Minute
	c.Postgres.Password = "pass"
	c.App.FrontendBaseURL = "https://darsly.uz"
	c.Redis.Password = "redispass"
	c.Minio.SecretKey = "a-strong-minio-secret"
	c.LiveKit.APIKey = "lkkey"
	c.LiveKit.APISecret = "a-strong-livekit-secret-value-0123456789"
	return c
}

func TestValidate_DevOK(t *testing.T) {
	c := base()
	c.App.Env = "development"
	require.NoError(t, c.Validate())
}

func TestValidate_WeakSecret(t *testing.T) {
	c := base()
	c.JWT.Secret = "short"
	require.Error(t, c.Validate(), "32 belgidan qisqa secret rad etilishi kerak")
}

func TestValidate_ProductionRejectsDefaultSecret(t *testing.T) {
	c := base()
	c.App.Env = "production"
	c.JWT.Secret = "dev_secret_change_me_least_32_chars_long_000"
	require.Error(t, c.Validate(), "production'da dev/default secret rad etilishi kerak")
}

func TestValidate_ProductionRequiresFrontendURL(t *testing.T) {
	c := base()
	c.App.Env = "production"
	c.App.FrontendBaseURL = ""
	require.Error(t, c.Validate(), "production'da FRONTEND_BASE_URL majburiy (WS Origin)")
}

func TestValidate_ProductionOK(t *testing.T) {
	c := base()
	c.App.Env = "production"
	require.NoError(t, c.Validate())
}

func TestValidate_ProductionRequiresLiveKitKeys(t *testing.T) {
	c := base()
	c.App.Env = "production"
	c.LiveKit.APIKey = ""
	require.Error(t, c.Validate(), "prod'da LIVEKIT_API_KEY majburiy")

	c = base()
	c.App.Env = "production"
	c.LiveKit.APISecret = ""
	require.Error(t, c.Validate(), "prod'da LIVEKIT_API_SECRET majburiy")

	c = base()
	c.LiveKit.APIKey, c.LiveKit.APISecret = "", ""
	require.NoError(t, c.Validate(), "dev'da LiveKit ixtiyoriy")
}

func TestLoad_CollectsParseWarnings(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "abc")
	t.Setenv("JWT_ACCESS_TTL", "15 daqiqa")
	c := config.Load()
	require.Len(t, c.Warnings, 2)
	require.Equal(t, 50, int(c.Postgres.MaxConns), "default saqlanadi")
}

func TestConfig_SecretsNotInJSON(t *testing.T) {
	c := base()
	c.Postgres.Password = "PGSECRET"
	c.Redis.Password = "REDSECRET"
	b, err := json.Marshal(c)
	require.NoError(t, err)
	for _, sec := range []string{"PGSECRET", "REDSECRET", c.JWT.Secret, c.LiveKit.APISecret, c.Minio.SecretKey} {
		require.NotContains(t, string(b), sec)
	}
}
