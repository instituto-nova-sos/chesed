ALTER TABLE person ADD COLUMN nationality VARCHAR(3) NOT NULL DEFAULT 'BRA';
CREATE INDEX idx_person_nationality ON person(nationality);
