-- Search vector trigger for person table
-- Weights: A = full_name, B = document_number, C = email/phone
CREATE OR REPLACE FUNCTION person_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('portuguese', COALESCE(NEW.full_name, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.document_number, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(NEW.email, '')), 'C') ||
        setweight(to_tsvector('simple', COALESCE(NEW.phone, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_person_search_vector
    BEFORE INSERT OR UPDATE ON person
    FOR EACH ROW EXECUTE FUNCTION person_search_vector_update();

-- Backfill existing rows (trigger fires on UPDATE)
UPDATE person SET updated_at = updated_at;
