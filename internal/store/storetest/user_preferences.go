package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// testUserPreferences exercises the per-user preferences JSON blob.
// The interface only exposes Get + Upsert; preferences are stored as
// an opaque JSON string with the application layer doing all parsing.
func testUserPreferences(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "Prefs User",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%s): %v", email, err)
		}
		return u.ID
	}

	t.Run("NotFoundBeforeUpsert", func(t *testing.T) {
		uid := seedUser(t, "prefs-missing@example.com")
		_, err := s.GetUserPreferencesJSON(ctx, uid)
		if err == nil {
			t.Fatalf("GetUserPreferencesJSON before upsert: expected error, got nil")
		}
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetUserPreferencesJSON before upsert: want ErrNotFound, got %v", err)
		}
	})

	t.Run("UpsertGet", func(t *testing.T) {
		uid := seedUser(t, "prefs-upsert@example.com")
		first := `{"theme":"dark","page_size":50}`
		if err := s.UpsertUserPreferencesJSON(ctx, uid, first); err != nil {
			t.Fatalf("UpsertUserPreferencesJSON(insert): %v", err)
		}
		got, err := s.GetUserPreferencesJSON(ctx, uid)
		if err != nil {
			t.Fatalf("GetUserPreferencesJSON: %v", err)
		}
		if got != first {
			t.Errorf("GetUserPreferencesJSON: got %q, want %q", got, first)
		}

		// Upsert again to verify the update path round-trips.
		second := `{"theme":"light","page_size":100}`
		if err := s.UpsertUserPreferencesJSON(ctx, uid, second); err != nil {
			t.Fatalf("UpsertUserPreferencesJSON(update): %v", err)
		}
		got, err = s.GetUserPreferencesJSON(ctx, uid)
		if err != nil {
			t.Fatalf("GetUserPreferencesJSON after update: %v", err)
		}
		if got != second {
			t.Errorf("GetUserPreferencesJSON after update: got %q, want %q", got, second)
		}
	})
}
