-- volunteer_agreement: tracks volunteer agreement acceptance (digital or manual upload)
CREATE TABLE volunteer_agreement (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id         UUID NOT NULL REFERENCES person(id),
    person_role_id    UUID NOT NULL REFERENCES person_role(id),
    campus_id         UUID NOT NULL REFERENCES campus(id),
    status            VARCHAR(20) NOT NULL DEFAULT 'PENDING'
                      CHECK (status IN ('PENDING', 'ACCEPTED', 'REJECTED')),
    signature_method  VARCHAR(20)
                      CHECK (signature_method IN ('DIGITAL', 'MANUAL_UPLOAD')),

    -- Digital signature fields
    accepted_at       TIMESTAMPTZ,
    accepted_by_user  UUID REFERENCES app_user(id),
    ip_address        INET,
    user_agent        TEXT,

    -- Manual upload fields
    document_path     VARCHAR(500),
    uploaded_at       TIMESTAMPTZ,
    uploaded_by       UUID REFERENCES app_user(id),

    -- Rejection fields
    rejected_at       TIMESTAMPTZ,
    rejection_reason  TEXT,

    -- Common fields
    agreement_version VARCHAR(20) NOT NULL DEFAULT 'v1',
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_volunteer_agreement_person ON volunteer_agreement(person_id);
CREATE INDEX idx_volunteer_agreement_status ON volunteer_agreement(status);
CREATE INDEX idx_volunteer_agreement_campus ON volunteer_agreement(campus_id);
CREATE UNIQUE INDEX uq_volunteer_agreement_person_role ON volunteer_agreement(person_role_id);
