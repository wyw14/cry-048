//go:build integration

// Package integration provides optional integration tests against a live PostgreSQL instance.
// Run with: go test -tags=integration ./tests/integration/...
// Requires POSTGRES_DSN environment variable pointing to a writable database with the schema applied.
package integration

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDatabaseConnection(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = ctx
	// A real integration test would open a pgxpool connection here and exercise
	// the postgres-backed repository implementation. This is intentionally a
	// smoke test since the default runtime mode is memory.
	t.Log("integration test scaffold requires a live PostgreSQL instance")
}
