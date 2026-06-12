package sqlite_test

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/mr-karan/logchef/internal/config"
	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/internal/store/sqlite"
	"github.com/mr-karan/logchef/internal/store/storetest"
)

// TestContract runs the backend-agnostic store contract against a fresh
// on-disk SQLite database per top-level subtest. This proves the SQLite
// backend satisfies the contract before the postgres backend is added.
func TestContract(t *testing.T) {
	factory := func(t *testing.T) store.Store {
		t.Helper()
		dbPath := filepath.Join(t.TempDir(), "contract.db")
		db, err := sqlite.New(sqlite.Options{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			Config: config.SQLiteConfig{Path: dbPath},
		})
		if err != nil {
			t.Fatalf("sqlite.New failed: %v", err)
		}
		t.Cleanup(func() {
			if cerr := db.Close(); cerr != nil {
				t.Logf("sqlite close: %v", cerr)
			}
		})
		return db
	}
	storetest.Run(t, factory)
}
