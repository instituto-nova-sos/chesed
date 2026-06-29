ALTER TABLE person
    ADD COLUMN sync_id   UUID,
    ADD COLUMN synced_at TIMESTAMPTZ;

CREATE UNIQUE INDEX uq_person_sync_id
    ON person(sync_id) WHERE sync_id IS NOT NULL;

DROP INDEX IF EXISTS idx_triage_sync;
CREATE UNIQUE INDEX uq_triage_sync_id
    ON triage(sync_id) WHERE sync_id IS NOT NULL;

DROP INDEX IF EXISTS idx_attendance_sync;
CREATE UNIQUE INDEX uq_attendance_sync_id
    ON attendance(sync_id) WHERE sync_id IS NOT NULL;
