# 04 - Domain Model

## Overview

The domain model is centered on a **unified Person entity** that can hold multiple roles (beneficiary, volunteer, professional, coordinator, administrator). All operational activities — triage, attendance, campaigns, donations — relate back to Person records.

---

## Core Domain Entities

### 1. Person (Pessoa)

The central entity. Every human in the system — whether beneficiary, volunteer, or professional — is first and foremost a Person.

```
Person
├── id (UUID)
├── full_name
├── birth_date
├── document_type (CPF, SSN, EU_ID, OTHER)
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
- Document type + number allows international support
- Address is a separate struct/table for normalization
- Campus association enables data segregation

### 2. PersonRole (PapelPessoa)

Associates a Person with one or more business roles. A person can be simultaneously a beneficiary and a volunteer.

```
PersonRole
├── id (UUID)
├── person_id (FK → Person)
├── role_type (VOLUNTEER, ASSISTED, PROFESSIONAL, COORDINATOR, ADMIN)
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
├── email (unique, used for login)
├── password_hash
├── access_profile (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER)
├── is_active
├── last_login
├── created_at
└── updated_at
```

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
├── is_active
├── created_at
└── updated_at
```

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
