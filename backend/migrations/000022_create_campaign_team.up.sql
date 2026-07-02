CREATE TABLE campaign_team (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id      UUID NOT NULL REFERENCES campaign(id) ON DELETE CASCADE,
    person_id        UUID NOT NULL REFERENCES person(id),
    role_in_campaign VARCHAR(30) NOT NULL
                     CHECK (role_in_campaign IN ('COORDINATOR', 'PROFESSIONAL', 'VOLUNTEER', 'SUPPORT')),
    assigned_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by      UUID,

    CONSTRAINT uq_campaign_person UNIQUE (campaign_id, person_id)
);

CREATE INDEX idx_campaign_team_campaign ON campaign_team(campaign_id);
