-- Sprint 10 (S11.3) — LGPD right to erasure.
--
-- Marks when a person's PII was scrubbed. Set on DATA_PROCESSING consent
-- revocation (consent-driven anonymization) and by the retention sweep. The
-- row is kept for referential integrity; PII columns are overwritten in place.
-- Nullable, so existing rows and seed data stay valid. person is already
-- RLS-enabled (000028), so no policy change is required for a new column.
ALTER TABLE person ADD COLUMN anonymized_at TIMESTAMPTZ;
