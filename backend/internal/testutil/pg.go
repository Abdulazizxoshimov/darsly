package testutil

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/zoom/darsly/internal/pkg/config"
	"github.com/zoom/darsly/internal/pkg/postgres"
)

// perPackageDBName har test paketiga alohida baza nomi beradi (CWD hash bo'yicha).
// `go test ./...` paketlarni parallel ishlatadi — umumiy baza truncate bilan
// bir-biriga xalaqit beradi. TEST_DB_NAME o'rnatilsa (masalan -p 1 bilan), o'sha ishlatiladi.
func perPackageDBName() string {
	if v := os.Getenv("TEST_DB_NAME"); v != "" {
		return v
	}
	wd, _ := os.Getwd()
	sum := sha1.Sum([]byte(wd))
	return "darsly_test_" + hex.EncodeToString(sum[:])[:10]
}

// testDBParams — integration test Postgres ulanish parametrlari.
// Dev porti (5442) default; TEST_DB_* env bilan override qilinadi (CI: 5432).
func testDBParams() (host, port, user, pass, db string) {
	get := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	return get("TEST_DB_HOST", "localhost"),
		get("TEST_DB_PORT", "5442"),
		get("TEST_DB_USER", "postgres"),
		get("TEST_DB_PASSWORD", "postgres"),
		perPackageDBName()
}

func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0) // .../internal/testutil/pg.go
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

// SetupTestDB test uchun 'darsly_test' bazasini tayyorlaydi (yaratadi, migratsiya
// qiladi, jadvallarni tozalaydi) va ulangan poolni qaytaradi. Postgres mavjud
// bo'lmasa test skip qilinadi. Har chaqiruvda barcha jadvallarni truncate qiladi.
func SetupTestDB(t *testing.T) *postgres.Postgres {
	t.Helper()
	host, port, user, pass, db := testDBParams()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1) Maintenance 'postgres' bazasiga ulanib, test bazasini yaratamiz.
	admDSN := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=postgres sslmode=disable"
	adm, err := pgx.Connect(ctx, admDSN)
	if err != nil {
		t.Skipf("Postgres mavjud emas (%s:%s): %v", host, port, err)
	}
	if _, err := adm.Exec(ctx, "CREATE DATABASE "+db); err != nil && !strings.Contains(err.Error(), "already exists") {
		_ = adm.Close(ctx)
		t.Skipf("test bazasini yaratib bo'lmadi: %v", err)
	}
	_ = adm.Close(ctx)

	// 2) Config + pool.
	cfg := &config.Config{Postgres: config.PostgresConfig{
		Host: host, Port: port, Database: db, Username: user, Password: pass,
		MaxConns: 10, MinConns: 1,
		MaxConnIdleTime: time.Minute, MaxConnLifetime: time.Minute, HealthCheckPeriod: time.Minute,
	}}
	pg, err := postgres.New(context.Background(), cfg, NewLogger())
	if err != nil {
		t.Skipf("test bazasiga ulanib bo'lmadi: %v", err)
	}

	// 3) Migratsiyalar.
	if err := postgres.RunMigrations(cfg.Postgres, migrationsDir()); err != nil {
		t.Fatalf("migratsiya xatosi: %v", err)
	}

	// 4) Toza holat — barcha domen jadvallarini tozalaymiz.
	truncate(t, pg)

	t.Cleanup(func() { pg.DB.Close() })
	return pg
}

func truncate(t *testing.T, pg *postgres.Postgres) {
	t.Helper()
	_, err := pg.DB.Exec(context.Background(),
		`TRUNCATE recordings, waiting_room_requests, notifications, lessons, password_resets, refresh_tokens, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
