-- Drop team_id from alerts, query_shares, and export_jobs. Add nullable
-- created_by to alerts (the other two already have it). Visibility and edit
-- access are now driven by source membership and creator/admin checks at the
-- application layer, mirroring the saved-queries change in 000017.
--
-- In Postgres we can DROP/ADD COLUMN in place; the SQLite version had to
-- rebuild each table and toggle foreign_keys for alert_history's FK to
-- alerts. Postgres preserves alert_history rows transparently here.

-- ---------------- alerts ----------------

DROP INDEX IF EXISTS idx_alerts_team_source;

ALTER TABLE alerts DROP COLUMN team_id;
ALTER TABLE alerts
    ADD COLUMN created_by BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_alerts_source ON alerts(source_id);
CREATE INDEX idx_alerts_created_by ON alerts(created_by);

-- ---------------- query_shares ----------------

DROP INDEX IF EXISTS idx_query_shares_team_source;

ALTER TABLE query_shares DROP COLUMN team_id;

CREATE INDEX IF NOT EXISTS idx_query_shares_source ON query_shares(source_id);
CREATE INDEX IF NOT EXISTS idx_query_shares_created_by ON query_shares(created_by);
CREATE INDEX IF NOT EXISTS idx_query_shares_expires_at ON query_shares(expires_at);

-- ---------------- export_jobs ----------------

DROP INDEX IF EXISTS idx_export_jobs_team_source;

ALTER TABLE export_jobs DROP COLUMN team_id;

CREATE INDEX IF NOT EXISTS idx_export_jobs_source ON export_jobs(source_id);
CREATE INDEX IF NOT EXISTS idx_export_jobs_created_by ON export_jobs(created_by);
CREATE INDEX IF NOT EXISTS idx_export_jobs_expires_at ON export_jobs(expires_at);
