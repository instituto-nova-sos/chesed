DROP INDEX IF EXISTS uq_attendance_sync_id;
CREATE INDEX idx_attendance_sync
    ON attendance(sync_id) WHERE sync_id IS NOT NULL;

DROP INDEX IF EXISTS uq_triage_sync_id;
CREATE INDEX idx_triage_sync
    ON triage(sync_id) WHERE sync_id IS NOT NULL;

DROP INDEX IF EXISTS uq_person_sync_id;

ALTER TABLE person
    DROP COLUMN IF EXISTS synced_at,
    DROP COLUMN IF EXISTS sync_id;
