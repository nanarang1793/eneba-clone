// Package testdb gives integration tests a connection to a real Postgres.
//
// Tests that need the database call testdb.Open(t). It skips the test when
// TEST_DATABASE_URL isn't set, so `go test ./...` still runs the plain unit
// tests anywhere. The database must be migrated and seeded first; run all of
// it with `docker compose run --rm test` (see README).
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open connects to TEST_DATABASE_URL or skips the test.
func Open(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set: skipping database test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	var products int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM products`).Scan(&products); err != nil {
		t.Fatalf("database not migrated? %v", err)
	}
	if products == 0 {
		t.Fatal("database has no products: load the seed first")
	}
	return pool
}

// Exec runs SQL and fails the test on error.
func Exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

// Tag returns a short random string, so every test run creates rows with
// unique names and never clashes with seed data or another run.
func Tag() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
