CREATE TABLE document (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id       UUID NOT NULL REFERENCES person(id),
    attendance_id   UUID REFERENCES attendance(id),
    campus_id       UUID NOT NULL REFERENCES campus(id),
    document_type   VARCHAR(30) NOT NULL
                    CHECK (document_type IN ('ID', 'PROOF_OF_RESIDENCE', 'MEDICAL_RECORD', 'EXAM', 'CONSENT', 'PHOTO', 'OTHER')),
    file_name       VARCHAR(255) NOT NULL,
    file_path       VARCHAR(500) NOT NULL,
    file_size       INTEGER,
    mime_type       VARCHAR(100),
    description     TEXT,
    uploaded_by     UUID NOT NULL,
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_person ON document(person_id);
CREATE INDEX idx_document_attendance ON document(attendance_id);
CREATE INDEX idx_document_campus ON document(campus_id);
