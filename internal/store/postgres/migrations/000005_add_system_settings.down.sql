-- Drop system_settings table
-- NOTE: SQLite original had "DROP TABLE IF NOT EXISTS" (typo, invalid SQL on both engines).
-- Corrected here to "IF EXISTS" so the down migration is runnable on Postgres.
DROP TABLE IF EXISTS system_settings;
