# 10 - Data Model

## Overview

Relational data model for PostgreSQL 16. All tables use UUID primary keys for offline-safe record creation. Timestamps use `timestamptz`. Soft deletes via `is_active` boolean.

---

## Phase 1 vs Phase 2 Table Categorization

### Phase 1 Tables (MVP)
The following tables are created in Phase 1 database migrations:
- `campus` — Multi-site organizational unit
- `person` — Central unified person entity
- `address` — Person address records
- `person_role` — Multi-role assignment per person
- `assisted_profile` — Extended social data for assisted persons
- `app_user` — System user account (Keycloak projection)
- `service_type` — Predefined service catalog
- `triage` — Initial assessment records
- `triage_requested_service` — Services requested during triage
- `attendance` — Service delivery records
- `attendance_transition` — Workflow state change history
- `audit_log` — Immutable audit trail
- `volunteer_agreement` — Volunteer agreement acceptance tracking

### Phase 2 Tables
The following tables are added in Phase 2 migrations:
- `campaign` — Social action events *(Sprint 5 — implemented)*
- `campaign_team` — Team member assignments *(Sprint 5 — implemented)*
- `document` — File attachments *(Sprint 6 — migration 000025)*
- `consent` — LGPD consent records with signature *(Sprint 6 — migration 000024)*
- `donation` — Financial and in-kind contributions

> **Sprint 5 note**: the Phase 1 migrations created `triage.campaign_id` and
> `attendance.campaign_id` as bare nullable UUID columns (no FK, since
> `campaign` did not exist yet). The Sprint 5 migrations create `campaign` and
> `campaign_team`, then add the `REFERENCES campaign(id)` foreign keys and the
> `idx_triage_campaign` / `idx_attendance_campaign` indexes shown in the DDL
> below.

---

## Tables

### campus

Core multi-site entity. All operational data is scoped to a campus.

```sql
CREATE TABLE campus (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(150) NOT NULL,
    region      VARCHAR(20) NOT NULL CHECK (region IN ('BRAZIL', 'USA', 'EUROPE')),
    city        VARCHAR(100),
    state       VARCHAR(100),
    country     VARCHAR(3) NOT NULL DEFAULT 'BRA',
    timezone    VARCHAR(64) NOT NULL DEFAULT 'America/Sao_Paulo',
    -- Fiscal identity for donation receipts (Sprint 10, S11.5); nullable
    legal_name    VARCHAR(200),
    cnpj          VARCHAR(18),
    address_line  VARCHAR(300),
    city_name     VARCHAR(100),
    state_code    VARCHAR(2),
    zip_code      VARCHAR(9),
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Migrations:**
- `000030_campus_timezone` — Adds `timezone VARCHAR(64) NOT NULL DEFAULT 'America/Sao_Paulo'`. Stores the campus IANA timezone for multi-region time rendering.
- `000032_campus_issuer_data` — Adds nullable `legal_name`, `cnpj`, `address_line`, `city_name`, `state_code`, `zip_code` for the donation-receipt issuer header (Sprint 10). Nullable so existing campuses stay valid; the receipt renders "—" when absent.

### person

Unified person registry. Central entity.

```sql
CREATE TABLE person (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name         VARCHAR(200) NOT NULL,
    birth_date        DATE,
    document_type     VARCHAR(20) NOT NULL DEFAULT 'CPF'
                      CHECK (document_type IN ('CPF', 'RG', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER')),
    document_number   VARCHAR(30),
    gender            VARCHAR(20) CHECK (gender IN ('M', 'F', 'OTHER', 'PREFER_NOT_TO_SAY')),
    email             VARCHAR(255),
    phone             VARCHAR(30),
    photo_url         VARCHAR(500),
    referral_source   VARCHAR(200),
    campus_id         UUID NOT NULL REFERENCES campus(id),
    anonymized_at     TIMESTAMPTZ,   -- set when PII is scrubbed (LGPD erasure, Sprint 10 S11.3)
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by        UUID,

    -- Search optimization
    search_vector     TSVECTOR,

    CONSTRAINT uq_person_document UNIQUE (document_type, document_number)
        WHERE document_number IS NOT NULL
);

CREATE INDEX idx_person_campus ON person(campus_id);
CREATE INDEX idx_person_name ON person(full_name);
CREATE INDEX idx_person_document ON person(document_type, document_number);
CREATE INDEX idx_person_search ON person USING GIN(search_vector);
```

**Migrations:**
- `000027_person_document_type_add_rg` — Adds `RG` to the `document_type` CHECK constraint. Fixes a drift where validators accepted `RG` but the DB CHECK rejected it.
- `000031_person_anonymized_at` — Adds nullable `anonymized_at TIMESTAMPTZ` (Sprint 10, S11.3). Set when a person's PII is scrubbed on `DATA_PROCESSING` consent revocation or by the retention sweep. Rows are kept for referential integrity; PII columns are overwritten (`full_name='[ANONYMIZED]'`, `document_number='ANON-'||id` to satisfy `uq_person_document`, contact fields nulled). The `search_vector` trigger refreshes automatically on the update.

### address

Person address. Separate table for normalization and history.

```sql
CREATE TABLE address (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id     UUID NOT NULL REFERENCES person(id) ON DELETE CASCADE,
    street        VARCHAR(300),
    number        VARCHAR(20),
    complement    VARCHAR(100),
    neighborhood  VARCHAR(100),
    city          VARCHAR(100),
    state         VARCHAR(100),
    zip_code      VARCHAR(20),
    country       VARCHAR(3) NOT NULL DEFAULT 'BRA',
    is_primary    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_address_person ON address(person_id);
```

### person_role

Multi-role association for a person.

```sql
CREATE TABLE person_role (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id               UUID NOT NULL REFERENCES person(id) ON DELETE CASCADE,
    role_type               VARCHAR(30) NOT NULL
                            CHECK (role_type IN ('VOLUNTEER', 'ASSISTED', 'PROFESSIONAL', 'COORDINATOR', 'ADMIN')),
    professional_specialty  VARCHAR(100),
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    activated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at          TIMESTAMPTZ,
    activated_by            UUID,
    deactivated_by          UUID,
    notes                   TEXT,

    CONSTRAINT uq_person_role UNIQUE (person_id, role_type)
);

CREATE INDEX idx_person_role_person ON person_role(person_id);
CREATE INDEX idx_person_role_type ON person_role(role_type);
```

### volunteer_agreement

Tracks volunteer agreement acceptance for persons with operational roles. Added in migrations 000015 and 000016.

```sql
CREATE TABLE volunteer_agreement (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id         UUID NOT NULL REFERENCES person(id) ON DELETE CASCADE,
    person_role_id    UUID NOT NULL REFERENCES person_role(id) ON DELETE CASCADE,
    campus_id         UUID NOT NULL REFERENCES campus(id),
    status            VARCHAR(20) NOT NULL DEFAULT 'PENDING'
                      CHECK (status IN ('PENDING', 'ACCEPTED', 'REJECTED')),
    signature_method  VARCHAR(20)
                      CHECK (signature_method IN ('DIGITAL', 'MANUAL_UPLOAD')),
    accepted_at       TIMESTAMPTZ,
    accepted_by_user  UUID,
    ip_address        INET,
    user_agent        TEXT,
    document_path     VARCHAR(500),
    uploaded_at       TIMESTAMPTZ,
    uploaded_by       UUID,
    rejected_at       TIMESTAMPTZ,
    rejection_reason  TEXT,
    agreement_version VARCHAR(20) NOT NULL DEFAULT '1.0',
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_volunteer_agreement_person ON volunteer_agreement(person_id);
CREATE INDEX idx_volunteer_agreement_role ON volunteer_agreement(person_role_id);
CREATE INDEX idx_volunteer_agreement_campus ON volunteer_agreement(campus_id);
CREATE INDEX idx_volunteer_agreement_status ON volunteer_agreement(status);
CREATE UNIQUE INDEX uq_volunteer_agreement_role ON volunteer_agreement(person_role_id)
    WHERE status IN ('PENDING', 'ACCEPTED');
```

**Migrations:**
- `000015_create_volunteer_agreement` — Creates the `volunteer_agreement` table with all columns and indexes.
- `000016_add_person_unique_constraints` — Adds unique constraints for email and phone per campus:
  - `uq_person_email_campus`: `UNIQUE (email, campus_id) WHERE email IS NOT NULL`
  - `uq_person_phone_campus`: `UNIQUE (phone, campus_id) WHERE phone IS NOT NULL`

### assisted_profile

Extended data for persons with ASSISTED role. Contains sensitive social information.

```sql
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
```

### app_user

Local projection of a Keycloak identity. Credentials are NOT stored locally. The `keycloak_subject_id` links to the external identity provider (Keycloak `sub` claim).

```sql
CREATE TABLE app_user (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id             UUID NOT NULL UNIQUE REFERENCES person(id),
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
```

### service_type

Catalog of available services.

```sql
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
```

### campaign

Social action events.

```sql
CREATE TABLE campaign (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(200) NOT NULL,
    description     TEXT,
    campaign_type   VARCHAR(30) NOT NULL
                    CHECK (campaign_type IN ('SOCIAL_ACTION', 'EDUCATIONAL', 'HEALTH', 'COMMUNITY', 'OTHER')),
    start_date      DATE NOT NULL,
    end_date        DATE,
    location_name   VARCHAR(200),
    location_address TEXT,
    campus_id       UUID NOT NULL REFERENCES campus(id),
    coordinator_id  UUID REFERENCES person(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'PLANNED'
                    CHECK (status IN ('PLANNED', 'ACTIVE', 'COMPLETED', 'CANCELLED')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID
);

CREATE INDEX idx_campaign_campus ON campaign(campus_id);
CREATE INDEX idx_campaign_status ON campaign(status);
CREATE INDEX idx_campaign_date ON campaign(start_date);
```

### campaign_team

Team assignment for campaigns.

```sql
CREATE TABLE campaign_team (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id         UUID NOT NULL REFERENCES campaign(id) ON DELETE CASCADE,
    person_id           UUID NOT NULL REFERENCES person(id),
    role_in_campaign    VARCHAR(30) NOT NULL
                        CHECK (role_in_campaign IN ('COORDINATOR', 'PROFESSIONAL', 'VOLUNTEER', 'SUPPORT')),
    assigned_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by         UUID,

    CONSTRAINT uq_campaign_person UNIQUE (campaign_id, person_id)
);

CREATE INDEX idx_campaign_team_campaign ON campaign_team(campaign_id);
```

### triage

Initial assessment record.

```sql
CREATE TABLE triage (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id       UUID NOT NULL REFERENCES person(id),
    campaign_id     UUID REFERENCES campaign(id),
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
CREATE INDEX idx_triage_campaign ON triage(campaign_id);
CREATE INDEX idx_triage_campus ON triage(campus_id);
CREATE INDEX idx_triage_date ON triage(triage_date);
CREATE INDEX idx_triage_sync ON triage(sync_id) WHERE sync_id IS NOT NULL;
```

### triage_requested_service

Many-to-many: triage → service types requested.

```sql
CREATE TABLE triage_requested_service (
    triage_id       UUID NOT NULL REFERENCES triage(id) ON DELETE CASCADE,
    service_type_id UUID NOT NULL REFERENCES service_type(id),
    PRIMARY KEY (triage_id, service_type_id)
);
```

### attendance

Service record.

```sql
CREATE TABLE attendance (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id       UUID NOT NULL REFERENCES person(id),
    triage_id       UUID REFERENCES triage(id),
    campaign_id     UUID REFERENCES campaign(id),
    campus_id       UUID NOT NULL REFERENCES campus(id),
    service_type_id UUID NOT NULL REFERENCES service_type(id),
    professional_id UUID NOT NULL REFERENCES person(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED'
                    CHECK (status IN ('SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'FOLLOW_UP', 'CANCELLED')),
                    -- MVP (Phase 1) uses: SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED.
                    -- FOLLOW_UP is added in a Phase 2 migration.
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
CREATE INDEX idx_attendance_campaign ON attendance(campaign_id);
CREATE INDEX idx_attendance_sync ON attendance(sync_id) WHERE sync_id IS NOT NULL;
```

### attendance_transition

Workflow state change history.

```sql
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
```

### document

File attachments. Implemented by migration `000025_create_document` (Sprint 6).

> Sprint 6 note: the original Phase-2 draft of this table lacked `campus_id`,
> `is_active`, and `created_at`/`updated_at`, violating the design principles
> at the top of this document and the CLAUDE.md guardrails (campus scoping on
> all operational tables, timestamps on all tables). The DDL below is the
> corrected, implemented shape. `file_path` stores the object-storage key
> (`documents/{campus_id}/{person_id}/{uuid}{ext}`), never a filesystem path;
> downloads are served via presigned URLs (docs/19).

```sql
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
```

### consent

LGPD consent records. Implemented by migration `000024_create_consent` (Sprint 6).

> Sprint 6 notes: `created_at`/`updated_at` added for convention compliance
> (`granted_at` remains the domain-meaningful grant timestamp). The active
> index is a **partial UNIQUE** index so the SQL layer enforces "at most one
> active consent per person and type" — a duplicate active grant surfaces as
> a 23505 and maps to HTTP 409 `duplicate`. Re-granting after a revocation is
> allowed (a new row; the revoked row stays for the audit trail, docs/13).
> `sync_id`/`synced_at` are reserved for a future offline slice; consent
> capture is **online-only** in Sprint 6 (docs/12).

```sql
CREATE TABLE consent (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id           UUID NOT NULL REFERENCES person(id),
    consent_type        VARCHAR(30) NOT NULL
                        CHECK (consent_type IN ('DATA_PROCESSING', 'IMAGE_USAGE', 'HEALTH_DATA', 'MINOR_GUARDIAN')),
    consent_version     VARCHAR(20) NOT NULL DEFAULT '1.0',
    purpose             TEXT NOT NULL,
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by_person   UUID REFERENCES person(id),
    signature_data      TEXT,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,
    campus_id           UUID NOT NULL REFERENCES campus(id),
    sync_id             UUID,
    synced_at           TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_person ON consent(person_id);
CREATE INDEX idx_consent_type ON consent(consent_type);
CREATE UNIQUE INDEX uq_consent_active_person_type
    ON consent(person_id, consent_type) WHERE is_active = TRUE;
```

### donation

Financial and in-kind contributions.

```sql
CREATE TABLE donation (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    donor_person_id     UUID REFERENCES person(id),
    campaign_id         UUID REFERENCES campaign(id),
    campus_id           UUID NOT NULL REFERENCES campus(id),
    donation_type       VARCHAR(20) NOT NULL
                        CHECK (donation_type IN ('FINANCIAL', 'GOODS', 'SERVICES')),
    amount              DECIMAL(12, 2),
    currency            VARCHAR(3) NOT NULL DEFAULT 'BRL'
                        CHECK (currency IN ('BRL', 'USD', 'EUR')),
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
```

**Migrations:**
- `000029_donation_currency_check` — Adds `CHECK (currency IN ('BRL', 'USD', 'EUR'))`. Amounts are stored in their native currency; no FX conversion is applied.

### audit_log

Append-only audit trail.

```sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID,
    action_type VARCHAR(30) NOT NULL
                CHECK (action_type IN ('CREATE', 'READ', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT', 'EXPORT', 'PERMISSION_CHANGE')),
    entity_type VARCHAR(50) NOT NULL,
    entity_id   UUID,
    module      VARCHAR(50),
    description TEXT,
    old_values  JSONB,
    new_values  JSONB,
    ip_address  INET,
    user_agent  TEXT,
    campus_id   UUID,
    success     BOOLEAN NOT NULL DEFAULT TRUE,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_user ON audit_log(user_id);
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_module ON audit_log(module);
CREATE INDEX idx_audit_campus ON audit_log(campus_id);
```

**Note**: This table is append-only. No UPDATE or DELETE operations. Consider partitioning by month for large volumes.

### ~~refresh_token~~ (Removed)

Refresh tokens are managed by Keycloak. No application-level token storage is needed.

---

## Audit Strategy

1. **Application-level logging**: Go middleware writes to `audit_log` on every mutation and sensitive read
2. **Field-level change tracking**: `old_values` and `new_values` JSONB columns store diffs
3. **Immutable audit table**: No updates or deletes allowed (enforced by application layer and optionally by DB trigger)
4. **Retention**: Audit logs retained indefinitely for compliance; partitioned by month for performance
5. **Query patterns**: Filter by user, entity, timestamp, module, campus
6. **Keycloak event integration**: Login/logout events are captured by Keycloak's event listener. Events can be forwarded to the application's audit_log table via Keycloak SPI event listener or Admin Events API polling for a unified audit trail.

---

## Multi-Region / Campus Segregation

### Application-Level (MVP)
- All queries include `WHERE campus_id = ?` from JWT claims
- Go middleware injects campus filter automatically
- Admin users can query across campuses with explicit parameter

### Database-Level Row-Level Security (RLS)

Migration `000028_enable_rls` enables PostgreSQL RLS as a defense-in-depth layer for
campus isolation. Application-level `WHERE campus_id` filtering is **retained** — RLS is
additive, not a replacement.

- **Session GUC**: each request transaction sets `app.current_campus` (via `set_config(..., is_local => true)`) from the authenticated user's campus. Policies read it with `current_setting('app.current_campus', true)`.
- **Fail-closed**: an unset GUC returns `NULL`, which matches no rows (zero rows), never all rows.
- **App role**: the application connects as the non-owner role `chesed_app`, so RLS applies to it. Migrations and seed run as the owner role, which bypasses RLS — no GUC needed for schema operations.
- **Covered tables**: `person`, `address`, `triage`, `triage_requested_service`, `attendance`, `attendance_transition`, `assisted_profile`, `campaign`, `campaign_team`, `consent`, `document`, `donation`.
- **Policy shape**:
  - Direct-column tables (`person`, `triage`, `attendance`, `campaign`, `consent`, `document`, `donation`) match `campus_id = current_setting('app.current_campus', true)::uuid`.
  - Join-inherited tables (`address`, `assisted_profile`, `campaign_team`, `attendance_transition`, `triage_requested_service`) have no `campus_id` column and isolate via an `EXISTS` check against their parent's `campus_id`.
- **Excluded (by design)**:
  - `audit_log` — nullable `campus_id`; records pre-campus events (login/provisioning denials). Append-only and application-filtered.
  - `volunteer_agreement` — carries `campus_id` but is reached only through a person the caller has already resolved (person is RLS-protected), and is written on the pre-campus self-registration path; left out of RLS to keep that path working. Access is transitively campus-bounded via person.
  - `person_role`, `app_user`, `service_type`, `campus` — identity/reference tables that are queried before a campus is established (provisioning, onboarding) or are the tenant boundary itself; their repositories run on the owner pool.
  - The completeness guarantee ("a repository that forgets its `WHERE campus_id` clause cannot leak") therefore holds for the **Covered tables** above; the excluded tables are enumerated here rather than silently uncovered.
- **Pre-campus routes**: `/self-register` and `/auth/me` run before a campus GUC exists (self-registration takes the campus from the request body; onboarding does a deliberate cross-campus email lookup). They are wrapped in a `BypassRLS` middleware that installs the owner connection, so these flows are not fail-closed by RLS. The owner path still writes `campus_id` explicitly, so data integrity is preserved.

```sql
-- Direct-column policy (migration 000028)
ALTER TABLE person ENABLE ROW LEVEL SECURITY;
CREATE POLICY person_campus_isolation ON person
    USING (campus_id = current_setting('app.current_campus', true)::uuid)
    WITH CHECK (campus_id = current_setting('app.current_campus', true)::uuid);

-- Join-inherited policy (migration 000028)
ALTER TABLE address ENABLE ROW LEVEL SECURITY;
CREATE POLICY address_campus_isolation ON address
    USING (EXISTS (SELECT 1 FROM person p
                   WHERE p.id = address.person_id
                     AND p.campus_id = current_setting('app.current_campus', true)::uuid));
```

---

## Indexing Strategy

| Table | Key Indexes | Purpose |
|-------|-----------|---------|
| person | campus_id, document, name, search_vector (GIN), email+campus (unique), phone+campus (unique) | Lookup, search, deduplication |
| volunteer_agreement | person_id, person_role_id, campus_id, status | Agreement lookup, filtering |
| attendance | person_id, professional_id, campus_id, status, date | List, filter, dashboard |
| triage | person_id, campaign_id, campus_id, date | List, filter |
| audit_log | user_id, timestamp, entity, module, campus_id | Compliance queries |
| campaign | campus_id, status, date | List, dashboard |

---

## Migration Numbering Convention

```
migrations/
├── 000001_create_campus.up.sql
├── 000001_create_campus.down.sql
├── 000002_create_person.up.sql
├── 000002_create_person.down.sql
├── 000003_create_address.up.sql
├── 000003_create_address.down.sql
├── ...
```

Each migration has an `up` (apply) and `down` (rollback) file. Numbered sequentially.

---

## Data Retention Policy

| Data Category | Retention Period | After Expiry | Legal Basis |
|---------------|-----------------|--------------|-------------|
| Person records | 5 years after last activity | Anonymize (keep aggregate stats) | LGPD Art. 15-16 |
| Attendance records | 10 years | Archive (read-only) | Operational + compliance |
| Triage records | 10 years | Archive (read-only) | Linked to attendance |
| Assisted profile (sensitive) | 5 years after last activity | Anonymize | LGPD Art. 18 |
| Consent records | Indefinite | Never deleted | Legal proof of consent |
| Audit logs | 10 years minimum | Archive (read-only) | Compliance requirement |
| Donation records | 10 years | Archive (read-only) | Tax/accounting requirements |
| Campaign records | 5 years after completion | Archive | Operational |

### LGPD Erasure (Right to Deletion)
When a data subject exercises their right to erasure:
1. Anonymize the `person` record: replace `full_name` with 'ANONYMIZED', clear `document_number`, `email`, `phone`, `photo_url`
2. Anonymize the `address` record: clear all fields, keep `campus_id` for aggregate stats
3. Anonymize the `assisted_profile`: clear all sensitive fields
4. Keep `attendance` and `triage` records with anonymized person reference for aggregate reporting
5. Keep `audit_log` entries (they reference user_id, not PII directly)
6. Revoke all active `consent` records
7. Log the erasure action in `audit_log`

Note: Complete physical deletion is performed only when legally required. Anonymization preserves aggregate data integrity.
