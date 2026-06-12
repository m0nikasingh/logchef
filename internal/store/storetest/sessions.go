package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// testSessions exercises the session methods. Sessions reference users
// via FK, so each subtest seeds its own user. The interface exposes no
// Update or List for sessions; the "list-shape" coverage is provided by
// CountUserSessions.
func testSessions(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "Session User",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%s): %v", email, err)
		}
		return u.ID
	}

	t.Run("CreateGet", func(t *testing.T) {
		uid := seedUser(t, "session-create@example.com")
		sess := &models.Session{
			ID:        models.SessionID("sess-create-1"),
			UserID:    uid,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if err := s.CreateSession(ctx, sess); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		got, err := s.GetSession(ctx, sess.ID)
		if err != nil {
			t.Fatalf("GetSession(%s): %v", sess.ID, err)
		}
		if got.ID != sess.ID || got.UserID != sess.UserID {
			t.Errorf("GetSession mismatch: got %+v, want id=%s user=%d",
				got, sess.ID, sess.UserID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetSession(ctx, models.SessionID("does-not-exist"))
		if err == nil {
			t.Fatalf("GetSession(bogus): expected error, got nil")
		}
		if !errors.Is(err, store.ErrSessionNotFound) {
			t.Errorf("GetSession(bogus): want ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("UniqueID", func(t *testing.T) {
		uid := seedUser(t, "session-dup@example.com")
		first := &models.Session{
			ID:        models.SessionID("sess-dup-id"),
			UserID:    uid,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if err := s.CreateSession(ctx, first); err != nil {
			t.Fatalf("CreateSession(first): %v", err)
		}
		second := &models.Session{
			ID:        models.SessionID("sess-dup-id"),
			UserID:    uid,
			ExpiresAt: time.Now().Add(2 * time.Hour),
		}
		err := s.CreateSession(ctx, second)
		if err == nil {
			t.Fatalf("CreateSession(duplicate id): expected error, got nil")
		}
		if !store.IsUniqueConstraintError(err) {
			t.Errorf("CreateSession(duplicate id): want unique-constraint sentinel, got %v", err)
		}
	})

	t.Run("CountActive", func(t *testing.T) {
		uid := seedUser(t, "session-count@example.com")
		// Two active sessions, one already-expired session. CountUserSessions
		// only counts active (not-yet-expired) rows.
		sessions := []*models.Session{
			{ID: "sess-count-a", UserID: uid, ExpiresAt: time.Now().Add(1 * time.Hour)},
			{ID: "sess-count-b", UserID: uid, ExpiresAt: time.Now().Add(2 * time.Hour)},
			{ID: "sess-count-expired", UserID: uid, ExpiresAt: time.Now().Add(-1 * time.Hour)},
		}
		for _, sess := range sessions {
			if err := s.CreateSession(ctx, sess); err != nil {
				t.Fatalf("CreateSession(%s): %v", sess.ID, err)
			}
		}
		count, err := s.CountUserSessions(ctx, uid)
		if err != nil {
			t.Fatalf("CountUserSessions: %v", err)
		}
		if count != 2 {
			t.Errorf("CountUserSessions: got %d, want 2 (active only)", count)
		}
	})

	t.Run("DeleteAndDeleteUserSessions", func(t *testing.T) {
		uid := seedUser(t, "session-delete@example.com")
		one := &models.Session{
			ID:        models.SessionID("sess-del-1"),
			UserID:    uid,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		two := &models.Session{
			ID:        models.SessionID("sess-del-2"),
			UserID:    uid,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if err := s.CreateSession(ctx, one); err != nil {
			t.Fatalf("CreateSession(one): %v", err)
		}
		if err := s.CreateSession(ctx, two); err != nil {
			t.Fatalf("CreateSession(two): %v", err)
		}

		// DeleteSession removes just the one.
		if err := s.DeleteSession(ctx, one.ID); err != nil {
			t.Fatalf("DeleteSession: %v", err)
		}
		_, err := s.GetSession(ctx, one.ID)
		if !errors.Is(err, store.ErrSessionNotFound) {
			t.Errorf("GetSession after DeleteSession: want ErrSessionNotFound, got %v", err)
		}
		// The other one is still there.
		if _, err := s.GetSession(ctx, two.ID); err != nil {
			t.Errorf("GetSession(two) still active: %v", err)
		}

		// DeleteUserSessions wipes the rest.
		if err := s.DeleteUserSessions(ctx, uid); err != nil {
			t.Fatalf("DeleteUserSessions: %v", err)
		}
		_, err = s.GetSession(ctx, two.ID)
		if !errors.Is(err, store.ErrSessionNotFound) {
			t.Errorf("GetSession after DeleteUserSessions: want ErrSessionNotFound, got %v", err)
		}
		count, err := s.CountUserSessions(ctx, uid)
		if err != nil {
			t.Fatalf("CountUserSessions after wipe: %v", err)
		}
		if count != 0 {
			t.Errorf("CountUserSessions after wipe: got %d, want 0", count)
		}
	})
}
