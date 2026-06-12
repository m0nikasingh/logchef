-- Decouple saved queries from teams: rename team_queries to saved_queries,
-- drop team_id, and add a nullable created_by (FK to users). Visibility and
-- edit access are now driven by source membership and creator/admin checks
-- at the application layer.

-- Drop the old team-scoped index before reshaping the table.
DROP INDEX IF EXISTS idx_team_queries_bookmarked;

ALTER TABLE team_queries RENAME TO saved_queries;
ALTER TABLE saved_queries DROP COLUMN team_id;
ALTER TABLE saved_queries
    ADD COLUMN created_by BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_saved_queries_source_bookmark
    ON saved_queries(source_id, is_bookmarked, updated_at);
CREATE INDEX IF NOT EXISTS idx_saved_queries_created_by
    ON saved_queries(created_by);
