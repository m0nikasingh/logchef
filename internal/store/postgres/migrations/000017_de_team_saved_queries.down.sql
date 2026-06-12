-- Best-effort revert: restore team_queries shape with team_id derived from
-- team_sources. created_by is dropped -- v1 schema does not carry it.

DROP INDEX IF EXISTS idx_saved_queries_created_by;
DROP INDEX IF EXISTS idx_saved_queries_source_bookmark;

ALTER TABLE saved_queries DROP COLUMN created_by;

-- Add team_id nullable first, backfill from team_sources, then enforce NOT NULL
-- and the FK constraint.
ALTER TABLE saved_queries ADD COLUMN team_id BIGINT;

UPDATE saved_queries sq
SET team_id = COALESCE(
    (SELECT MIN(ts.team_id) FROM team_sources ts WHERE ts.source_id = sq.source_id),
    (SELECT MIN(id) FROM teams)
);

ALTER TABLE saved_queries ALTER COLUMN team_id SET NOT NULL;
ALTER TABLE saved_queries
    ADD CONSTRAINT saved_queries_team_id_fkey
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

ALTER TABLE saved_queries RENAME TO team_queries;

CREATE INDEX IF NOT EXISTS idx_team_queries_bookmarked
    ON team_queries(team_id, source_id, is_bookmarked, updated_at);
