CREATE TABLE person_role (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id               UUID NOT NULL REFERENCES person(id) ON DELETE CASCADE,
    role_type               VARCHAR(30) NOT NULL
                            CHECK (role_type IN ('VOLUNTEER', 'ASSISTED', 'PROFESSIONAL', 'COORDINATOR', 'ADMIN')),
    professional_specialty  VARCHAR(100),
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    activated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at          TIMESTAMPTZ,
    activated_by            UUID,
    deactivated_by          UUID,
    notes                   TEXT,
    CONSTRAINT uq_person_role UNIQUE (person_id, role_type)
);

CREATE INDEX idx_person_role_person ON person_role(person_id);
CREATE INDEX idx_person_role_type ON person_role(role_type);
