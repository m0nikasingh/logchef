package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// testTeams exercises team CRUD and team-membership methods. The
// AddTeamMember path requires a real user row (the sqlite implementation
// pre-checks existence), so the membership subtest seeds one.
func testTeams(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	t.Run("CreateGet", func(t *testing.T) {
		team := &models.Team{Name: "alpha", Description: "alpha team"}
		if err := s.CreateTeam(ctx, team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}
		if team.ID == 0 {
			t.Fatalf("CreateTeam did not populate ID")
		}

		got, err := s.GetTeam(ctx, team.ID)
		if err != nil {
			t.Fatalf("GetTeam(%d): %v", team.ID, err)
		}
		if got.Name != team.Name || got.Description != team.Description {
			t.Errorf("GetTeam mismatch: got %+v, want name=%s desc=%s",
				got, team.Name, team.Description)
		}

		byName, err := s.GetTeamByName(ctx, team.Name)
		if err != nil {
			t.Fatalf("GetTeamByName(%s): %v", team.Name, err)
		}
		if byName.ID != team.ID {
			t.Errorf("GetTeamByName id mismatch: got %d, want %d", byName.ID, team.ID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetTeam(ctx, models.TeamID(999999))
		if err == nil {
			t.Fatalf("GetTeam(bogus): expected error, got nil")
		}
		if !errors.Is(err, store.ErrTeamNotFound) {
			t.Errorf("GetTeam(bogus): want ErrTeamNotFound, got %v", err)
		}
	})

	t.Run("UniqueName", func(t *testing.T) {
		first := &models.Team{Name: "dup-team", Description: "first"}
		if err := s.CreateTeam(ctx, first); err != nil {
			t.Fatalf("CreateTeam(first): %v", err)
		}
		second := &models.Team{Name: "dup-team", Description: "second"}
		err := s.CreateTeam(ctx, second)
		if err == nil {
			t.Fatalf("CreateTeam(duplicate name): expected error, got nil")
		}
		if !store.IsUniqueConstraintError(err) {
			t.Errorf("CreateTeam(duplicate name): want unique-constraint sentinel, got %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		before, err := s.ListTeams(ctx)
		if err != nil {
			t.Fatalf("ListTeams(before): %v", err)
		}
		seed := []*models.Team{
			{Name: "list-team-1"},
			{Name: "list-team-2"},
		}
		for _, team := range seed {
			if err := s.CreateTeam(ctx, team); err != nil {
				t.Fatalf("CreateTeam(%s): %v", team.Name, err)
			}
		}
		after, err := s.ListTeams(ctx)
		if err != nil {
			t.Fatalf("ListTeams(after): %v", err)
		}
		if len(after) != len(before)+len(seed) {
			t.Errorf("ListTeams length: got %d, want %d", len(after), len(before)+len(seed))
		}
		names := make(map[string]bool, len(after))
		for _, team := range after {
			names[team.Name] = true
		}
		for _, team := range seed {
			if !names[team.Name] {
				t.Errorf("ListTeams missing seeded name %s", team.Name)
			}
		}
	})

	t.Run("UpdateDeleteAndMembers", func(t *testing.T) {
		team := &models.Team{Name: "mutate-team", Description: "before"}
		if err := s.CreateTeam(ctx, team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}

		team.Description = "after"
		if err := s.UpdateTeam(ctx, team); err != nil {
			t.Fatalf("UpdateTeam: %v", err)
		}
		got, err := s.GetTeam(ctx, team.ID)
		if err != nil {
			t.Fatalf("GetTeam after update: %v", err)
		}
		if got.Description != "after" {
			t.Errorf("UpdateTeam not observed: got desc=%q", got.Description)
		}

		// Membership round-trip: a real user must exist for AddTeamMember
		// because the sqlite backend pre-checks existence.
		user := &models.User{
			Email:    "team-member@example.com",
			FullName: "Team Member",
			Role:     models.UserRoleMember,
			Status:   models.UserStatusActive,
		}
		if err := s.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser for membership: %v", err)
		}
		if err := s.AddTeamMember(ctx, team.ID, user.ID, models.TeamRoleMember); err != nil {
			t.Fatalf("AddTeamMember: %v", err)
		}
		member, err := s.GetTeamMember(ctx, team.ID, user.ID)
		if err != nil {
			t.Fatalf("GetTeamMember: %v", err)
		}
		if member == nil {
			t.Fatalf("GetTeamMember returned nil for inserted member")
		}
		if member.Role != models.TeamRoleMember {
			t.Errorf("GetTeamMember role: got %q, want %q", member.Role, models.TeamRoleMember)
		}
		members, err := s.ListTeamMembers(ctx, team.ID)
		if err != nil {
			t.Fatalf("ListTeamMembers: %v", err)
		}
		if len(members) != 1 || members[0].UserID != user.ID {
			t.Errorf("ListTeamMembers: got %+v, want one entry for user %d", members, user.ID)
		}
		if err := s.RemoveTeamMember(ctx, team.ID, user.ID); err != nil {
			t.Fatalf("RemoveTeamMember: %v", err)
		}

		// Delete the team and verify it's gone.
		if err := s.DeleteTeam(ctx, team.ID); err != nil {
			t.Fatalf("DeleteTeam: %v", err)
		}
		_, err = s.GetTeam(ctx, team.ID)
		if !errors.Is(err, store.ErrTeamNotFound) {
			t.Errorf("GetTeam after delete: want ErrTeamNotFound, got %v", err)
		}
	})
}
