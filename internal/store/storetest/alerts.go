package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// makeAlert builds a minimal valid alert definition for one source.
func makeAlert(sourceID models.SourceID, name string, createdBy *models.UserID) *models.Alert {
	return &models.Alert{
		SourceID:          sourceID,
		Name:              name,
		Description:       "test alert",
		QueryType:         models.AlertQueryTypeSQL,
		Query:             "SELECT count() FROM logs",
		LookbackSeconds:   300,
		ThresholdOperator: models.AlertThresholdGreaterThan,
		ThresholdValue:    10,
		FrequencySeconds:  300,
		Severity:          models.AlertSeverityWarning,
		IsActive:          true,
		CreatedBy:         createdBy,
	}
}

// testAlerts exercises alert CRUD and the alert-history lifecycle. The
// alerts schema has no uniqueness constraint beyond the primary key, so
// this domain has no unique-constraint subtest; in its place the suite
// covers the history-insert / resolve flow.
func testAlerts(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "Alert User",
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
		uid := seedUser(t, "alert-create@example.com")
		src := seedSource(t, "alert-create", "al_db", "al_create")

		alert := makeAlert(src.ID, "alert-create-1", &uid)
		if err := s.CreateAlert(ctx, alert); err != nil {
			t.Fatalf("CreateAlert: %v", err)
		}
		if alert.ID == 0 {
			t.Fatalf("CreateAlert did not populate ID")
		}

		got, err := s.GetAlert(ctx, alert.ID)
		if err != nil {
			t.Fatalf("GetAlert(%d): %v", alert.ID, err)
		}
		if got.Name != "alert-create-1" || got.SourceID != src.ID || got.ThresholdValue != 10 {
			t.Errorf("GetAlert mismatch: got %+v", got)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetAlert(ctx, models.AlertID(999999))
		if err == nil {
			t.Fatalf("GetAlert(bogus): expected error, got nil")
		}
		// GetAlert maps sql.ErrNoRows to the generic store.ErrNotFound.
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetAlert(bogus): want ErrNotFound, got %v", err)
		}
	})

	t.Run("ListBySourceAndForUser", func(t *testing.T) {
		uid := seedUser(t, "alert-list@example.com")
		src := seedSource(t, "alert-list", "al_db", "al_list")

		// Wire user -> team -> source for ListAlertsForUser visibility.
		team := &models.Team{Name: "alert-list-team"}
		if err := s.CreateTeam(ctx, team); err != nil {
			t.Fatalf("CreateTeam: %v", err)
		}
		if err := s.AddTeamMember(ctx, team.ID, uid, models.TeamRoleMember); err != nil {
			t.Fatalf("AddTeamMember: %v", err)
		}
		if err := s.AddTeamSource(ctx, team.ID, src.ID); err != nil {
			t.Fatalf("AddTeamSource: %v", err)
		}

		seedNames := []string{"alert-list-a", "alert-list-b"}
		for _, name := range seedNames {
			a := makeAlert(src.ID, name, &uid)
			if err := s.CreateAlert(ctx, a); err != nil {
				t.Fatalf("CreateAlert(%s): %v", name, err)
			}
		}

		bySource, err := s.ListAlertsBySource(ctx, src.ID)
		if err != nil {
			t.Fatalf("ListAlertsBySource: %v", err)
		}
		if len(bySource) != len(seedNames) {
			t.Errorf("ListAlertsBySource length: got %d, want %d", len(bySource), len(seedNames))
		}

		forUser, err := s.ListAlertsForUser(ctx, uid)
		if err != nil {
			t.Fatalf("ListAlertsForUser: %v", err)
		}
		names := make(map[string]bool, len(forUser))
		for _, a := range forUser {
			names[a.Name] = true
		}
		for _, name := range seedNames {
			if !names[name] {
				t.Errorf("ListAlertsForUser missing %s", name)
			}
		}
	})

	t.Run("UpdateAndMarkEvaluated", func(t *testing.T) {
		uid := seedUser(t, "alert-mutate@example.com")
		src := seedSource(t, "alert-mutate", "al_db", "al_mutate")
		alert := makeAlert(src.ID, "alert-mutate", &uid)
		if err := s.CreateAlert(ctx, alert); err != nil {
			t.Fatalf("CreateAlert: %v", err)
		}

		alert.Description = "after"
		alert.ThresholdValue = 99
		if err := s.UpdateAlert(ctx, alert); err != nil {
			t.Fatalf("UpdateAlert: %v", err)
		}
		got, err := s.GetAlert(ctx, alert.ID)
		if err != nil {
			t.Fatalf("GetAlert after update: %v", err)
		}
		if got.Description != "after" || got.ThresholdValue != 99 {
			t.Errorf("UpdateAlert not observed: got desc=%q threshold=%v", got.Description, got.ThresholdValue)
		}

		if err := s.MarkAlertEvaluated(ctx, alert.ID); err != nil {
			t.Fatalf("MarkAlertEvaluated: %v", err)
		}
		if err := s.MarkAlertTriggered(ctx, alert.ID); err != nil {
			t.Fatalf("MarkAlertTriggered: %v", err)
		}
	})

	t.Run("HistoryInsertResolveList", func(t *testing.T) {
		uid := seedUser(t, "alert-history@example.com")
		src := seedSource(t, "alert-history", "al_db", "al_history")
		alert := makeAlert(src.ID, "alert-history", &uid)
		if err := s.CreateAlert(ctx, alert); err != nil {
			t.Fatalf("CreateAlert: %v", err)
		}

		value := 42.0
		entry, err := s.InsertAlertHistory(ctx, alert.ID, models.AlertStatusTriggered,
			&value, "triggered", map[string]any{"k": "v"})
		if err != nil {
			t.Fatalf("InsertAlertHistory: %v", err)
		}
		if entry.ID == 0 || entry.Status != models.AlertStatusTriggered {
			t.Errorf("InsertAlertHistory unexpected entry: %+v", entry)
		}

		latest, err := s.GetLatestUnresolvedAlertHistory(ctx, alert.ID)
		if err != nil {
			t.Fatalf("GetLatestUnresolvedAlertHistory: %v", err)
		}
		if latest.ID != entry.ID {
			t.Errorf("GetLatestUnresolvedAlertHistory id mismatch: got %d, want %d", latest.ID, entry.ID)
		}

		if err := s.ResolveAlertHistory(ctx, entry.ID, "resolved by test"); err != nil {
			t.Fatalf("ResolveAlertHistory: %v", err)
		}

		history, err := s.ListAlertHistory(ctx, alert.ID, 10)
		if err != nil {
			t.Fatalf("ListAlertHistory: %v", err)
		}
		if len(history) != 1 {
			t.Errorf("ListAlertHistory length: got %d, want 1", len(history))
		}
	})

	t.Run("ThresholdPrecision", func(t *testing.T) {
		uid := seedUser(t, "alert-precision@example.com")
		src := seedSource(t, "alert-precision", "al_db", "al_precision")

		// 16777217 = 2^24 + 1, the smallest positive integer that float32
		// cannot represent exactly (it rounds to 16777216). Regression
		// guard for the threshold_value DOUBLE PRECISION widening.
		const exact = 16777217.0
		alert := makeAlert(src.ID, "alert-precision", &uid)
		alert.ThresholdValue = exact
		if err := s.CreateAlert(ctx, alert); err != nil {
			t.Fatalf("CreateAlert: %v", err)
		}

		got, err := s.GetAlert(ctx, alert.ID)
		if err != nil {
			t.Fatalf("GetAlert(%d): %v", alert.ID, err)
		}
		if got.ThresholdValue != exact {
			t.Errorf("GetAlert ThresholdValue: got %v, want %v (lost precision via float32)",
				got.ThresholdValue, exact)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		uid := seedUser(t, "alert-delete@example.com")
		src := seedSource(t, "alert-delete", "al_db", "al_delete")
		alert := makeAlert(src.ID, "alert-delete", &uid)
		if err := s.CreateAlert(ctx, alert); err != nil {
			t.Fatalf("CreateAlert: %v", err)
		}
		if err := s.DeleteAlert(ctx, alert.ID); err != nil {
			t.Fatalf("DeleteAlert: %v", err)
		}
		_, err := s.GetAlert(ctx, alert.ID)
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetAlert after delete: want ErrNotFound, got %v", err)
		}
	})
}
