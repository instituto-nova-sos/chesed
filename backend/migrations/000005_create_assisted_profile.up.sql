CREATE TABLE assisted_profile (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id           UUID NOT NULL UNIQUE REFERENCES person(id) ON DELETE CASCADE,
    family_composition  TEXT,
    income_range        VARCHAR(30) CHECK (income_range IN ('NO_INCOME', 'UP_TO_1MW', '1_TO_2MW', '2_TO_3MW', 'ABOVE_3MW')),
    housing_situation   VARCHAR(30) CHECK (housing_situation IN ('OWN', 'RENTED', 'BORROWED', 'SHELTER', 'STREET', 'OTHER')),
    education_level     VARCHAR(30) CHECK (education_level IN ('NONE', 'ELEMENTARY', 'HIGH_SCHOOL', 'COLLEGE', 'POST_GRAD')),
    employment_status   VARCHAR(30) CHECK (employment_status IN ('EMPLOYED', 'UNEMPLOYED', 'INFORMAL', 'RETIRED', 'STUDENT', 'OTHER')),
    special_needs       TEXT,
    social_observations TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
