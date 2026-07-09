-- Adds a column to store the drawn digital signature (base64 PNG data URL) for
-- volunteer agreements accepted via the DIGITAL method, mirroring consent.signature_data.
-- Nullable: MANUAL_UPLOAD acceptances and legacy DIGITAL rows have no drawn signature.
ALTER TABLE volunteer_agreement ADD COLUMN signature_data TEXT;
