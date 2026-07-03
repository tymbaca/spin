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

func WaitForPostgres(ctx context.Context, port int) error {
	dsn := postgresDSN(port)
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
	return fmt.Errorf("postgres on port %d did not become ready within timeout", port)
}

func ping(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

func RunGoose(ctx context.Context, port int, dir string) error {
	if err := WaitForPostgres(ctx, port); err != nil {
		return err
	}

	dsn := postgresDSN(port)
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

func postgresDSN(port int) string {
	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port)
}
