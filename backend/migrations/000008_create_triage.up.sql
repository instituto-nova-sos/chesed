CREATE TABLE triage (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id       UUID NOT NULL REFERENCES person(id),
    campaign_id     UUID,
    campus_id       UUID NOT NULL REFERENCES campus(id),
    main_complaint  TEXT NOT NULL,
    assigned_team   UUID REFERENCES person(id),
    triage_date     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    location        VARCHAR(300),
    triaged_by      UUID NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sync_id         UUID,
    synced_at       TIMESTAMPTZ
);

CREATE INDEX idx_triage_person ON triage(person_id);
CREATE INDEX idx_triage_campus ON triage(campus_id);
CREATE INDEX idx_triage_date ON triage(triage_date);
CREATE INDEX idx_triage_sync ON triage(sync_id) WHERE sync_id IS NOT NULL;

CREATE TABLE triage_requested_service (
    triage_id       UUID NOT NULL REFERENCES triage(id) ON DELETE CASCADE,
    service_type_id UUID NOT NULL REFERENCES service_type(id),
    PRIMARY KEY (triage_id, service_type_id)
);
