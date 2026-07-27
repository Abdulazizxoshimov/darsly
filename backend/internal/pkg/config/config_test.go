package config_test

import (
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
