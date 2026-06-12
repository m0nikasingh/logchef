-- Restore is_bookmarked on saved_queries. Best-effort backfill: a query is
-- marked bookmarked if it exists in its creator's personal collection. This
-- is approximate (the original pre-1.6 signal was lost) but matches the
-- intent of the up migration.

DROP INDEX IF EXISTS idx_saved_queries_created_by;
DROP INDEX IF EXISTS idx_saved_queries_source;

ALTER TABLE saved_queries ADD COLUMN is_bookmarked BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE saved_queries sq
SET is_bookmarked = TRUE
WHERE EXISTS (
    SELECT 1
    FROM collection_items ci
    JOIN collections pc ON pc.id = ci.collection_id
    WHERE ci.saved_query_id = sq.id
      AND pc.is_personal = TRUE
      AND pc.created_by = sq.created_by
);

CREATE INDEX IF NOT EXISTS idx_saved_queries_source_bookmark
    ON saved_queries(source_id, is_bookmarked, updated_at);
CREATE INDEX IF NOT EXISTS idx_saved_queries_created_by ON saved_queries(created_by);

-- Personal collections created by 000020.up are intentionally not removed --
-- collections are user-visible state, and dropping them on revert would lose
-- user-curated items that survived the bookmark migration.
