CREATE TABLE consent (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id           UUID NOT NULL REFERENCES person(id),
    consent_type        VARCHAR(30) NOT NULL
                        CHECK (consent_type IN ('DATA_PROCESSING', 'IMAGE_USAGE', 'HEALTH_DATA', 'MINOR_GUARDIAN')),
    consent_version     VARCHAR(20) NOT NULL DEFAULT '1.0',
    purpose             TEXT NOT NULL,
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by_person   UUID REFERENCES person(id),
    signature_data      TEXT,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,
    campus_id           UUID NOT NULL REFERENCES campus(id),
    sync_id             UUID,
    synced_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_person ON consent(person_id);
CREATE INDEX idx_consent_type ON consent(consent_type);
CREATE UNIQUE INDEX uq_consent_active_person_type
    ON consent(person_id, consent_type) WHERE is_active = TRUE;
