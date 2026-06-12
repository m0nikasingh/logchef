// Package store defines the backend-agnostic interface for logchef's
// operational metadata (sessions, users, teams, sources, saved queries,
// alerts, API tokens, system settings). Implementations live in
// subpackages (sqlite, postgres).
package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/mr-karan/logchef/pkg/models"
)

// Common error sentinels returned by Store implementations. Callers use
// errors.Is to branch.
var (
	ErrNotFound         = errors.New("not found")
	ErrUserNotFound     = errors.New("user not found")
	ErrTeamNotFound     = errors.New("team not found")
	ErrSourceNotFound   = errors.New("source not found")
	ErrSessionNotFound  = errors.New("session not found")
	ErrQueryNotFound    = errors.New("query not found")
	ErrUniqueConstraint = errors.New("unique constraint violation")
	ErrUserExists       = errors.New("user already exists")
	ErrTeamExists       = errors.New("team already exists")
	ErrSourceExists     = errors.New("source already exists")
)

// IsNotFoundError reports whether err (or any wrapped error) is a
// not-found sentinel from any Store implementation, the models package,
// or the standard database/sql layer (sql.ErrNoRows).
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrTeamNotFound) ||
		errors.Is(err, ErrSourceNotFound) ||
		errors.Is(err, ErrSessionNotFound) ||
		errors.Is(err, ErrQueryNotFound) ||
		errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, models.ErrNotFound) ||
		errors.Is(err, models.ErrUserNotFound) ||
		errors.Is(err, models.ErrTeamNotFound)
}

func IsUserNotFoundError(err error) bool {
	return errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, models.ErrUserNotFound) ||
		(errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "user")) ||
		(errors.Is(err, models.ErrNotFound) && strings.Contains(err.Error(), "user"))
}

func IsTeamNotFoundError(err error) bool {
	return errors.Is(err, ErrTeamNotFound) ||
		errors.Is(err, models.ErrTeamNotFound) ||
		(errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "team")) ||
		(errors.Is(err, models.ErrNotFound) && strings.Contains(err.Error(), "team"))
}

func IsSourceNotFoundError(err error) bool {
	return errors.Is(err, ErrSourceNotFound) ||
		(errors.Is(err, ErrNotFound) && strings.Contains(err.Error(), "source")) ||
		(errors.Is(err, models.ErrNotFound) && strings.Contains(err.Error(), "source"))
}

func IsUniqueConstraintError(err error) bool {
	return errors.Is(err, ErrUniqueConstraint) ||
		errors.Is(err, ErrUserExists) ||
		errors.Is(err, ErrTeamExists) ||
		errors.Is(err, ErrSourceExists)
}
