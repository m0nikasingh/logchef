package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/koanf/v2"
)

// minimalValidTOML is the smallest config payload that passes Load's
// required-field validation (admin_emails, api_token_secret, OIDC block).
// Individual tests append database/postgres stanzas as needed.
const minimalValidTOML = `
[auth]
admin_emails = ["admin@example.com"]
api_token_secret = "0123456789abcdef0123456789abcdef"

[oidc]
provider_url = "https://issuer.example.com"
auth_url = "https://issuer.example.com/auth"
token_url = "https://issuer.example.com/token"
client_id = "client"
redirect_url = "https://app.example.com/callback"
`

// writeConfig writes a TOML file in t.TempDir() and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "logchef.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// clearLogchefEnv unsets every LOGCHEF_* var so a stray host env does not
// leak into the test (e.g. an exported LOGCHEF_AUTH__ADMIN_EMAILS).
func clearLogchefEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 && strings.HasPrefix(kv[:i], envPrefix) {
			t.Setenv(kv[:i], "")
			_ = os.Unsetenv(kv[:i])
		}
	}
}

// TestDatabaseDriverDefault verifies applyDefaults assigns sqlite when no
// driver is configured. We exercise applyDefaults directly to keep the test
// independent of the file-loading and validation pipeline.
func TestDatabaseDriverDefault(t *testing.T) {
	k := koanf.New(".")
	var cfg Config
	applyDefaults(k, &cfg)

	if cfg.Database.Driver != databaseDriverSQLite {
		t.Fatalf("Database.Driver = %q, want %q", cfg.Database.Driver, databaseDriverSQLite)
	}
}

// TestDatabaseDriverUnknownRejected verifies Load fails fast when an
// unsupported driver is configured.
func TestDatabaseDriverUnknownRejected(t *testing.T) {
	clearLogchefEnv(t)
	path := writeConfig(t, minimalValidTOML+`
[database]
driver = "mysql"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown driver, got nil")
	}
	if !strings.Contains(err.Error(), "unknown database.driver") {
		t.Fatalf("error %q does not contain %q", err.Error(), "unknown database.driver")
	}
}

// TestPostgresRequiresDSN verifies postgres without a DSN is rejected.
func TestPostgresRequiresDSN(t *testing.T) {
	clearLogchefEnv(t)
	path := writeConfig(t, minimalValidTOML+`
[database]
driver = "postgres"
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing postgres.dsn, got nil")
	}
	if !strings.Contains(err.Error(), "requires postgres.dsn") {
		t.Fatalf("error %q does not contain %q", err.Error(), "requires postgres.dsn")
	}
}

// TestPostgresEnvOverride verifies LOGCHEF_* env vars set the driver and DSN
// even when the config file has no database/postgres stanza.
func TestPostgresEnvOverride(t *testing.T) {
	clearLogchefEnv(t)
	t.Setenv("LOGCHEF_DATABASE__DRIVER", "postgres")
	t.Setenv("LOGCHEF_POSTGRES__DSN", "postgres://localhost/db?sslmode=disable")

	path := writeConfig(t, minimalValidTOML)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Database.Driver != "postgres" {
		t.Fatalf("Database.Driver = %q, want %q", cfg.Database.Driver, "postgres")
	}
	if cfg.Postgres.DSN != "postgres://localhost/db?sslmode=disable" {
		t.Fatalf("Postgres.DSN = %q, want %q", cfg.Postgres.DSN, "postgres://localhost/db?sslmode=disable")
	}
	// Sanity-check that the pool defaults were applied for the postgres branch.
	if cfg.Postgres.MaxOpenConns != defaultPostgresMaxOpenConns {
		t.Errorf("Postgres.MaxOpenConns = %d, want %d", cfg.Postgres.MaxOpenConns, defaultPostgresMaxOpenConns)
	}
	if cfg.Postgres.ConnMaxLifetime != defaultPostgresConnMaxLife {
		t.Errorf("Postgres.ConnMaxLifetime = %s, want %s", cfg.Postgres.ConnMaxLifetime, defaultPostgresConnMaxLife)
	}
}
