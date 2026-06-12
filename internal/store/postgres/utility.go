package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mr-karan/logchef/internal/store"
	"github.com/mr-karan/logchef/internal/store/postgres/sqlc"
	"github.com/mr-karan/logchef/pkg/models"
)

// Sentinels are re-exported from internal/store so callers using errors.Is
// see the same sentinels regardless of backend.
var (
	ErrNotFound         = store.ErrNotFound
	ErrUserNotFound     = store.ErrUserNotFound
	ErrTeamNotFound     = store.ErrTeamNotFound
	ErrSourceNotFound   = store.ErrSourceNotFound
	ErrSessionNotFound  = store.ErrSessionNotFound
	ErrQueryNotFound    = store.ErrQueryNotFound
	ErrUniqueConstraint = store.ErrUniqueConstraint
	ErrUserExists       = store.ErrUserExists
	ErrTeamExists       = store.ErrTeamExists
	ErrSourceExists     = store.ErrSourceExists
)

// IsNotFoundError reports whether err is any flavor of not-found.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, models.ErrNotFound) ||
		errors.Is(err, models.ErrUserNotFound) ||
		errors.Is(err, models.ErrTeamNotFound)
}

// IsUserNotFoundError reports whether err is specifically a user-not-found.
func IsUserNotFoundError(err error) bool {
	return errors.Is(err, ErrUserNotFound) ||
		(errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "user"))
}

// IsTeamNotFoundError reports whether err is specifically a team-not-found.
func IsTeamNotFoundError(err error) bool {
	return errors.Is(err, ErrTeamNotFound) ||
		(errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "team"))
}

// IsSourceNotFoundError reports whether err is specifically a source-not-found.
func IsSourceNotFoundError(err error) bool {
	return errors.Is(err, ErrSourceNotFound) ||
		(errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "source"))
}

// IsUniqueConstraintError reports whether err is a unique-constraint violation.
func IsUniqueConstraintError(err error) bool {
	return errors.Is(err, ErrUniqueConstraint) || isUniqueConstraintPostgresError(err)
}

// nullString wraps a string into sql.NullString, treating empty as NULL.
func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

// mapSourceRowToModel maps a postgres sqlc.Source to a models.Source.
// Unlike the sqlite mapper, postgres returns native bool/time so no int-to-bool
// conversion is needed.
func mapSourceRowToModel(row *sqlc.Source) *models.Source {
	if row == nil {
		return nil
	}
	return &models.Source{
		ID:                models.SourceID(row.ID),
		Name:              row.Name,
		MetaIsAutoCreated: row.MetaIsAutoCreated,
		MetaTSField:       row.MetaTsField,
		MetaSeverityField: row.MetaSeverityField.String,
		Description:       row.Description.String,
		TTLDays:           int(row.TtlDays),
		Connection: models.ConnectionInfo{
			Host:      row.Host,
			Username:  row.Username,
			Password:  row.Password,
			Database:  row.Database,
			TableName: row.TableName,
			TLSEnable: row.TlsEnable,
		},
		Timestamps: models.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
		Managed:   row.Managed,
		SecretRef: row.SecretRef.String,
	}
}

// isUniqueConstraintPostgresError reports whether err is a Postgres 23505 violation.
func isUniqueConstraintPostgresError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UniqueViolation
	}
	return false
}

// wrapError wraps an error with additional context (matches sqlite signature).
func wrapError(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// handleNotFoundError maps sql.ErrNoRows into the appropriate domain sentinel
// based on a free-form prefix. Mirrors the sqlite helper exactly so domain
// files port without touching call sites.
func handleNotFoundError(err error, prefix string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		// Order matters: more specific patterns first ("query" before "team"
		// so "team query" matches query, not team).
		if strings.Contains(prefix, "user") {
			if strings.Contains(prefix, "email") {
				return wrapError(ErrUserNotFound, "getting user email %s", strings.TrimPrefix(prefix, "getting user email "))
			}
			return wrapError(ErrUserNotFound, prefix)
		}
		if strings.Contains(prefix, "query") {
			return wrapError(ErrQueryNotFound, prefix)
		}
		if strings.Contains(prefix, "team") {
			return wrapError(ErrTeamNotFound, prefix)
		}
		if strings.Contains(prefix, "source") {
			return wrapError(ErrSourceNotFound, prefix)
		}
		if strings.Contains(prefix, "session") {
			return wrapError(ErrSessionNotFound, prefix)
		}
		return wrapError(ErrNotFound, prefix)
	}

	return wrapError(err, prefix)
}

// handleUniqueConstraintError maps a Postgres unique-violation onto a per-domain
// sentinel. Mirrors the sqlite signature so domain files port unchanged.
func handleUniqueConstraintError(err error, table, column, value string) error {
	if err == nil {
		return nil
	}

	if isUniqueConstraintPostgresError(err) {
		switch {
		case table == "users" && column == "email":
			return wrapError(ErrUserExists, "email %s", value)
		case table == "teams" && column == "name":
			return wrapError(ErrTeamExists, "name %s", value)
		case table == "sources" && (column == "name" || column == "database_table"):
			return wrapError(ErrSourceExists, value)
		default:
			return wrapError(ErrUniqueConstraint, "%s.%s: %s", table, column, value)
		}
	}

	return err
}
