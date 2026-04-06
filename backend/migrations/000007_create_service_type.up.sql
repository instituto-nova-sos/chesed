CREATE TABLE service_type (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(30) NOT NULL
                CHECK (category IN ('LEGAL', 'MEDICAL', 'NUTRITIONAL', 'PHYSIOTHERAPY', 'SOCIAL', 'EDUCATIONAL', 'PSYCHOLOGICAL', 'OTHER')),
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
