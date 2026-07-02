-- The Phase 1 migrations created triage.campaign_id / attendance.campaign_id
-- as bare nullable UUID columns because campaign did not exist yet. Now that
-- 000021 creates it, enforce referential integrity and add the documented
-- indexes (docs/10-data-model.md).
ALTER TABLE triage
    ADD CONSTRAINT fk_triage_campaign FOREIGN KEY (campaign_id) REFERENCES campaign(id);
ALTER TABLE attendance
    ADD CONSTRAINT fk_attendance_campaign FOREIGN KEY (campaign_id) REFERENCES campaign(id);

CREATE INDEX idx_triage_campaign ON triage(campaign_id);
CREATE INDEX idx_attendance_campaign ON attendance(campaign_id);
