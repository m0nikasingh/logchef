package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/internal/store/postgres"
	"github.com/mr-karan/logchef/internal/store/storetest"
)

// TestPostgresStore runs the backend-agnostic contract suite against a live
// Postgres instance. The suite is skipped when LOGCHEF_TEST_POSTGRES_DSN is
// unset so the package remains test-runnable in environments without Postgres
// (developer laptops, sandboxes without Docker). CI sets the env var.
func TestPostgresStore(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("LOGCHEF_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		t.Skip("set LOGCHEF_TEST_POSTGRES_DSN to enable Postgres contract suite")
	}

	storetest.Run(t, func(t *testing.T) store.Store {
		t.Helper()
		schemaDSN := perSchemaDSN(t, baseDSN)
		db, err := postgres.New(postgres.Options{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			Config: postgres.Config{
				DSN:             schemaDSN,
				MaxOpenConns:    5,
				MaxIdleConns:    2,
				ConnMaxLifetime: 5 * time.Minute,
			},
		})
		if err != nil {
			t.Fatalf("postgres.New: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		return db
	})
}

// perSchemaDSN creates a unique schema in the target database, registers a
// cleanup that drops it, and returns a DSN whose search_path points at that
// schema. This gives each test an isolated namespace without spinning a fresh
// database per test.
func perSchemaDSN(t *testing.T, baseDSN string) string {
	t.Helper()

	admin, err := sql.Open("pgx", baseDSN)
	if err != nil {
		t.Fatalf("open base DSN: %v", err)
	}
	defer admin.Close()

	// Include the OS PID so parallel `go test` invocations against the same
	// database (e.g. two CI shards sharing one Postgres) cannot collide on
	// the unix-nano portion if they start within the same nanosecond.
	schema := fmt.Sprintf("logchef_test_%d_pid%d", time.Now().UnixNano(), os.Getpid())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema %q: %v", schema, err)
	}

	t.Cleanup(func() {
		db, err := sql.Open("pgx", baseDSN)
		if err != nil {
			t.Logf("cleanup: open base DSN: %v", err)
			return
		}
		defer db.Close()
		if _, err := db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil && !errors.Is(err, sql.ErrNoRows) {
			t.Logf("cleanup: drop schema %q: %v", schema, err)
		}
	})

	u, err := url.Parse(baseDSN)
	if err != nil {
		t.Fatalf("parse base DSN: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}
