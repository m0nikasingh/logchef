package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
)

// testSettings exercises the system_settings methods. There is no
// separate Create; values are written via UpsertSetting and reads use
// the typed Get helpers. List/ListByCategory still leak sqlc-generated
// types (sqlc.SystemSetting), a known concern tracked for Phase 3.
func testSettings(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	t.Run("UpsertGet", func(t *testing.T) {
		if err := s.UpsertSetting(ctx,
			"settings.upsert.string", "hello", "string", "server", "first write", false,
		); err != nil {
			t.Fatalf("UpsertSetting(insert): %v", err)
		}
		got, err := s.GetSetting(ctx, "settings.upsert.string")
		if err != nil {
			t.Fatalf("GetSetting: %v", err)
		}
		if got != "hello" {
			t.Errorf("GetSetting: got %q, want hello", got)
		}

		// Upsert again to verify update path.
		if err := s.UpsertSetting(ctx,
			"settings.upsert.string", "world", "string", "server", "second write", false,
		); err != nil {
			t.Fatalf("UpsertSetting(update): %v", err)
		}
		got, err = s.GetSetting(ctx, "settings.upsert.string")
		if err != nil {
			t.Fatalf("GetSetting after update: %v", err)
		}
		if got != "world" {
			t.Errorf("GetSetting after update: got %q, want world", got)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetSetting(ctx, "settings.does.not.exist")
		if err == nil {
			t.Fatalf("GetSetting(bogus): expected error, got nil")
		}
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetSetting(bogus): want ErrNotFound, got %v", err)
		}
	})

	t.Run("TypedHelpersWithDefault", func(t *testing.T) {
		if err := s.UpsertSetting(ctx,
			"settings.typed.bool", "true", "boolean", "server", "", false,
		); err != nil {
			t.Fatalf("UpsertSetting(bool): %v", err)
		}
		if got := s.GetBoolSetting(ctx, "settings.typed.bool", false); !got {
			t.Errorf("GetBoolSetting: got false, want true")
		}
		// Missing key falls back to default.
		if got := s.GetBoolSetting(ctx, "settings.typed.missing", true); !got {
			t.Errorf("GetBoolSetting(missing): got false, want default true")
		}
		if got := s.GetSettingWithDefault(ctx, "settings.typed.missing", "fallback"); got != "fallback" {
			t.Errorf("GetSettingWithDefault: got %q, want fallback", got)
		}
	})

	t.Run("ListAndListByCategory", func(t *testing.T) {
		seed := []struct {
			key, value, category string
		}{
			{"settings.list.alerts.a", "1", "alerts"},
			{"settings.list.alerts.b", "2", "alerts"},
			{"settings.list.ai.a", "3", "ai"},
		}
		for _, e := range seed {
			if err := s.UpsertSetting(ctx, e.key, e.value, "string", e.category, "", false); err != nil {
				t.Fatalf("UpsertSetting(%s): %v", e.key, err)
			}
		}

		all, err := s.ListSettings(ctx)
		if err != nil {
			t.Fatalf("ListSettings: %v", err)
		}
		keys := make(map[string]bool, len(all))
		for _, row := range all {
			keys[row.Key] = true
		}
		for _, e := range seed {
			if !keys[e.key] {
				t.Errorf("ListSettings missing seeded key %s", e.key)
			}
		}

		alerts, err := s.ListSettingsByCategory(ctx, "alerts")
		if err != nil {
			t.Fatalf("ListSettingsByCategory: %v", err)
		}
		seen := 0
		for _, row := range alerts {
			if row.Category != "alerts" {
				t.Errorf("ListSettingsByCategory returned wrong category: %q", row.Category)
			}
			if row.Key == "settings.list.alerts.a" || row.Key == "settings.list.alerts.b" {
				seen++
			}
		}
		if seen != 2 {
			t.Errorf("ListSettingsByCategory(alerts): saw %d seeded rows, want 2", seen)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "settings.delete.target"
		if err := s.UpsertSetting(ctx, key, "doomed", "string", "server", "", false); err != nil {
			t.Fatalf("UpsertSetting: %v", err)
		}
		if err := s.DeleteSetting(ctx, key); err != nil {
			t.Fatalf("DeleteSetting: %v", err)
		}
		_, err := s.GetSetting(ctx, key)
		if !errors.Is(err, store.ErrNotFound) {
			t.Errorf("GetSetting after delete: want ErrNotFound, got %v", err)
		}
	})
}
