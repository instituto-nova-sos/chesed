CREATE TABLE campaign (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(200) NOT NULL,
    description      TEXT,
    campaign_type    VARCHAR(30) NOT NULL
                     CHECK (campaign_type IN ('SOCIAL_ACTION', 'EDUCATIONAL', 'HEALTH', 'COMMUNITY', 'OTHER')),
    start_date       DATE NOT NULL,
    end_date         DATE,
    location_name    VARCHAR(200),
    location_address TEXT,
    campus_id        UUID NOT NULL REFERENCES campus(id),
    coordinator_id   UUID REFERENCES person(id),
    status           VARCHAR(20) NOT NULL DEFAULT 'PLANNED'
                     CHECK (status IN ('PLANNED', 'ACTIVE', 'COMPLETED', 'CANCELLED')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       UUID
);

CREATE INDEX idx_campaign_campus ON campaign(campus_id);
CREATE INDEX idx_campaign_status ON campaign(status);
CREATE INDEX idx_campaign_date ON campaign(start_date);
