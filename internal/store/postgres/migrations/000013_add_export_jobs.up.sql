CREATE TABLE IF NOT EXISTS export_jobs (
    id TEXT PRIMARY KEY,
    team_id BIGINT NOT NULL,
    source_id BIGINT NOT NULL,
    created_by BIGINT NOT NULL,
    status TEXT NOT NULL,
    format TEXT NOT NULL,
    request_json TEXT NOT NULL,
    file_name TEXT,
    file_path TEXT,
    error_message TEXT,
    rows_exported BIGINT NOT NULL DEFAULT 0,
    bytes_written BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_team_source ON export_jobs(team_id, source_id);
CREATE INDEX IF NOT EXISTS idx_export_jobs_created_by ON export_jobs(created_by);
CREATE INDEX IF NOT EXISTS idx_export_jobs_expires_at ON export_jobs(expires_at);
