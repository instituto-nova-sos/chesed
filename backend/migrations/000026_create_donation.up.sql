CREATE TABLE donation (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    donor_person_id     UUID REFERENCES person(id),
    campaign_id         UUID REFERENCES campaign(id),
    campus_id           UUID NOT NULL REFERENCES campus(id),
    donation_type       VARCHAR(20) NOT NULL
                        CHECK (donation_type IN ('FINANCIAL', 'GOODS', 'SERVICES')),
    amount              DECIMAL(12, 2),
    currency            VARCHAR(3) NOT NULL DEFAULT 'BRL',
    item_description    TEXT,
    donation_date       DATE NOT NULL DEFAULT CURRENT_DATE,
    receipt_number      VARCHAR(50) UNIQUE,
    receipt_issued_at   TIMESTAMPTZ,
    notes               TEXT,
    registered_by       UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_donation_campus ON donation(campus_id);
CREATE INDEX idx_donation_campaign ON donation(campaign_id);
CREATE INDEX idx_donation_donor ON donation(donor_person_id);
CREATE INDEX idx_donation_date ON donation(donation_date);
