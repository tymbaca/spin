package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	defaultTimeout = 30 * time.Second
	pollInterval   = 500 * time.Millisecond
)

type PostgresConfig struct {
	Port     int
	User     string
	Password string
	Database string
}

func (cfg PostgresConfig) withDefaults() PostgresConfig {
	if cfg.User == "" {
		cfg.User = "postgres"
	}
	if cfg.Password == "" {
		cfg.Password = "postgres"
	}
	if cfg.Database == "" {
		cfg.Database = "postgres"
	}
	return cfg
}

func (cfg PostgresConfig) dsn() string {
	cfg = cfg.withDefaults()
	return fmt.Sprintf(
		"postgres://%s:%s@127.0.0.1:%d/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Port, cfg.Database,
	)
}

func WaitForPostgres(ctx context.Context, cfg PostgresConfig) error {
	dsn := cfg.withDefaults().dsn()
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultTimeout)
	}

	for time.Now().Before(deadline) {
		if err := ping(dsn); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for postgres: %w", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
	return fmt.Errorf("postgres on port %d did not become ready within timeout", cfg.Port)
}

func ping(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

func RunGoose(ctx context.Context, cfg PostgresConfig, dir string) error {
	cfg = cfg.withDefaults()
	if err := WaitForPostgres(ctx, cfg); err != nil {
		return err
	}

	dsn := cfg.dsn()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, dir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	fmt.Printf("migrations applied from %s\n", dir)
	return nil
}
