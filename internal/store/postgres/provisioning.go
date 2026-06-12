package postgres

import (
	"context"

	"github.com/mr-karan/logchef/pkg/models"
)

// IsSourceManaged returns true if the source is managed by provisioning config.
func (db *DB) IsSourceManaged(ctx context.Context, id models.SourceID) (bool, error) {
	return db.queries.IsSourceManaged(ctx, int64(id))
}

// IsTeamManaged returns true if the team is managed by provisioning config.
func (db *DB) IsTeamManaged(ctx context.Context, id models.TeamID) (bool, error) {
	return db.queries.IsTeamManaged(ctx, int64(id))
}

// IsUserManaged returns true if the user is managed by provisioning config.
func (db *DB) IsUserManaged(ctx context.Context, id models.UserID) (bool, error) {
	return db.queries.IsUserManaged(ctx, int64(id))
}
