CREATE TABLE person (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name         VARCHAR(200) NOT NULL,
    birth_date        DATE,
    document_type     VARCHAR(20) NOT NULL DEFAULT 'CPF'
                      CHECK (document_type IN ('CPF', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER')),
    document_number   VARCHAR(30),
    gender            VARCHAR(20) CHECK (gender IN ('M', 'F', 'OTHER', 'PREFER_NOT_TO_SAY')),
    email             VARCHAR(255),
    phone             VARCHAR(30),
    photo_url         VARCHAR(500),
    referral_source   VARCHAR(200),
    campus_id         UUID NOT NULL REFERENCES campus(id),
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        UUID,
    search_vector     TSVECTOR,
    CONSTRAINT uq_person_document UNIQUE NULLS NOT DISTINCT (document_type, document_number)
);

CREATE INDEX idx_person_campus ON person(campus_id);
CREATE INDEX idx_person_name ON person(full_name);
CREATE INDEX idx_person_document ON person(document_type, document_number);
CREATE INDEX idx_person_search ON person USING GIN(search_vector);
