-- Revert person.document_type to the original allowed values (without RG).
-- Rows with document_type = 'RG' must be reconciled before this migration is
-- applied; the CHECK will otherwise reject them.
ALTER TABLE person DROP CONSTRAINT IF EXISTS person_document_type_check;

ALTER TABLE person
    ADD CONSTRAINT person_document_type_check
    CHECK (document_type IN ('CPF', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER'));
