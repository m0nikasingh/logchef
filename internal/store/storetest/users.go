package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// testUsers exercises the user-related methods on store.Store. All
// subtests share a single freshly minted store and use unique emails so
// they do not collide on the email uniqueness constraint.
func testUsers(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	t.Run("CreateGet", func(t *testing.T) {
		u := &models.User{
			Email:    "alice@example.com",
			FullName: "Alice Example",
			Role:     models.UserRoleAdmin,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if u.ID == 0 {
			t.Fatalf("CreateUser did not populate ID")
		}

		got, err := s.GetUser(ctx, u.ID)
		if err != nil {
			t.Fatalf("GetUser(%d): %v", u.ID, err)
		}
		if got.Email != u.Email || got.FullName != u.FullName || got.Role != u.Role {
			t.Errorf("GetUser mismatch: got %+v, want email=%s name=%s role=%s",
				got, u.Email, u.FullName, u.Role)
		}

		// GetUserByEmail round-trips by the unique key as well.
		byEmail, err := s.GetUserByEmail(ctx, u.Email)
		if err != nil {
			t.Fatalf("GetUserByEmail(%s): %v", u.Email, err)
		}
		if byEmail.ID != u.ID {
			t.Errorf("GetUserByEmail id mismatch: got %d, want %d", byEmail.ID, u.ID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetUser(ctx, models.UserID(999999))
		if err == nil {
			t.Fatalf("GetUser(bogus): expected error, got nil")
		}
		if !errors.Is(err, store.ErrUserNotFound) {
			t.Errorf("GetUser(bogus): want ErrUserNotFound, got %v", err)
		}
	})

	t.Run("UniqueEmail", func(t *testing.T) {
		first := &models.User{
			Email:    "dup@example.com",
			FullName: "First",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, first); err != nil {
			t.Fatalf("CreateUser(first): %v", err)
		}
		second := &models.User{
			Email:    "dup@example.com",
			FullName: "Second",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		err := s.CreateUser(ctx, second)
		if err == nil {
			t.Fatalf("CreateUser(duplicate email): expected error, got nil")
		}
		if !store.IsUniqueConstraintError(err) {
			t.Errorf("CreateUser(duplicate email): want unique-constraint sentinel, got %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		// Capture pre-existing count from earlier subtests so we count only
		// the rows we add here.
		before, err := s.ListUsers(ctx)
		if err != nil {
			t.Fatalf("ListUsers(before): %v", err)
		}
		seed := []*models.User{
			{Email: "list1@example.com", FullName: "List One", Role: models.UserRoleMember, Status: models.UserStatusActive},
			{Email: "list2@example.com", FullName: "List Two", Role: models.UserRoleMember, Status: models.UserStatusActive},
		}
		for _, u := range seed {
			if err := s.CreateUser(ctx, u); err != nil {
				t.Fatalf("CreateUser(%s): %v", u.Email, err)
			}
		}

		after, err := s.ListUsers(ctx)
		if err != nil {
			t.Fatalf("ListUsers(after): %v", err)
		}
		if len(after) != len(before)+len(seed) {
			t.Errorf("ListUsers length: got %d, want %d", len(after), len(before)+len(seed))
		}
		emails := make(map[string]bool, len(after))
		for _, u := range after {
			emails[u.Email] = true
		}
		for _, u := range seed {
			if !emails[u.Email] {
				t.Errorf("ListUsers missing seeded email %s", u.Email)
			}
		}
	})

	t.Run("UpdateDelete", func(t *testing.T) {
		u := &models.User{
			Email:    "upd@example.com",
			FullName: "Before Update",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}

		u.FullName = "After Update"
		u.Role = models.UserRoleAdmin
		if err := s.UpdateUser(ctx, u); err != nil {
			t.Fatalf("UpdateUser: %v", err)
		}

		got, err := s.GetUser(ctx, u.ID)
		if err != nil {
			t.Fatalf("GetUser after update: %v", err)
		}
		if got.FullName != "After Update" || got.Role != models.UserRoleAdmin {
			t.Errorf("UpdateUser not observed: got name=%q role=%q", got.FullName, got.Role)
		}

		if err := s.DeleteUser(ctx, u.ID); err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}
		_, err = s.GetUser(ctx, u.ID)
		if !errors.Is(err, store.ErrUserNotFound) {
			t.Errorf("GetUser after delete: want ErrUserNotFound, got %v", err)
		}
	})
}
