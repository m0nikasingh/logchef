package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// testSavedQueries exercises saved-query CRUD. The schema for saved_queries
// has no uniqueness constraint on (source_id, name), so this domain has
// no unique-constraint subtest. The list path requires the requesting
// user to reach the source via a team, so the List subtest wires up a
// team with the source attached and the user as a member.
func testSavedQueries(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "SQ User",
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

	t.Run("CreateGet", func(t *testing.T) {
		uid := seedUser(t, "sq-create@example.com")
		src := seedSource(t, "sq-create", "sq_db", "sq_create")

		owner := uid
		created, err := s.CreateSavedQuery(ctx, src.ID, nil,
			"create-1", "saved query", string(models.SavedQueryTypeSQL),
			`{"version":1,"sourceId":1,"content":"SELECT 1"}`, &owner)
		if err != nil {
			t.Fatalf("CreateSavedQuery: %v", err)
		}
		if created.ID == 0 {
			t.Fatalf("CreateSavedQuery did not populate ID")
		}

		got, err := s.GetSavedQuery(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetSavedQuery(%d): %v", created.ID, err)
		}
		if got.Name != "create-1" || got.SourceID != src.ID || got.CreatedBy == nil || *got.CreatedBy != uid {
			t.Errorf("GetSavedQuery mismatch: got %+v, want name=create-1 source=%d owner=%d",
				got, src.ID, uid)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetSavedQuery(ctx, 999999)
		if err == nil {
			t.Fatalf("GetSavedQuery(bogus): expected error, got nil")
		}
		if !errors.Is(err, store.ErrQueryNotFound) {
			t.Errorf("GetSavedQuery(bogus): want ErrQueryNotFound, got %v", err)
		}
	})

	t.Run("ListForUserBySource", func(t *testing.T) {
		uid := seedUser(t, "sq-list@example.com")
		src := seedSource(t, "sq-list", "sq_db", "sq_list")
		owner := uid

		// Wire user -> team -> source so ListSavedQueriesForUser* sees them.
		team := &models.Team{Name: "sq-list-team"}
		if err := s.CreateTeam(ctx, team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}
		if err := s.AddTeamMember(ctx, team.ID, uid, models.TeamRoleMember); err != nil {
			t.Fatalf("AddTeamMember: %v", err)
		}
		if err := s.AddTeamSource(ctx, team.ID, src.ID); err != nil {
			t.Fatalf("AddTeamSource: %v", err)
		}

		seedNames := []string{"sq-list-a", "sq-list-b"}
		for _, name := range seedNames {
			if _, err := s.CreateSavedQuery(ctx, src.ID, nil,
				name, "", string(models.SavedQueryTypeSQL),
				`{"version":1,"sourceId":1,"content":"SELECT 1"}`, &owner); err != nil {
				t.Fatalf("CreateSavedQuery(%s): %v", name, err)
			}
		}

		// By-source listing.
		bySource, err := s.ListSavedQueriesForUserBySource(ctx, uid, src.ID)
		if err != nil {
			t.Fatalf("ListSavedQueriesForUserBySource: %v", err)
		}
		if len(bySource) != len(seedNames) {
			t.Errorf("ListSavedQueriesForUserBySource length: got %d, want %d", len(bySource), len(seedNames))
		}

		// Cross-source listing.
		all, err := s.ListSavedQueriesForUser(ctx, uid)
		if err != nil {
			t.Fatalf("ListSavedQueriesForUser: %v", err)
		}
		names := make(map[string]bool, len(all))
		for _, q := range all {
			names[q.Name] = true
		}
		for _, name := range seedNames {
			if !names[name] {
				t.Errorf("ListSavedQueriesForUser missing seeded name %s", name)
			}
		}
	})

	t.Run("UpdateDelete", func(t *testing.T) {
		uid := seedUser(t, "sq-mutate@example.com")
		src := seedSource(t, "sq-mutate", "sq_db", "sq_mutate")
		owner := uid

		created, err := s.CreateSavedQuery(ctx, src.ID, nil,
			"sq-mutate", "before", string(models.SavedQueryTypeSQL),
			`{"version":1,"sourceId":1,"content":"SELECT 1"}`, &owner)
		if err != nil {
			t.Fatalf("CreateSavedQuery: %v", err)
		}

		if err := s.UpdateSavedQuery(ctx, created.ID,
			"sq-mutate-renamed", "after", string(models.SavedQueryTypeLogchefQL),
			`{"version":1,"sourceId":1,"content":"level=error"}`); err != nil {
			t.Fatalf("UpdateSavedQuery: %v", err)
		}
		got, err := s.GetSavedQuery(ctx, created.ID)
		if err != nil {
			t.Fatalf("GetSavedQuery after update: %v", err)
		}
		if got.Name != "sq-mutate-renamed" || got.Description != "after" ||
			got.QueryType != models.SavedQueryTypeLogchefQL {
			t.Errorf("UpdateSavedQuery not observed: got %+v", got)
		}

		if err := s.DeleteSavedQuery(ctx, created.ID); err != nil {
			t.Fatalf("DeleteSavedQuery: %v", err)
		}
		_, err = s.GetSavedQuery(ctx, created.ID)
		if !errors.Is(err, store.ErrQueryNotFound) {
			t.Errorf("GetSavedQuery after delete: want ErrQueryNotFound, got %v", err)
		}
	})
}
