CREATE TABLE app_user (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id             UUID UNIQUE REFERENCES person(id),
    email                 VARCHAR(255) NOT NULL UNIQUE,
    keycloak_subject_id   VARCHAR(255) NOT NULL UNIQUE,
    access_profile        VARCHAR(30) NOT NULL
                          CHECK (access_profile IN ('ADMIN', 'COORDINATOR', 'PROFESSIONAL', 'SECRETARY', 'VOLUNTEER')),
    campus_id             UUID NOT NULL REFERENCES campus(id),
    is_active             BOOLEAN NOT NULL DEFAULT TRUE,
    last_login            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_email ON app_user(email);
CREATE INDEX idx_user_campus ON app_user(campus_id);
CREATE UNIQUE INDEX idx_user_keycloak_sub ON app_user(keycloak_subject_id);
