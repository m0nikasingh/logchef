package storetest

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/mr-karan/logchef/pkg/models"
)

// testQueryShares covers the query-share happy path: create, get,
// touch (updates last_accessed_at), delete. Both backends currently
// return raw sql.ErrNoRows from GetQueryShare on miss, so the post-
// delete assertion checks for that sentinel.
func testQueryShares(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "Share User",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%s): %v", email, err)
		}
		return u.ID
	}

	// seedSource inserts a source with a unique (database, table) pair.
	seedSource := func(t *testing.T, name, database, table string) *models.Source {
		t.Helper()
		src := makeSource(name, database, table)
		if err := s.CreateSource(ctx, src); err != nil {
			t.Fatalf("CreateSource(%s): %v", name, err)
		}
		return src
	}

	t.Run("CreateGetTouchDelete", func(t *testing.T) {
		uid := seedUser(t, "share-roundtrip@example.com")
		src := seedSource(t, "share-roundtrip", "qs_db", "qs_roundtrip")

		now := time.Now().UTC().Truncate(time.Second)
		share := &models.QueryShare{
			Token:     "share-token-1",
			SourceID:  src.ID,
			CreatedBy: uid,
			Payload:   []byte(`{"version":1,"mode":"sql","query":"SELECT 1","limit":100}`),
			ExpiresAt: now.Add(1 * time.Hour),
			CreatedAt: now,
		}
		if err := s.CreateQueryShare(ctx, share); err != nil {
			t.Fatalf("CreateQueryShare: %v", err)
		}

		got, err := s.GetQueryShare(ctx, share.Token)
		if err != nil {
			t.Fatalf("GetQueryShare(%s): %v", share.Token, err)
		}
		if got.Token != share.Token || got.SourceID != share.SourceID ||
			got.CreatedBy != share.CreatedBy {
			t.Errorf("GetQueryShare mismatch: got %+v", got)
		}
		if string(got.Payload) != string(share.Payload) {
			t.Errorf("GetQueryShare Payload: got %s, want %s", got.Payload, share.Payload)
		}
		if got.LastAccessedAt != nil {
			t.Errorf("GetQueryShare LastAccessedAt: want nil before touch, got %v", got.LastAccessedAt)
		}

		// Touch records the access timestamp.
		accessedAt := now.Add(5 * time.Minute)
		if err := s.TouchQueryShare(ctx, share.Token, accessedAt); err != nil {
			t.Fatalf("TouchQueryShare: %v", err)
		}
		gotAfterTouch, err := s.GetQueryShare(ctx, share.Token)
		if err != nil {
			t.Fatalf("GetQueryShare(after touch): %v", err)
		}
		if gotAfterTouch.LastAccessedAt == nil {
			t.Fatalf("GetQueryShare LastAccessedAt: want non-nil after touch, got nil")
		}
		if !gotAfterTouch.LastAccessedAt.Equal(accessedAt) {
			t.Errorf("GetQueryShare LastAccessedAt: got %v, want %v",
				gotAfterTouch.LastAccessedAt, accessedAt)
		}

		// Delete removes the row.
		if err := s.DeleteQueryShare(ctx, share.Token); err != nil {
			t.Fatalf("DeleteQueryShare: %v", err)
		}
		_, err = s.GetQueryShare(ctx, share.Token)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("GetQueryShare after delete: want sql.ErrNoRows, got %v", err)
		}
	})
}
