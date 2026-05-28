CREATE TABLE campus (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(150) NOT NULL,
    region      VARCHAR(20) NOT NULL CHECK (region IN ('BRAZIL', 'USA', 'EUROPE')),
    city        VARCHAR(100),
    state       VARCHAR(100),
    country     VARCHAR(3) NOT NULL DEFAULT 'BRA',
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
