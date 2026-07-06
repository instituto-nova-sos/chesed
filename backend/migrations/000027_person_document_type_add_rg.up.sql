-- Add 'RG' to the allowed person.document_type values.
-- The original CHECK from migration 000002 is an unnamed inline constraint, so
-- we look it up in pg_constraint (the one whose definition references
-- document_type, not the gender check) and drop it before adding a named
-- constraint that includes RG. Idempotent: safe to re-run.
DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    ALTER TABLE person DROP CONSTRAINT IF EXISTS person_document_type_check;

    SELECT con.conname INTO constraint_name
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
    WHERE rel.relname = 'person'
      AND nsp.nspname = 'public'
      AND con.contype = 'c'
      AND pg_get_constraintdef(con.oid) LIKE '%document_type%';

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE person DROP CONSTRAINT %I', constraint_name);
    END IF;

    ALTER TABLE person
        ADD CONSTRAINT person_document_type_check
        CHECK (document_type IN ('CPF', 'RG', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER'));
END $$;
