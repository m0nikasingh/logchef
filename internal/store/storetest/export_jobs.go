package storetest

import (
	"context"
	"testing"
	"time"

	"github.com/mr-karan/logchef/pkg/models"
)

// testExportJobs covers the export-job happy path. The "Roundtrip5GiB"
// subtest is the regression for the BIGINT widening of bytes_written:
// 5 GiB exceeds the INT32 range, so a narrow column would silently
// truncate it to a negative number.
func testExportJobs(t *testing.T, newStore Factory) {
	s := newStore(t)
	ctx := context.Background()

	// seedUser inserts a user with a unique email and returns its ID.
	seedUser := func(t *testing.T, email string) models.UserID {
		t.Helper()
		u := &models.User{
			Email:    email,
			FullName: "Export User",
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

	t.Run("Roundtrip5GiB", func(t *testing.T) {
		uid := seedUser(t, "export-bytes@example.com")
		src := seedSource(t, "export-bytes", "ex_db", "ex_bytes")

		now := time.Now().UTC().Truncate(time.Second)
		job := &models.ExportJob{
			ID:             "export-bytes-1",
			SourceID:       src.ID,
			CreatedBy:      uid,
			Status:         models.ExportJobStatusPending,
			Format:         "csv",
			RequestPayload: []byte(`{"raw_sql":"SELECT 1","format":"csv"}`),
			ExpiresAt:      now.Add(24 * time.Hour),
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.CreateExportJob(ctx, job); err != nil {
			t.Fatalf("CreateExportJob: %v", err)
		}

		// 5 GiB - exceeds INT32 (max 2 GiB - 1). Regression guard for
		// the bytes_written BIGINT widening.
		const fiveGiB = int64(5) * 1024 * 1024 * 1024
		completedAt := now.Add(1 * time.Minute)
		if err := s.CompleteExportJob(ctx, job.ID, "export.csv", "/tmp/export.csv",
			1234, fiveGiB, completedAt); err != nil {
			t.Fatalf("CompleteExportJob: %v", err)
		}

		got, err := s.GetExportJob(ctx, job.ID)
		if err != nil {
			t.Fatalf("GetExportJob(%s): %v", job.ID, err)
		}
		if got.BytesWritten != fiveGiB {
			t.Errorf("GetExportJob BytesWritten: got %d, want %d (5 GiB)",
				got.BytesWritten, fiveGiB)
		}
		if got.RowsExported != 1234 {
			t.Errorf("GetExportJob RowsExported: got %d, want 1234", got.RowsExported)
		}
		if got.Status != models.ExportJobStatusComplete {
			t.Errorf("GetExportJob Status: got %q, want %q", got.Status, models.ExportJobStatusComplete)
		}
		if got.FileName != "export.csv" || got.FilePath != "/tmp/export.csv" {
			t.Errorf("GetExportJob file fields: got name=%q path=%q",
				got.FileName, got.FilePath)
		}
	})
}
