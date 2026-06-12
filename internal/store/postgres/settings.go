package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	pgsqlc "github.com/mr-karan/logchef/internal/store/postgres/sqlc"
	sqlc "github.com/mr-karan/logchef/internal/store/sqlite/sqlc"
)

// The Store interface still references sqlite-internal sqlc.SystemSetting in
// ListSettings/ListSettingsByCategory return types. Until that interface leak
// is resolved (see T22 concern notes), this file translates field-by-field at
// the boundary between postgres/sqlc (native bool) and sqlite/sqlc (int64).

// GetSetting retrieves a setting value from the database.
func (db *DB) GetSetting(ctx context.Context, key string) (string, error) {
	setting, err := db.queries.GetSystemSetting(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get setting %s: %w", key, err)
	}
	return setting.Value, nil
}

// GetSettingWithDefault retrieves a setting value or returns the default if not found.
func (db *DB) GetSettingWithDefault(ctx context.Context, key, defaultValue string) string {
	value, err := db.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetBoolSetting retrieves a boolean setting value.
func (db *DB) GetBoolSetting(ctx context.Context, key string, defaultValue bool) bool {
	value, err := db.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolVal
}

// GetIntSetting retrieves an integer setting value.
func (db *DB) GetIntSetting(ctx context.Context, key string, defaultValue int) int {
	value, err := db.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// GetFloat64Setting retrieves a float64 setting value.
func (db *DB) GetFloat64Setting(ctx context.Context, key string, defaultValue float64) float64 {
	value, err := db.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}
	floatVal, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return floatVal
}

// GetDurationSetting retrieves a duration setting value.
func (db *DB) GetDurationSetting(ctx context.Context, key string, defaultValue time.Duration) time.Duration {
	value, err := db.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}
	durationVal, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return durationVal
}

// ListSettings retrieves all settings.
func (db *DB) ListSettings(ctx context.Context) ([]sqlc.SystemSetting, error) {
	settings, err := db.queries.ListSystemSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	out := make([]sqlc.SystemSetting, 0, len(settings))
	for _, s := range settings {
		out = append(out, systemSettingFromPG(s))
	}
	return out, nil
}

// ListSettingsByCategory retrieves settings for a specific category.
func (db *DB) ListSettingsByCategory(ctx context.Context, category string) ([]sqlc.SystemSetting, error) {
	settings, err := db.queries.ListSystemSettingsByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings for category %s: %w", category, err)
	}
	out := make([]sqlc.SystemSetting, 0, len(settings))
	for _, s := range settings {
		out = append(out, systemSettingFromPG(s))
	}
	return out, nil
}

// UpsertSetting inserts or updates a setting.
func (db *DB) UpsertSetting(ctx context.Context, key, value, valueType, category, description string, isSensitive bool) error {
	err := db.queries.UpsertSystemSetting(ctx, pgsqlc.UpsertSystemSettingParams{
		Key:         key,
		Value:       value,
		ValueType:   valueType,
		Category:    category,
		Description: sql.NullString{String: description, Valid: description != ""},
		IsSensitive: isSensitive,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert setting %s: %w", key, err)
	}
	return nil
}

// systemSettingFromPG converts a postgres-sqlc SystemSetting into the
// sqlite-sqlc shape required by the Store interface. The only field that
// differs across backends is IsSensitive (bool in postgres, int64 in sqlite).
func systemSettingFromPG(s pgsqlc.SystemSetting) sqlc.SystemSetting {
	var isSensitive int64
	if s.IsSensitive {
		isSensitive = 1
	}
	return sqlc.SystemSetting{
		Key:         s.Key,
		Value:       s.Value,
		ValueType:   s.ValueType,
		Category:    s.Category,
		Description: s.Description,
		IsSensitive: isSensitive,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// DeleteSetting deletes a setting.
func (db *DB) DeleteSetting(ctx context.Context, key string) error {
	err := db.queries.DeleteSystemSetting(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting %s: %w", key, err)
	}
	return nil
}
