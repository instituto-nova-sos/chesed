CREATE TABLE attendance (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id       UUID NOT NULL REFERENCES person(id),
    triage_id       UUID REFERENCES triage(id),
    campaign_id     UUID,
    campus_id       UUID NOT NULL REFERENCES campus(id),
    service_type_id UUID NOT NULL REFERENCES service_type(id),
    professional_id UUID NOT NULL REFERENCES person(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED'
                    CHECK (status IN ('SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'FOLLOW_UP', 'CANCELLED')),
    attendance_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    observations    TEXT,
    recommendations TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID,
    sync_id         UUID,
    synced_at       TIMESTAMPTZ
);

CREATE INDEX idx_attendance_person ON attendance(person_id);
CREATE INDEX idx_attendance_professional ON attendance(professional_id);
CREATE INDEX idx_attendance_campus ON attendance(campus_id);
CREATE INDEX idx_attendance_status ON attendance(status);
CREATE INDEX idx_attendance_date ON attendance(attendance_date);
CREATE INDEX idx_attendance_sync ON attendance(sync_id) WHERE sync_id IS NOT NULL;

CREATE TABLE attendance_transition (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attendance_id   UUID NOT NULL REFERENCES attendance(id) ON DELETE CASCADE,
    from_status     VARCHAR(20) NOT NULL,
    to_status       VARCHAR(20) NOT NULL,
    reason          TEXT,
    transitioned_by UUID NOT NULL,
    transitioned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transition_attendance ON attendance_transition(attendance_id);
