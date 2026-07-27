package postgres

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/zoom/darsly/internal/pkg/config"
)

// RunMigrations applies all pending up-migrations from migrationsPath.
// migrationsPath should be a relative or absolute directory, e.g. "migrations".
// Returns nil when no migrations are pending (ErrNoChange is silenced).
func RunMigrations(cfg config.PostgresConfig, migrationsPath string) error {
	dbURL := buildMigrateURL(cfg)
	sourceURL := "file://" + migrationsPath

	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate.Up: %w", err)
	}
	return nil
}

func buildMigrateURL(cfg config.PostgresConfig) string {
	// If DATABASE_URL is set, convert its scheme to pgx5://
	if raw := os.Getenv("DATABASE_URL"); raw != "" {
		u, err := url.Parse(raw)
		if err == nil {
			u.Scheme = "pgx5"
			return u.String()
		}
	}
	u := &url.URL{
		Scheme: "pgx5",
		User:   url.UserPassword(cfg.Username, cfg.Password),
		Host:   cfg.Host + ":" + cfg.Port,
		Path:   "/" + cfg.Database,
	}
	return u.String()
}
