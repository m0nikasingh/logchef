package storetest

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/mr-karan/logchef/internal/store/sqlite/sqlc"
	"github.com/mr-karan/logchef/pkg/models"
)

// testAPITokens exercises API token CRUD. The store.Store interface for
// API tokens leaks sqlc-generated types (sqlc.CreateAPITokenParams,
// sqlc.ApiToken, sqlc.DeleteAPITokenParams); this is a known concern
// tracked for Phase 3 cleanup but the contract suite tests the
// interface as it stands today.
//
// GetAPIToken returns models.ErrNotFound (not store.ErrNotFound) when
// the row is missing, so the not-found subtest asserts against the
// models package sentinel.
func testAPITokens(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "Token User",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%s): %v", email, err)
		}
		return u.ID
	}

	t.Run("CreateGet", func(t *testing.T) {
		uid := seedUser(t, "tok-create@example.com")
		id, err := s.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
			UserID:    int64(uid),
			Name:      "tok-create",
			TokenHash: "hash-create",
			Prefix:    "lc_create__",
			ExpiresAt: sql.NullTime{Time: time.Now().Add(24 * time.Hour), Valid: true},
			Scopes:    "[]",
		})
		if err != nil {
			t.Fatalf("CreateAPIToken: %v", err)
		}
		if id == 0 {
			t.Fatalf("CreateAPIToken did not return id")
		}

		tok, err := s.GetAPIToken(ctx, id)
		if err != nil {
			t.Fatalf("GetAPIToken(%d): %v", id, err)
		}
		if tok.ID != id || tok.UserID != int64(uid) || tok.Name != "tok-create" {
			t.Errorf("GetAPIToken mismatch: got %+v", tok)
		}

		byHash, err := s.GetAPITokenByHash(ctx, "hash-create")
		if err != nil {
			t.Fatalf("GetAPITokenByHash: %v", err)
		}
		if byHash.ID != id {
			t.Errorf("GetAPITokenByHash id mismatch: got %d, want %d", byHash.ID, id)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetAPIToken(ctx, 999999)
		if err == nil {
			t.Fatalf("GetAPIToken(bogus): expected error, got nil")
		}
		if !errors.Is(err, models.ErrNotFound) {
			t.Errorf("GetAPIToken(bogus): want models.ErrNotFound, got %v", err)
		}
	})

	t.Run("UniqueHash", func(t *testing.T) {
		uid := seedUser(t, "tok-dup@example.com")
		_, err := s.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
			UserID:    int64(uid),
			Name:      "tok-dup-1",
			TokenHash: "hash-dup",
			Prefix:    "lc_dup__",
			Scopes:    "[]",
		})
		if err != nil {
			t.Fatalf("CreateAPIToken(first): %v", err)
		}
		_, err = s.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
			UserID:    int64(uid),
			Name:      "tok-dup-2",
			TokenHash: "hash-dup",
			Prefix:    "lc_dup__",
			Scopes:    "[]",
		})
		if err == nil {
			t.Fatalf("CreateAPIToken(duplicate hash): expected error, got nil")
		}
	})

	t.Run("ListForUser", func(t *testing.T) {
		uid := seedUser(t, "tok-list@example.com")
		seed := []string{"tok-list-a", "tok-list-b"}
		for _, name := range seed {
			if _, err := s.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
				UserID:    int64(uid),
				Name:      name,
				TokenHash: "hash-" + name,
				Prefix:    "lc_list__",
				Scopes:    "[]",
			}); err != nil {
				t.Fatalf("CreateAPIToken(%s): %v", name, err)
			}
		}
		tokens, err := s.ListAPITokensForUser(ctx, int64(uid))
		if err != nil {
			t.Fatalf("ListAPITokensForUser: %v", err)
		}
		if len(tokens) != len(seed) {
			t.Errorf("ListAPITokensForUser length: got %d, want %d", len(tokens), len(seed))
		}
	})

	t.Run("UpdateLastUsedAndDelete", func(t *testing.T) {
		uid := seedUser(t, "tok-mutate@example.com")
		id, err := s.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
			UserID:    int64(uid),
			Name:      "tok-mutate",
			TokenHash: "hash-mutate",
			Prefix:    "lc_mutate_",
			Scopes:    "[]",
		})
		if err != nil {
			t.Fatalf("CreateAPIToken: %v", err)
		}

		if err := s.UpdateAPITokenLastUsed(ctx, id); err != nil {
			t.Fatalf("UpdateAPITokenLastUsed: %v", err)
		}
		got, err := s.GetAPIToken(ctx, id)
		if err != nil {
			t.Fatalf("GetAPIToken after UpdateLastUsed: %v", err)
		}
		if !got.LastUsedAt.Valid {
			t.Errorf("UpdateAPITokenLastUsed did not set last_used_at")
		}

		if err := s.DeleteAPIToken(ctx, sqlc.DeleteAPITokenParams{ID: id, UserID: int64(uid)}); err != nil {
			t.Fatalf("DeleteAPIToken: %v", err)
		}
		_, err = s.GetAPIToken(ctx, id)
		if !errors.Is(err, models.ErrNotFound) {
			t.Errorf("GetAPIToken after delete: want models.ErrNotFound, got %v", err)
		}
	})
}
