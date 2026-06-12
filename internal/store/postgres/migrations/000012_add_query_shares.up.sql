CREATE TABLE IF NOT EXISTS query_shares (
    token TEXT PRIMARY KEY,
    team_id BIGINT NOT NULL,
    source_id BIGINT NOT NULL,
    created_by BIGINT NOT NULL,
    payload_json TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_accessed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_query_shares_team_source ON query_shares(team_id, source_id);
CREATE INDEX IF NOT EXISTS idx_query_shares_created_by ON query_shares(created_by);
CREATE INDEX IF NOT EXISTS idx_query_shares_expires_at ON query_shares(expires_at);
