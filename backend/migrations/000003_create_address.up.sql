CREATE TABLE address (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id     UUID NOT NULL REFERENCES person(id) ON DELETE CASCADE,
    street        VARCHAR(300),
    number        VARCHAR(20),
    complement    VARCHAR(100),
    neighborhood  VARCHAR(100),
    city          VARCHAR(100),
    state         VARCHAR(100),
    zip_code      VARCHAR(20),
    country       VARCHAR(3) NOT NULL DEFAULT 'BRA',
    is_primary    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_address_person ON address(person_id);
