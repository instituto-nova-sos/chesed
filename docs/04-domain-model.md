# 04 - Domain Model

## Overview

The domain model is centered on a **unified Person entity** that can hold multiple roles (assisted, volunteer, professional, coordinator, admin). All operational activities — triage, attendance, campaigns, donations — relate back to Person records.

---

## Core Domain Entities

### 1. Person (Pessoa)

The central entity. Every human in the system — whether assisted, volunteer, or professional — is first and foremost a Person.

```
Person
├── id (UUID)
├── full_name
├── birth_date
├── document_type (CPF, RG, SSN, EU_ID, PASSPORT, OTHER)
├── document_number (unique per type)
├── gender (M, F, OTHER, PREFER_NOT_TO_SAY)
├── email
├── phone
├── address (embedded or related)
│   ├── street
│   ├── number
│   ├── complement
│   ├── neighborhood
│   ├── city
│   ├── state
│   ├── zip_code
│   └── country
├── campus_id (FK → Campus)
├── referral_source
├── photo_url
├── is_active
├── created_at
├── updated_at
└── created_by (FK → User)
```

**Design decisions:**
- UUID as primary key for offline creation without collision
- Document type + number allows international support: `CPF`, `RG`, `SSN`, `EU_ID`, and `PASSPORT` cover Brazilian and international identity documents (RF-02a)
- Address is a separate struct/table for normalization
- Campus association enables data segregation

### 2. PersonRole (PapelPessoa)

Associates a Person with one or more business roles. A person can be simultaneously a beneficiary and a volunteer.

```
PersonRole
├── id (UUID)
├── person_id (FK → Person)
├── role_type (ASSISTED, VOLUNTEER, PROFESSIONAL, COORDINATOR, ADMIN)
├── professional_specialty (nullable — only for PROFESSIONAL role)
├── is_active
├── activated_at
├── deactivated_at
├── activated_by (FK → User)
├── deactivated_by (FK → User)
└── notes
```

**Supported roles:**
| Role | Description |
|------|------------|
| `ASSISTED` | Beneficiary receiving services |
| `VOLUNTEER` | Community member assisting in operations |
| `PROFESSIONAL` | Licensed professional providing specialized services |
| `COORDINATOR` | Team lead managing campaigns and volunteers |
| `ADMIN` | System administrator |

**Role Hierarchy Rule:**
- `VOLUNTEER` is the base operational role. When a person is assigned `PROFESSIONAL`, `COORDINATOR`, or `ADMIN`, the `VOLUNTEER` role is automatically added if not already present.
- `ASSISTED` is independent and does not imply or require `VOLUNTEER`.
- This hierarchy is enforced at the service layer during role assignment, not at the database level.

> **Person Roles vs. Access Profiles**
>
> These are two distinct taxonomies:
> - **Person Roles** (ASSISTED, VOLUNTEER, PROFESSIONAL, COORDINATOR, ADMIN): Describe what someone IS in the NGO. A person can hold multiple roles simultaneously. Stored in the `person_role` table.
> - **Access Profiles** (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER): Describe system login permissions. Each `app_user` has exactly one access profile. Stored in the `app_user` table.
>
> Example: A person who is both a beneficiary and a volunteer has person roles [ASSISTED, VOLUNTEER] and might have an app_user with access_profile VOLUNTEER.
>
> Note: SECRETARY is an access profile only (not a person role). It represents staff who register persons and create triages but are not volunteers or professionals in the field.

### 2.1. VolunteerAgreement (TermoVoluntariado)

Tracks the acceptance or rejection of the volunteer agreement for persons with operational roles (VOLUNTEER, PROFESSIONAL, COORDINATOR, ADMIN). Volunteers must accept the agreement before accessing platform features.

```
VolunteerAgreement
├── id (UUID)
├── person_id (FK → Person)
├── person_role_id (FK → PersonRole)
├── campus_id (FK → Campus)
├── status (PENDING, ACCEPTED, REJECTED)
├── signature_method (DIGITAL, MANUAL_UPLOAD)
├── accepted_at (nullable)
├── accepted_by_user (FK → User, nullable)
├── ip_address (nullable)
├── user_agent (nullable)
├── document_path (nullable — for manual uploads)
├── uploaded_at (nullable)
├── uploaded_by (FK → User, nullable)
├── rejected_at (nullable)
├── rejection_reason (nullable)
├── agreement_version
├── notes (nullable)
├── created_at
└── updated_at
```

**Design decisions:**
- Linked to both `person_id` and `person_role_id` for traceability (which role triggered the agreement).
- Supports two signature methods: `DIGITAL` (self-service acceptance via the platform) and `MANUAL_UPLOAD` (coordinator uploads a signed physical document).
- Rejection is recorded in the database; the person remains visible for coordinator follow-up but cannot access platform features.
- `agreement_version` enables tracking which version of the agreement text the person accepted.

### 3. AssistedProfile (PerfilAssistido)

Extended information for persons with the ASSISTED role. Contains sensitive social data.

```
AssistedProfile
├── id (UUID)
├── person_id (FK → Person, unique)
├── family_composition
├── income_range (ENUM)
├── housing_situation (ENUM)
├── education_level (ENUM)
├── employment_status (ENUM)
├── special_needs
├── social_observations (text)
├── created_at
└── updated_at
```

### 4. User (Usuario)

System access account linked to a Person. Not all persons have user accounts.

```
User
├── id (UUID)
├── person_id (FK → Person, unique)
├── keycloak_subject_id (unique, links to Keycloak identity)
├── email (unique, used for login)
├── access_profile (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER)
├── campus_id (FK → Campus)
├── is_active
├── last_login
├── created_at
└── updated_at
```

> **Note**: Credentials (passwords, MFA tokens) are managed by Keycloak, not stored locally. The `keycloak_subject_id` is the foreign key into the external identity provider. The `app_user` table is a local projection of Keycloak identity.

### 5. Campus

Represents a physical location / church campus. Enables multi-site data segregation.

```
Campus
├── id (UUID)
├── name
├── region (BRAZIL, USA, EUROPE)
├── city
├── state
├── country
├── timezone (IANA, default America/Sao_Paulo)
├── is_active
├── created_at
└── updated_at
```

The `timezone` attribute holds the campus IANA timezone (e.g. `America/Sao_Paulo`, `America/New_York`, `Europe/Lisbon`) so times are rendered in the campus's local zone across regions. It defaults to `America/Sao_Paulo`.

**MVP scope**: Each person belongs to exactly one campus. Each app_user is associated with exactly one campus (via Keycloak user attribute). Multi-campus assignment for persons and users is planned for Phase 2.

---

## Service Domain

### 6. ServiceType (TipoServico)

Catalog of available service categories.

```
ServiceType
├── id (UUID)
├── name
├── category (LEGAL, MEDICAL, NUTRITIONAL, PHYSIOTHERAPY, SOCIAL, EDUCATIONAL, PSYCHOLOGICAL, OTHER)
├── description
├── is_active
├── created_at
└── updated_at
```

### 7. Campaign (Campanha)

A social action event organized by the NGO.

```
Campaign
├── id (UUID)
├── name
├── description
├── campaign_type (SOCIAL_ACTION, EDUCATIONAL, HEALTH, COMMUNITY, OTHER)
├── start_date
├── end_date
├── location_name
├── location_address
├── campus_id (FK → Campus)
├── coordinator_id (FK → Person)
├── status (PLANNED, ACTIVE, COMPLETED, CANCELLED)
├── created_at
├── updated_at
└── created_by (FK → User)
```

### 8. CampaignTeam (EquipeCampanha)

Associates persons (volunteers/professionals) with a campaign.

```
CampaignTeam
├── id (UUID)
├── campaign_id (FK → Campaign)
├── person_id (FK → Person)
├── role_in_campaign (COORDINATOR, PROFESSIONAL, VOLUNTEER, SUPPORT)
├── assigned_at
└── assigned_by (FK → User)
```

---

## Attendance Domain

### 9. Triage (Triagem)

Initial assessment when a person first contacts the organization at an event or campaign.

```
Triage
├── id (UUID)
├── person_id (FK → Person)
├── campaign_id (FK → Campaign, nullable)
├── campus_id (FK → Campus)
├── main_complaint (text)
├── requested_services (M2M → ServiceType)
├── assigned_team (FK → Person, nullable — coordinator)
├── triage_date
├── location
├── triaged_by (FK → User)
├── notes
├── created_at
├── sync_id (UUID — for offline creation)
└── synced_at (nullable)
```

**Lifecycle**: Triages are immutable after creation. Once a triage is recorded, it serves as a snapshot of the person's initial assessment. It cannot be edited or cancelled. If the assessment was incorrect, a new triage can be created. Triages may generate zero or more Attendance records.

### 10. Attendance (Atendimento)

A service record documenting work performed for a person.

```
Attendance
├── id (UUID)
├── person_id (FK → Person)
├── triage_id (FK → Triage, nullable)
├── campaign_id (FK → Campaign, nullable)
├── campus_id (FK → Campus)
├── service_type_id (FK → ServiceType)
├── professional_id (FK → Person)
├── status (SCHEDULED, IN_PROGRESS, COMPLETED, FOLLOW_UP, CANCELLED)
├── attendance_date
├── observations (text)
├── recommendations (text)
├── created_at
├── updated_at
├── created_by (FK → User)
├── sync_id (UUID)
└── synced_at (nullable)
```

### 11. AttendanceTransition (TransicaoAtendimento)

Records state changes in the attendance workflow.

```
AttendanceTransition
├── id (UUID)
├── attendance_id (FK → Attendance)
├── from_status
├── to_status
├── reason (text, nullable)
├── transitioned_by (FK → User)
├── transitioned_at
```

---

## Attendance Lifecycle

```
┌─────────────┐
│   TRIAGE     │  Initial assessment at event/campaign
└──────┬───────┘
       │ Create attendance
       ▼
┌─────────────┐
│  SCHEDULED   │  Service assigned to professional/team
└──────┬───────┘
       │ Begin service
       ▼
┌─────────────┐
│ IN_PROGRESS  │  Service being performed
└──────┬───────┘
       │
       ├──────────────────┐
       │ Complete          │ Needs follow-up
       ▼                  ▼
┌─────────────┐   ┌─────────────┐
│  COMPLETED   │   │  FOLLOW_UP   │
└─────────────┘   └──────┬───────┘
       ▲                  │
       │                  │ Complete follow-up
       │                  │
       └──────────────────┘

       Any state → CANCELLED (with reason)
```

**Rules:**
- A Triage can generate zero or more Attendances
- An Attendance always references a Person and a ServiceType
- Campaign and Triage associations are optional (walk-in attendances are valid)
- Every status change creates an AttendanceTransition record
- COMPLETED and CANCELLED are terminal states (COMPLETED can reopen to FOLLOW_UP)

**MVP (Phase 1) states**: SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED. The FOLLOW_UP state and its transitions (COMPLETED → FOLLOW_UP, FOLLOW_UP → IN_PROGRESS, FOLLOW_UP → COMPLETED) are introduced in Phase 2.

---

## Supporting Domain

### 12. Document (Documento)

File attachments associated with a person or attendance.

```
Document
├── id (UUID)
├── person_id (FK → Person)
├── attendance_id (FK → Attendance, nullable)
├── document_type (ID, PROOF_OF_RESIDENCE, MEDICAL_RECORD, EXAM, CONSENT, PHOTO, OTHER)
├── file_name
├── file_path (storage URL)
├── file_size
├── mime_type
├── uploaded_by (FK → User)
├── uploaded_at
└── description
```

### 13. Consent (Consentimento)

Records consent given by a person for data processing or image usage.

```
Consent
├── id (UUID)
├── person_id (FK → Person)
├── consent_type (DATA_PROCESSING, IMAGE_USAGE, HEALTH_DATA, MINOR_GUARDIAN)
├── consent_version
├── purpose (text)
├── granted_at
├── granted_by_person (FK → Person — may be guardian)
├── signature_data (base64 or file path)
├── is_active
├── revoked_at (nullable)
├── revoked_reason (nullable)
├── campus_id (FK → Campus)
├── sync_id (UUID)
└── synced_at (nullable)
```

### 14. Donation (Doacao)

Records financial or in-kind contributions.

```
Donation
├── id (UUID)
├── donor_person_id (FK → Person, nullable — anonymous allowed)
├── campaign_id (FK → Campaign, nullable)
├── campus_id (FK → Campus)
├── donation_type (FINANCIAL, GOODS, SERVICES)
├── amount (decimal, nullable — for financial)
├── currency (BRL, USD, EUR)
├── item_description (text, nullable — for goods/services)
├── donation_date
├── receipt_number (unique, nullable)
├── receipt_issued_at (nullable)
├── notes
├── registered_by (FK → User)
├── created_at
└── updated_at
```

---

## Audit Domain

### 15. AuditLog (LogAuditoria)

Tracks all data access and modifications for compliance.

```
AuditLog
├── id (UUID)
├── user_id (FK → User, nullable)
├── action_type (CREATE, READ, UPDATE, DELETE, LOGIN, LOGOUT, EXPORT, PERMISSION_CHANGE)
├── entity_type (string — e.g., "Person", "Attendance")
├── entity_id (UUID)
├── module (string — e.g., "attendance", "person", "campaign")
├── description (text)
├── old_values (JSON, nullable)
├── new_values (JSON, nullable)
├── ip_address
├── user_agent
├── campus_id (FK → Campus, nullable)
├── success (boolean)
├── timestamp
```

---

## Entity Relationship Summary

```
Campus ──────────────┐
                     │
Person ──────────────┤
  ├── PersonRole     │
  │   └── VolunteerAgreement
  ├── AssistedProfile│
  ├── User           │
  ├── Document       │
  ├── Consent        │
  └── Donation       │
                     │
Campaign ────────────┤
  └── CampaignTeam   │
                     │
ServiceType          │
                     │
Triage ──────────────┤
  └── requested_services (M2M → ServiceType)
                     │
Attendance ──────────┘
  ├── AttendanceTransition
  └── Document

AuditLog (cross-cutting)
```

---

## Key Design Principles

1. **Person-centric**: Everything connects back to Person. No orphaned records.
2. **UUID everywhere**: Enables offline record creation without ID collision.
3. **Soft-delete pattern**: Records are deactivated, not deleted (except for LGPD erasure).
4. **Audit by default**: Every mutation creates an audit log entry.
5. **Campus-scoped**: Most entities belong to a Campus for data segregation.
6. **Sync-aware**: Entities created offline carry `sync_id` and `synced_at` fields.
7. **Temporal tracking**: `created_at`, `updated_at`, `created_by` on all entities.
