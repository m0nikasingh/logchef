-- Add managed flag and secret_ref for declarative provisioning.
-- managed: FALSE = UI-managed (default), TRUE = config-managed.
-- secret_ref: stores the env var name that provided the password (for export round-trip).
ALTER TABLE sources ADD COLUMN managed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE sources ADD COLUMN secret_ref TEXT;
ALTER TABLE teams ADD COLUMN managed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN managed BOOLEAN NOT NULL DEFAULT FALSE;
