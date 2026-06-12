CREATE TABLE IF NOT EXISTS query_folders (
    id BIGSERIAL PRIMARY KEY,
    team_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    color TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(team_id, name)
);

CREATE TABLE IF NOT EXISTS query_folder_items (
    folder_id BIGINT NOT NULL,
    query_id BIGINT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    added_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (folder_id, query_id),
    FOREIGN KEY (folder_id) REFERENCES query_folders(id) ON DELETE CASCADE,
    FOREIGN KEY (query_id) REFERENCES team_queries(id) ON DELETE CASCADE,
    FOREIGN KEY (added_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_query_folders_team_id ON query_folders(team_id);
CREATE INDEX IF NOT EXISTS idx_query_folders_team_sort ON query_folders(team_id, sort_order, name);
CREATE INDEX IF NOT EXISTS idx_query_folder_items_query_id ON query_folder_items(query_id);
CREATE INDEX IF NOT EXISTS idx_query_folder_items_folder_order ON query_folder_items(folder_id, sort_order);
