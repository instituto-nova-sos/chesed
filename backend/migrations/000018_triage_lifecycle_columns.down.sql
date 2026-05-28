DROP INDEX IF EXISTS idx_triage_active;

ALTER TABLE triage
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS updated_at;
