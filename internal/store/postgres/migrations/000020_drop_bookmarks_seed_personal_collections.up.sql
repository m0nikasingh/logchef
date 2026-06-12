-- Retire the saved-query bookmark flag. Personal collections take its place:
-- every existing user gets a personal collection back-filled with the queries
-- they had bookmarked (best-effort, only for queries with a known creator).

-- 1. Create a personal collection for every user that does not have one yet.
--    The default name is "My Collection" -- personal collections are never
--    shared, so the label is purely owner-facing. Users can rename via the UI.
--    Existing personal collections are left untouched.
INSERT INTO collections (name, description, is_personal, created_by, created_at, updated_at)
SELECT 'My Collection', '', TRUE, u.id, NOW(), NOW()
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM collections c WHERE c.created_by = u.id AND c.is_personal = TRUE
);

-- 2. Ensure the owner-membership row exists for every personal collection
--    (safe to re-run; fixes any half-broken state from the v1.6.0-dev preview
--    where SQL parsing dropped the membership write).
INSERT INTO collection_members (collection_id, user_id, role, added_by)
SELECT c.id, c.created_by, 'owner', c.created_by
FROM collections c
WHERE c.is_personal = TRUE
ON CONFLICT (collection_id, user_id) DO NOTHING;

-- 3. Migrate bookmarked queries into their creator's personal collection.
--    Bookmarks were per-query, not per-user, so the creator is the only
--    honest signal we have. NOTE: Legacy queries with NULL created_by are
--    silently skipped -- there's no user to attribute the bookmark to. If the
--    deployment has a non-trivial number of NULL-creator bookmarked queries,
--    snapshot saved_queries before running this migration.
INSERT INTO collection_items (collection_id, saved_query_id, sort_order, added_by)
SELECT
    pc.id,
    sq.id,
    0,
    sq.created_by
FROM saved_queries sq
JOIN collections pc ON pc.created_by = sq.created_by AND pc.is_personal = TRUE
WHERE sq.is_bookmarked = TRUE AND sq.created_by IS NOT NULL
ON CONFLICT (collection_id, saved_query_id) DO NOTHING;

-- 4. Drop the is_bookmarked column from saved_queries. In Postgres we can
--    DROP COLUMN directly without rebuilding the table; collection_items'
--    FK to saved_queries(id) is preserved transparently.
DROP INDEX IF EXISTS idx_saved_queries_source_bookmark;

ALTER TABLE saved_queries DROP COLUMN is_bookmarked;

CREATE INDEX IF NOT EXISTS idx_saved_queries_source ON saved_queries(source_id);
CREATE INDEX IF NOT EXISTS idx_saved_queries_created_by ON saved_queries(created_by);
