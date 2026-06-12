package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// testCollections exercises collection CRUD plus member and item
// management. GetCollectionMember and GetPersonalCollection return
// raw sql.ErrNoRows on absence (non-standard sentinel), so this suite
// avoids asserting their not-found error shape -- it only asserts
// the happy paths for those methods.
func testCollections(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "Coll User",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser(%s): %v", email, err)
		}
		return u.ID
	}

	t.Run("CreateGet", func(t *testing.T) {
		owner := seedUser(t, "coll-create@example.com")
		c, err := s.CreateCollection(ctx, "coll-create", "first", false, owner)
		if err != nil {
			t.Fatalf("CreateCollection: %v", err)
		}
		if c.ID == 0 {
			t.Fatalf("CreateCollection did not populate ID")
		}

		got, err := s.GetCollection(ctx, c.ID)
		if err != nil {
			t.Fatalf("GetCollection(%d): %v", c.ID, err)
		}
		if got.Name != "coll-create" || got.Description != "first" || got.IsPersonal {
			t.Errorf("GetCollection mismatch: got %+v", got)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetCollection(ctx, 999999)
		if err == nil {
			t.Fatalf("GetCollection(bogus): expected error, got nil")
		}
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetCollection(bogus): want ErrNotFound, got %v", err)
		}
	})

	t.Run("UniquePersonalPerUser", func(t *testing.T) {
		uid := seedUser(t, "coll-personal@example.com")
		if _, err := s.CreateCollection(ctx, "personal", "", true, uid); err != nil {
			t.Fatalf("CreateCollection(personal #1): %v", err)
		}
		_, err := s.CreateCollection(ctx, "personal-2", "", true, uid)
		if err == nil {
			t.Fatalf("CreateCollection(personal #2): expected error, got nil")
		}
		if !store.IsUniqueConstraintError(err) {
			t.Errorf("CreateCollection(personal #2): want unique-constraint sentinel, got %v", err)
		}
	})

	t.Run("ListWithMembership", func(t *testing.T) {
		owner := seedUser(t, "coll-list-owner@example.com")
		member := seedUser(t, "coll-list-member@example.com")

		// Personal collection for the owner. Listing-for-user joins on
		// membership, so the owner row must also be added as a member.
		personal, err := s.CreateCollection(ctx, "personal-list", "", true, owner)
		if err != nil {
			t.Fatalf("CreateCollection(personal): %v", err)
		}
		if err := s.AddCollectionMember(ctx, personal.ID, owner, models.CollectionRoleOwner, nil); err != nil {
			t.Fatalf("AddCollectionMember(personal owner): %v", err)
		}
		// GetPersonalCollection happy-path.
		got, err := s.GetPersonalCollection(ctx, owner)
		if err != nil {
			t.Fatalf("GetPersonalCollection: %v", err)
		}
		if got.ID != personal.ID {
			t.Errorf("GetPersonalCollection id mismatch: got %d, want %d", got.ID, personal.ID)
		}

		// Shared collection with owner + member.
		shared, err := s.CreateCollection(ctx, "shared-list", "shared", false, owner)
		if err != nil {
			t.Fatalf("CreateCollection(shared): %v", err)
		}
		if err := s.AddCollectionMember(ctx, shared.ID, owner, models.CollectionRoleOwner, nil); err != nil {
			t.Fatalf("AddCollectionMember(shared owner): %v", err)
		}
		addedBy := owner
		if err := s.AddCollectionMember(ctx, shared.ID, member, models.CollectionRoleMember, &addedBy); err != nil {
			t.Fatalf("AddCollectionMember(shared member): %v", err)
		}

		// Owner sees both collections.
		ownerList, err := s.ListCollectionsForUser(ctx, owner)
		if err != nil {
			t.Fatalf("ListCollectionsForUser(owner): %v", err)
		}
		if len(ownerList) != 2 {
			t.Errorf("ListCollectionsForUser(owner): got %d, want 2", len(ownerList))
		}

		// Member sees just the shared one.
		memberList, err := s.ListCollectionsForUser(ctx, member)
		if err != nil {
			t.Fatalf("ListCollectionsForUser(member): %v", err)
		}
		if len(memberList) != 1 || memberList[0].ID != shared.ID {
			t.Errorf("ListCollectionsForUser(member): got %+v, want only shared %d", memberList, shared.ID)
		}

		// GetCollectionMember + ListCollectionMembers round-trip.
		gm, err := s.GetCollectionMember(ctx, shared.ID, member)
		if err != nil {
			t.Fatalf("GetCollectionMember: %v", err)
		}
		if gm.UserID != member || gm.Role != models.CollectionRoleMember {
			t.Errorf("GetCollectionMember: got %+v", gm)
		}
		members, err := s.ListCollectionMembers(ctx, shared.ID)
		if err != nil {
			t.Fatalf("ListCollectionMembers: %v", err)
		}
		if len(members) != 2 {
			t.Errorf("ListCollectionMembers: got %d, want 2", len(members))
		}
	})

	t.Run("ItemsAndUpdateDelete", func(t *testing.T) {
		owner := seedUser(t, "coll-items@example.com")
		c, err := s.CreateCollection(ctx, "coll-items", "before", false, owner)
		if err != nil {
			t.Fatalf("CreateCollection: %v", err)
		}
		if err := s.AddCollectionMember(ctx, c.ID, owner, models.CollectionRoleOwner, nil); err != nil {
			t.Fatalf("AddCollectionMember(owner): %v", err)
		}

		// Source + saved query so we have a real item to link.
		src := makeSource("coll-items-src", "coll_db", "coll_items")
		if err := s.CreateSource(ctx, src); err != nil {
			t.Fatalf("CreateSource: %v", err)
		}
		ownerID := owner
		sq, err := s.CreateSavedQuery(ctx, src.ID, nil,
			"coll-items-q", "", string(models.SavedQueryTypeSQL),
			`{"version":1,"sourceId":1,"content":"SELECT 1"}`, &ownerID)
		if err != nil {
			t.Fatalf("CreateSavedQuery: %v", err)
		}

		addedBy := owner
		if err := s.AddCollectionItem(ctx, c.ID, sq.ID, 0, &addedBy); err != nil {
			t.Fatalf("AddCollectionItem: %v", err)
		}
		items, err := s.ListCollectionItems(ctx, c.ID)
		if err != nil {
			t.Fatalf("ListCollectionItems: %v", err)
		}
		if len(items) != 1 || items[0].Query.ID != sq.ID {
			t.Errorf("ListCollectionItems: got %+v, want one item for query %d", items, sq.ID)
		}

		if err := s.UpdateCollection(ctx, c.ID, "coll-items-renamed", "after"); err != nil {
			t.Fatalf("UpdateCollection: %v", err)
		}
		got, err := s.GetCollection(ctx, c.ID)
		if err != nil {
			t.Fatalf("GetCollection after update: %v", err)
		}
		if got.Name != "coll-items-renamed" || got.Description != "after" {
			t.Errorf("UpdateCollection not observed: got %+v", got)
		}

		if err := s.RemoveCollectionItem(ctx, c.ID, sq.ID); err != nil {
			t.Fatalf("RemoveCollectionItem: %v", err)
		}
		items, err = s.ListCollectionItems(ctx, c.ID)
		if err != nil {
			t.Fatalf("ListCollectionItems after remove: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("ListCollectionItems after remove: got %d, want 0", len(items))
		}

		if err := s.RemoveCollectionMember(ctx, c.ID, owner); err != nil {
			t.Fatalf("RemoveCollectionMember: %v", err)
		}

		if err := s.DeleteCollection(ctx, c.ID); err != nil {
			t.Fatalf("DeleteCollection: %v", err)
		}
		_, err = s.GetCollection(ctx, c.ID)
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetCollection after delete: want ErrNotFound, got %v", err)
		}
	})
}
