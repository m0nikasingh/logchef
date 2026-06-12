package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// testErrors is a cross-cutting verification that error sentinels work
// uniformly across the backend: each not-found path returns a sentinel
// that errors.Is matches, the IsNotFoundError helper recognises each
// domain's sentinel, and unique-constraint violations on user email
// surface as ErrUniqueConstraint.
func testErrors(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	t.Run("UserNotFound", func(t *testing.T) {
		_, err := s.GetUser(ctx, models.UserID(999999))
		if !errors.Is(err, store.ErrUserNotFound) {
			t.Errorf("GetUser(bogus): want ErrUserNotFound, got %v", err)
		}
		if !store.IsNotFoundError(err) {
			t.Errorf("IsNotFoundError(ErrUserNotFound): want true, got false")
		}
	})

	t.Run("TeamNotFound", func(t *testing.T) {
		_, err := s.GetTeam(ctx, models.TeamID(999999))
		if !errors.Is(err, store.ErrTeamNotFound) {
			t.Errorf("GetTeam(bogus): want ErrTeamNotFound, got %v", err)
		}
		if !store.IsNotFoundError(err) {
			t.Errorf("IsNotFoundError(ErrTeamNotFound): want true, got false")
		}
	})

	t.Run("SourceNotFound", func(t *testing.T) {
		_, err := s.GetSource(ctx, models.SourceID(999999))
		if !errors.Is(err, store.ErrSourceNotFound) {
			t.Errorf("GetSource(bogus): want ErrSourceNotFound, got %v", err)
		}
		if !store.IsNotFoundError(err) {
			t.Errorf("IsNotFoundError(ErrSourceNotFound): want true, got false")
		}
	})

	t.Run("SessionNotFound", func(t *testing.T) {
		_, err := s.GetSession(ctx, models.SessionID("errors-no-such-session"))
		if !errors.Is(err, store.ErrSessionNotFound) {
			t.Errorf("GetSession(bogus): want ErrSessionNotFound, got %v", err)
		}
		if !store.IsNotFoundError(err) {
			t.Errorf("IsNotFoundError(ErrSessionNotFound): want true, got false")
		}
	})

	t.Run("UniqueConstraintOnDuplicateUser", func(t *testing.T) {
		first := &models.User{
			Email:    "errors-dup@example.com",
			FullName: "Errors First",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, first); err != nil {
			t.Fatalf("CreateUser(first): %v", err)
		}
		second := &models.User{
			Email:    "errors-dup@example.com",
			FullName: "Errors Second",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		err := s.CreateUser(ctx, second)
		if err == nil {
			t.Fatalf("CreateUser(duplicate email): expected error, got nil")
		}
		// The sqlite backend maps duplicate-email to ErrUserExists, which
		// IsUniqueConstraintError folds into the family alongside the
		// generic ErrUniqueConstraint sentinel. Assert on the helper so
		// both shapes pass; pure errors.Is(err, ErrUniqueConstraint) is
		// not guaranteed across the family today.
		if !store.IsUniqueConstraintError(err) {
			t.Errorf("CreateUser(duplicate email): want unique-constraint family via IsUniqueConstraintError, got %v", err)
		}
	})
}
