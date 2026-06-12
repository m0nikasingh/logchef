package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/mr-karan/logchef/internal/store/sqlite/sqlc"
	"github.com/mr-karan/logchef/pkg/models"
)

// Store is the backend-agnostic interface for logchef's operational
// metadata. Two implementations exist: internal/store/sqlite (default,
// single-binary) and internal/store/postgres (multi-replica). All
// methods accept context.Context and return either a typed model or one
// of the error sentinels defined in errors.go.
//
// The interface mirrors the public method set of *sqlite.DB verbatim.
// A handful of methods still reference sqlc-generated row/param types
// (sqlc.ApiToken, sqlc.CreateAPITokenParams, sqlc.DeleteAPITokenParams,
// sqlc.SystemSetting, sqlc.ListTeamsForUserRow). These leak the
// SQLite-internal sqlc package into the interface; future tasks may
// replace them with domain model types so the postgres backend does not
// have to import the SQLite-specific sqlc package. The *sqlc.Queries
// transaction handle (WriteQueriesWithTx) is intentionally NOT exposed:
// it remains a method on the concrete sqlite type only.
type Store interface {
	// Lifecycle.
	Close() error

	// Write transactions (used by provisioning + alert reconciliation).
	BeginWriteTx(ctx context.Context) (*sql.Tx, error)

	// --- Users ---
	CreateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, id models.UserID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	ListUsers(ctx context.Context) ([]*models.User, error)
	ListServiceAccounts(ctx context.Context) ([]*models.User, error)
	CountAdminUsers(ctx context.Context) (int, error)
	DeleteUser(ctx context.Context, id models.UserID) error

	// --- Teams ---
	CreateTeam(ctx context.Context, team *models.Team) error
	GetTeam(ctx context.Context, teamID models.TeamID) (*models.Team, error)
	UpdateTeam(ctx context.Context, team *models.Team) error
	DeleteTeam(ctx context.Context, teamID models.TeamID) error
	ListTeams(ctx context.Context) ([]*models.Team, error)
	AddTeamMember(ctx context.Context, teamID models.TeamID, userID models.UserID, role models.TeamRole) error
	GetTeamMember(ctx context.Context, teamID models.TeamID, userID models.UserID) (*models.TeamMember, error)
	UpdateTeamMemberRole(ctx context.Context, teamID models.TeamID, userID models.UserID, role models.TeamRole) error
	RemoveTeamMember(ctx context.Context, teamID models.TeamID, userID models.UserID) error
	ListTeamMembers(ctx context.Context, teamID models.TeamID) ([]*models.TeamMember, error)
	ListTeamMembersWithDetails(ctx context.Context, teamID models.TeamID) ([]*models.TeamMember, error)
	ListUserTeams(ctx context.Context, userID models.UserID) ([]*models.Team, error)
	AddTeamSource(ctx context.Context, teamID models.TeamID, sourceID models.SourceID) error
	RemoveTeamSource(ctx context.Context, teamID models.TeamID, sourceID models.SourceID) error
	ListTeamSources(ctx context.Context, teamID models.TeamID) ([]*models.Source, error)
	ListSourceTeams(ctx context.Context, sourceID models.SourceID) ([]*models.Team, error)
	ListSourcesForUser(ctx context.Context, userID models.UserID) ([]*models.Source, error)
	GetTeamByName(ctx context.Context, name string) (*models.Team, error)
	TeamHasSource(ctx context.Context, teamID models.TeamID, sourceID models.SourceID) (bool, error)
	UserHasSourceAccess(ctx context.Context, userID models.UserID, sourceID models.SourceID) (bool, error)
	ListTeamsForUser(ctx context.Context, userID models.UserID) ([]sqlc.ListTeamsForUserRow, error)

	// --- Sources ---
	CreateSource(ctx context.Context, source *models.Source) error
	GetSource(ctx context.Context, id models.SourceID) (*models.Source, error)
	GetSourceByName(ctx context.Context, database, tableName string) (*models.Source, error)
	ListSources(ctx context.Context) ([]*models.Source, error)
	UpdateSource(ctx context.Context, source *models.Source) error
	DeleteSource(ctx context.Context, id models.SourceID) error

	// --- Sessions ---
	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, sessionID models.SessionID) (*models.Session, error)
	DeleteSession(ctx context.Context, sessionID models.SessionID) error
	DeleteUserSessions(ctx context.Context, userID models.UserID) error
	CountUserSessions(ctx context.Context, userID models.UserID) (int, error)

	// --- Saved queries ---
	CreateSavedQuery(ctx context.Context, sourceID models.SourceID, createdFromTeamID *models.TeamID, name, description, queryType, queryContent string, createdBy *models.UserID) (*models.SavedQuery, error)
	GetSavedQuery(ctx context.Context, queryID int) (*models.SavedQuery, error)
	UpdateSavedQuery(ctx context.Context, queryID int, name, description, queryType, queryContent string) error
	DeleteSavedQuery(ctx context.Context, queryID int) error
	ListSavedQueriesForUser(ctx context.Context, userID models.UserID) ([]*models.SavedQuery, error)
	ListSavedQueriesForUserBySource(ctx context.Context, userID models.UserID, sourceID models.SourceID) ([]*models.SavedQuery, error)

	// --- Collections ---
	CreateCollection(ctx context.Context, name, description string, isPersonal bool, createdBy models.UserID) (*models.Collection, error)
	GetCollection(ctx context.Context, collectionID int) (*models.Collection, error)
	GetPersonalCollection(ctx context.Context, userID models.UserID) (*models.Collection, error)
	UpdateCollection(ctx context.Context, collectionID int, name, description string) error
	DeleteCollection(ctx context.Context, collectionID int) error
	ListCollectionsForUser(ctx context.Context, userID models.UserID) ([]*models.Collection, error)
	AddCollectionMember(ctx context.Context, collectionID int, userID models.UserID, role models.CollectionRole, addedBy *models.UserID) error
	GetCollectionMember(ctx context.Context, collectionID int, userID models.UserID) (*models.CollectionMember, error)
	ListCollectionMembers(ctx context.Context, collectionID int) ([]*models.CollectionMember, error)
	RemoveCollectionMember(ctx context.Context, collectionID int, userID models.UserID) error
	AddCollectionItem(ctx context.Context, collectionID, savedQueryID, sortOrder int, addedBy *models.UserID) error
	RemoveCollectionItem(ctx context.Context, collectionID, savedQueryID int) error
	ListCollectionItems(ctx context.Context, collectionID int) ([]*models.CollectionItem, error)

	// --- Alerts ---
	CreateAlert(ctx context.Context, alert *models.Alert) error
	UpdateAlert(ctx context.Context, alert *models.Alert) error
	DeleteAlert(ctx context.Context, alertID models.AlertID) error
	GetAlert(ctx context.Context, alertID models.AlertID) (*models.Alert, error)
	ListAlertsBySource(ctx context.Context, sourceID models.SourceID) ([]*models.Alert, error)
	ListAlertsForUser(ctx context.Context, userID models.UserID) ([]*models.Alert, error)
	ListActiveAlertsDue(ctx context.Context) ([]*models.Alert, error)
	MarkAlertEvaluated(ctx context.Context, alertID models.AlertID) error
	MarkAlertTriggered(ctx context.Context, alertID models.AlertID) error
	InsertAlertHistory(ctx context.Context, alertID models.AlertID, status models.AlertStatus, value *float64, message string, payload map[string]any) (*models.AlertHistoryEntry, error)
	GetLatestUnresolvedAlertHistory(ctx context.Context, alertID models.AlertID) (*models.AlertHistoryEntry, error)
	ResolveAlertHistory(ctx context.Context, historyID int64, message string) error
	UpdateAlertHistoryPayload(ctx context.Context, historyID int64, payload map[string]any) error
	ListAlertHistory(ctx context.Context, alertID models.AlertID, limit int) ([]*models.AlertHistoryEntry, error)
	PruneAlertHistory(ctx context.Context, alertID models.AlertID, keep int) error

	// --- API tokens ---
	CreateAPIToken(ctx context.Context, params sqlc.CreateAPITokenParams) (int64, error)
	GetAPIToken(ctx context.Context, id int64) (sqlc.ApiToken, error)
	GetAPITokenByHash(ctx context.Context, tokenHash string) (sqlc.ApiToken, error)
	ListAPITokensForUser(ctx context.Context, userID int64) ([]sqlc.ApiToken, error)
	UpdateAPITokenLastUsed(ctx context.Context, id int64) error
	DeleteAPIToken(ctx context.Context, params sqlc.DeleteAPITokenParams) error
	DeleteExpiredAPITokens(ctx context.Context) error

	// --- Settings ---
	GetSetting(ctx context.Context, key string) (string, error)
	GetSettingWithDefault(ctx context.Context, key, defaultValue string) string
	GetBoolSetting(ctx context.Context, key string, defaultValue bool) bool
	GetIntSetting(ctx context.Context, key string, defaultValue int) int
	GetFloat64Setting(ctx context.Context, key string, defaultValue float64) float64
	GetDurationSetting(ctx context.Context, key string, defaultValue time.Duration) time.Duration
	ListSettings(ctx context.Context) ([]sqlc.SystemSetting, error)
	ListSettingsByCategory(ctx context.Context, category string) ([]sqlc.SystemSetting, error)
	UpsertSetting(ctx context.Context, key, value, valueType, category, description string, isSensitive bool) error
	DeleteSetting(ctx context.Context, key string) error

	// --- User preferences ---
	GetUserPreferencesJSON(ctx context.Context, userID models.UserID) (string, error)
	UpsertUserPreferencesJSON(ctx context.Context, userID models.UserID, preferencesJSON string) error

	// --- Query shares ---
	CreateQueryShare(ctx context.Context, share *models.QueryShare) error
	GetQueryShare(ctx context.Context, token string) (*models.QueryShare, error)
	TouchQueryShare(ctx context.Context, token string, accessedAt time.Time) error
	DeleteQueryShare(ctx context.Context, token string) error
	GetUserTeamForSource(ctx context.Context, userID models.UserID, sourceID models.SourceID) (models.TeamID, error)
	PruneExpiredQueryShares(ctx context.Context, before time.Time) error

	// --- Export jobs ---
	CreateExportJob(ctx context.Context, job *models.ExportJob) error
	GetExportJob(ctx context.Context, id string) (*models.ExportJob, error)
	UpdateExportJobRunning(ctx context.Context, id string, updatedAt time.Time) error
	CompleteExportJob(ctx context.Context, id, fileName, filePath string, rowsExported int, bytesWritten int64, completedAt time.Time) error
	FailExportJob(ctx context.Context, id, errorMessage string, updatedAt time.Time) error
	ListExpiredExportJobPaths(ctx context.Context, before time.Time) ([]string, error)
	DeleteExpiredExportJobs(ctx context.Context, before time.Time) error

	// --- Provisioning helpers ---
	IsSourceManaged(ctx context.Context, id models.SourceID) (bool, error)
	IsTeamManaged(ctx context.Context, id models.TeamID) (bool, error)
	IsUserManaged(ctx context.Context, id models.UserID) (bool, error)
}
