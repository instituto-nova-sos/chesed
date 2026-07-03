DROP INDEX IF EXISTS idx_attendance_campaign;
DROP INDEX IF EXISTS idx_triage_campaign;
ALTER TABLE attendance DROP CONSTRAINT IF EXISTS fk_attendance_campaign;
ALTER TABLE triage DROP CONSTRAINT IF EXISTS fk_triage_campaign;
