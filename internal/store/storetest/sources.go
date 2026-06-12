package storetest

import (
	"context"
	"errors"
	"testing"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/pkg/models"
)

// makeSource builds a minimal source with a unique database/table pair so
// each subtest's row stands apart under the (database, table_name)
// uniqueness constraint.
func makeSource(name, database, table string) *models.Source {
	return &models.Source{
		Name:        name,
		MetaTSField: "timestamp",
		Connection: models.ConnectionInfo{
			Host:      "clickhouse.example.com:9440",
			Username:  "logchef_reader",
			Password:  "secret",
			Database:  database,
			TableName: table,
			TLSEnable: true,
		},
		Description: "test source",
		TTLDays:     7,
	}
}

// testSources exercises CRUD on the sources table.
func testSources(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	t.Run("CreateGet", func(t *testing.T) {
		src := makeSource("primary", "default", "logs_one")
		if err := s.CreateSource(ctx, src); err != nil {
			t.Fatalf("CreateSource: %v", err)
		}
		if src.ID == 0 {
			t.Fatalf("CreateSource did not populate ID")
		}

		got, err := s.GetSource(ctx, src.ID)
		if err != nil {
			t.Fatalf("GetSource(%d): %v", src.ID, err)
		}
		if got.Name != src.Name ||
			got.Connection.Database != src.Connection.Database ||
			got.Connection.TableName != src.Connection.TableName {
			t.Errorf("GetSource mismatch: got %+v, want name=%s db=%s table=%s",
				got, src.Name, src.Connection.Database, src.Connection.TableName)
		}

		byName, err := s.GetSourceByName(ctx, src.Connection.Database, src.Connection.TableName)
		if err != nil {
			t.Fatalf("GetSourceByName: %v", err)
		}
		if byName.ID != src.ID {
			t.Errorf("GetSourceByName id mismatch: got %d, want %d", byName.ID, src.ID)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := s.GetSource(ctx, models.SourceID(999999))
		if err == nil {
			t.Fatalf("GetSource(bogus): expected error, got nil")
		}
		if !errors.Is(err, store.ErrSourceNotFound) {
			t.Errorf("GetSource(bogus): want ErrSourceNotFound, got %v", err)
		}
	})

	t.Run("UniqueDBTable", func(t *testing.T) {
		first := makeSource("dup-source-a", "dup_db", "dup_table")
		if err := s.CreateSource(ctx, first); err != nil {
			t.Fatalf("CreateSource(first): %v", err)
		}
		second := makeSource("dup-source-b", "dup_db", "dup_table")
		err := s.CreateSource(ctx, second)
		if err == nil {
			t.Fatalf("CreateSource(duplicate database/table): expected error, got nil")
		}
		if !store.IsUniqueConstraintError(err) {
			t.Errorf("CreateSource(duplicate database/table): want unique-constraint sentinel, got %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		before, err := s.ListSources(ctx)
		if err != nil {
			t.Fatalf("ListSources(before): %v", err)
		}
		seed := []*models.Source{
			makeSource("list-1", "ldb", "logs_a"),
			makeSource("list-2", "ldb", "logs_b"),
		}
		for _, src := range seed {
			if err := s.CreateSource(ctx, src); err != nil {
				t.Fatalf("CreateSource(%s): %v", src.Name, err)
			}
		}
		after, err := s.ListSources(ctx)
		if err != nil {
			t.Fatalf("ListSources(after): %v", err)
		}
		if len(after) != len(before)+len(seed) {
			t.Errorf("ListSources length: got %d, want %d", len(after), len(before)+len(seed))
		}
		names := make(map[string]bool, len(after))
		for _, src := range after {
			names[src.Name] = true
		}
		for _, src := range seed {
			if !names[src.Name] {
				t.Errorf("ListSources missing seeded name %s", src.Name)
			}
		}
	})

	t.Run("UpdateDelete", func(t *testing.T) {
		src := makeSource("mutate-source", "mdb", "logs_mut")
		if err := s.CreateSource(ctx, src); err != nil {
			t.Fatalf("CreateSource: %v", err)
		}

		src.Description = "updated description"
		src.TTLDays = 30
		if err := s.UpdateSource(ctx, src); err != nil {
			t.Fatalf("UpdateSource: %v", err)
		}
		got, err := s.GetSource(ctx, src.ID)
		if err != nil {
			t.Fatalf("GetSource after update: %v", err)
		}
		if got.Description != "updated description" || got.TTLDays != 30 {
			t.Errorf("UpdateSource not observed: got desc=%q ttl=%d", got.Description, got.TTLDays)
		}

		if err := s.DeleteSource(ctx, src.ID); err != nil {
			t.Fatalf("DeleteSource: %v", err)
		}
		_, err = s.GetSource(ctx, src.ID)
		if !errors.Is(err, store.ErrSourceNotFound) {
			t.Errorf("GetSource after delete: want ErrSourceNotFound, got %v", err)
		}
	})
}
