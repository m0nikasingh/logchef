// Package storetest is a backend-agnostic contract test suite for
// store.Store implementations. Both the SQLite and Postgres backends
// import this package from a thin _test.go shim that supplies a
// "factory" -- a function that returns a freshly initialised
// store.Store for each test case.
package storetest

import (
	"testing"

	"github.com/mr-karan/logchef/internal/store"
)

// Factory produces a fresh, empty store.Store for one test. The
// implementation MUST set t.Cleanup to release resources (close DB,
// drop schema, remove temp file).
type Factory func(t *testing.T) store.Store

// Run executes the full contract suite against the supplied factory.
// Each top-level subtest gets its own factory invocation, so the suite
// is safe to use with t.Parallel().
func Run(t *testing.T, newStore Factory) {
	t.Helper()
	t.Run("Users", func(t *testing.T) { testUsers(t, newStore) })
	t.Run("Teams", func(t *testing.T) { testTeams(t, newStore) })
	t.Run("Sources", func(t *testing.T) { testSources(t, newStore) })
	t.Run("Sessions", func(t *testing.T) { testSessions(t, newStore) })
	t.Run("SavedQueries", func(t *testing.T) { testSavedQueries(t, newStore) })
	t.Run("Alerts", func(t *testing.T) { testAlerts(t, newStore) })
	t.Run("APITokens", func(t *testing.T) { testAPITokens(t, newStore) })
	t.Run("Settings", func(t *testing.T) { testSettings(t, newStore) })
	t.Run("UserPreferences", func(t *testing.T) { testUserPreferences(t, newStore) })
	t.Run("Collections", func(t *testing.T) { testCollections(t, newStore) })
	t.Run("ExportJobs", func(t *testing.T) { testExportJobs(t, newStore) })
	t.Run("QueryShares", func(t *testing.T) { testQueryShares(t, newStore) })
	t.Run("Errors", func(t *testing.T) { testErrors(t, newStore) })
}
