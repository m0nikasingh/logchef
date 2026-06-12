-- Best-effort revert: restore team_id NOT NULL on alerts, query_shares, and
-- export_jobs by picking the smallest team_id linked to the row's source.
-- alerts.created_by is dropped (v1 schema doesn't carry it).

-- ---------------- alerts ----------------

DROP INDEX IF EXISTS idx_alerts_created_by;
DROP INDEX IF EXISTS idx_alerts_source;

ALTER TABLE alerts DROP COLUMN created_by;

ALTER TABLE alerts ADD COLUMN team_id BIGINT;
UPDATE alerts a
SET team_id = COALESCE(
    (SELECT MIN(ts.team_id) FROM team_sources ts WHERE ts.source_id = a.source_id),
    (SELECT MIN(id) FROM teams)
);
ALTER TABLE alerts ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE alerts
    ADD CONSTRAINT alerts_team_id_fkey
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

CREATE INDEX idx_alerts_team_source ON alerts(team_id, source_id);

-- ---------------- query_shares ----------------

DROP INDEX IF EXISTS idx_query_shares_expires_at;
DROP INDEX IF EXISTS idx_query_shares_created_by;
DROP INDEX IF EXISTS idx_query_shares_source;

ALTER TABLE query_shares ADD COLUMN team_id BIGINT;
UPDATE query_shares qs
SET team_id = COALESCE(
    (SELECT MIN(ts.team_id) FROM team_sources ts WHERE ts.source_id = qs.source_id),
    (SELECT MIN(id) FROM teams)
);
ALTER TABLE query_shares ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE query_shares
    ADD CONSTRAINT query_shares_team_id_fkey
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_query_shares_team_source ON query_shares(team_id, source_id);
CREATE INDEX IF NOT EXISTS idx_query_shares_created_by ON query_shares(created_by);
CREATE INDEX IF NOT EXISTS idx_query_shares_expires_at ON query_shares(expires_at);

-- ---------------- export_jobs ----------------

DROP INDEX IF EXISTS idx_export_jobs_expires_at;
DROP INDEX IF EXISTS idx_export_jobs_created_by;
DROP INDEX IF EXISTS idx_export_jobs_source;

ALTER TABLE export_jobs ADD COLUMN team_id BIGINT;
UPDATE export_jobs ej
SET team_id = COALESCE(
    (SELECT MIN(ts.team_id) FROM team_sources ts WHERE ts.source_id = ej.source_id),
    (SELECT MIN(id) FROM teams)
);
ALTER TABLE export_jobs ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE export_jobs
    ADD CONSTRAINT export_jobs_team_id_fkey
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_export_jobs_team_source ON export_jobs(team_id, source_id);
CREATE INDEX IF NOT EXISTS idx_export_jobs_created_by ON export_jobs(created_by);
CREATE INDEX IF NOT EXISTS idx_export_jobs_expires_at ON export_jobs(expires_at);
