-- Cross-team Collections. Each user has a "personal" collection auto-created
-- on first /api/v1/collections fetch (handled in app code). Other collections
-- are invite-only with two roles: owner (full control) and member (read +
-- bookmark). Items are saved-query references; an item visible to a member
-- who lacks source access still surfaces with a runnable: false flag from the
-- application layer.

-- created_by is nullable + ON DELETE SET NULL so deleting a user does NOT
-- destroy shared collections their teammates rely on. The collection becomes
-- ownerless; remaining members keep access, and a global admin can manage it.
-- For personal collections (is_personal = TRUE), the unique partial index
-- below means a user can have at most one -- and when the user is deleted the
-- row is left as a tombstone with NULL created_by, which is semantically fine
-- (no one will ever see it because nobody else is a member of a personal
-- collection by definition).
CREATE TABLE collections (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    is_personal BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Each user gets at most one personal collection. The unique partial index
-- only applies when is_personal = TRUE, so users can still own multiple shared
-- collections.
CREATE UNIQUE INDEX idx_collections_one_personal_per_user
    ON collections(created_by) WHERE is_personal = TRUE;
CREATE INDEX idx_collections_created_by ON collections(created_by);

CREATE TABLE collection_members (
    collection_id BIGINT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    added_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collection_id, user_id)
);

CREATE INDEX idx_collection_members_user ON collection_members(user_id);

CREATE TABLE collection_items (
    collection_id BIGINT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    saved_query_id BIGINT NOT NULL REFERENCES saved_queries(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    added_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collection_id, saved_query_id)
);

CREATE INDEX idx_collection_items_query ON collection_items(saved_query_id);
CREATE INDEX idx_collection_items_collection_order
    ON collection_items(collection_id, sort_order, saved_query_id);
