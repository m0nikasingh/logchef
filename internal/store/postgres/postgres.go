package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/internal/store/postgres/sqlc"
)

// Compile-time assertion that *DB satisfies store.Store.
var _ store.Store = (*DB)(nil)

//go:embed migrations/*.sql
var migrationFS embed.FS

// DB is the Postgres-backed implementation of store.Store.
// It mirrors internal/store/sqlite.DB but uses a single pool (no read/write split).
type DB struct {
	db      *sql.DB
	queries sqlc.Querier
	log     *slog.Logger
}

// Config is the runtime configuration consumed by New.
//
// Task 25 will add a `config.PostgresConfig` (under internal/config) with koanf
// tags and defaults, and the bootstrap layer will translate it into this struct.
// We keep this type backend-local so the postgres package does not depend on
// internal/config and can be exercised in isolation by the contract suite.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Options carries the constructor inputs.
type Options struct {
	Logger *slog.Logger
	Config Config
}

// New opens the Postgres pool, runs migrations under an advisory lock,
// and returns a ready-to-use *DB.
func New(opts Options) (*DB, error) {
	if opts.Logger == nil {
		return nil, errors.New("postgres: Options.Logger is required")
	}
	if opts.Config.DSN == "" {
		return nil, errors.New("postgres: Options.Config.DSN is required")
	}

	db, err := sql.Open("pgx", opts.Config.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}
	db.SetMaxOpenConns(opts.Config.MaxOpenConns)
	db.SetMaxIdleConns(opts.Config.MaxIdleConns)
	db.SetConnMaxLifetime(opts.Config.ConnMaxLifetime)

	// Bound the startup ping so an unreachable DB does not stall boot up to
	// the driver default (~75s for pgx via several connect retries).
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	if err := runMigrations(db, opts.Logger); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres: migrate: %w", err)
	}

	return &DB{
		db:      db,
		queries: sqlc.New(db),
		log:     opts.Logger,
	}, nil
}

func runMigrations(db *sql.DB, log *slog.Logger) error {
	driver, err := migratepg.WithInstance(db, &migratepg.Config{
		// Default options: golang-migrate's postgres driver uses pg_advisory_lock()
		// keyed on a schema-derived hash, so concurrent replicas serialize safely.
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("driver: %w", err)
	}

	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("up: %w", err)
	}
	log.Info("postgres migrations applied")
	return nil
}

// Close closes the underlying pool.
func (db *DB) Close() error {
	return db.db.Close()
}

// BeginWriteTx mirrors the sqlite signature so existing callers compile unchanged.
// Postgres does not need a separate write lane; any transaction is fine.
func (db *DB) BeginWriteTx(ctx context.Context) (*sql.Tx, error) {
	return db.db.BeginTx(ctx, nil)
}

// WriteQueriesWithTx returns sqlc queries bound to the given Tx.
func (db *DB) WriteQueriesWithTx(tx *sql.Tx) *sqlc.Queries {
	return sqlc.New(tx)
}
