# 09 - Backlog

## Epic Overview

| Epic | Phase | Priority | Description |
|------|-------|----------|-------------|
| E01 | 0 | P0 | Project Setup and Infrastructure |
| E02 | 1 | P0 | Authentication and Authorization |
| E03 | 1 | P0 | Person Management |
| E04 | 1 | P0 | Triage and Attendance |
| E05 | 1 | P0 | Offline Sync |
| E06 | 1 | P0 | Basic Reports |
| E07 | 2 | P1 | Campaign and Event Management |
| E08 | 2 | P1 | Document and Consent Management |
| E09 | 2 | P1 | Donation Tracking |
| E10 | 2 | P1 | Advanced Reports and Dashboards |
| E11 | 3 | P2 | Multi-Region and Compliance |
| E12 | 3 | P2 | External Integrations |

---

## E01: Project Setup and Infrastructure

### Stories

**S01.1 - Go project skeleton**
- As a developer, I need a Go project with standard layout so I can begin implementing features.
- Acceptance criteria:
  - `cmd/server/main.go` with HTTP server startup
  - `internal/` package structure (config, domain, handler, service, repository, middleware)
  - `go.mod` with initial dependencies (chi, pgx, golang-jwt, golang-migrate)
  - Health check endpoint (`GET /api/v1/health`)
  - Structured logging with slog
  - Configuration loading from environment variables

**S01.2 - React project skeleton**
- As a developer, I need a React + TypeScript project with PWA capabilities.
- Acceptance criteria:
  - Vite project with React 18+ and TypeScript
  - Tailwind CSS configured
  - PWA manifest and service worker shell
  - ESLint + Prettier configured
  - Base layout component (responsive sidebar + header)
  - Routing setup with React Router

**S01.3 - Docker Compose environment**
- As a developer, I need a single command to start all services locally.
- Acceptance criteria:
  - `docker compose up` starts Go API, PostgreSQL, and React dev server
  - Hot reload for both Go and React
  - Environment variables via `.env` file
  - PostgreSQL data persisted in named volume

**S01.4 - Database migration framework**
- As a developer, I need SQL-based migrations so the database schema is versioned.
- Acceptance criteria:
  - golang-migrate configured
  - CLI command to run migrations (`make migrate-up`, `make migrate-down`)
  - Initial migration creates `campus` and `audit_log` tables
  - Migration files in `backend/migrations/`

**S01.5 - CI/CD pipeline**
- As a developer, I need automated testing on every push.
- Acceptance criteria:
  - GitHub Actions workflow runs Go tests and React build
  - PostgreSQL service container for integration tests
  - Build fails on lint errors or test failures

### Technical Enablers

**TE01.1 - Makefile with common commands**
- `make run`, `make test`, `make migrate-up`, `make migrate-down`, `make seed`, `make lint`

**TE01.2 - Environment configuration**
- `.env.example` with all required variables documented
- `internal/config` package loads from env with validation

---

## E02: Authentication and Authorization

### Stories

**S02.1 - User registration (admin-initiated)**
- As an admin, I can create system users associated with registered persons so they can access the system.
- Acceptance criteria:
  - `POST /api/v1/users` creates a user linked to a person_id
  - Requires admin role
  - Sets access profile (admin, coordinator, professional, secretary, volunteer)
  - Assigns campus
  - Sends no email (MVP); credentials provided by admin

**S02.2 - Login with email and password**
- As a user, I can log in with my email and password to access the system.
- Acceptance criteria:
  - `POST /api/v1/auth/login` accepts email + password
  - Returns JWT access token (15min TTL) + refresh token (7 days)
  - Invalid credentials return 401 with generic error
  - Successful login creates audit log entry
  - Failed login creates audit log entry

**S02.3 - Token refresh**
- As a user, my session stays active without re-entering credentials.
- Acceptance criteria:
  - `POST /api/v1/auth/refresh` accepts refresh token
  - Returns new access token
  - Expired refresh token returns 401

**S02.4 - Password recovery**
- As a user, I can reset my password if I forget it.
- Acceptance criteria:
  - `POST /api/v1/auth/forgot-password` accepts email
  - Generates secure reset token (1 hour TTL)
  - `POST /api/v1/auth/reset-password` accepts token + new password
  - Always returns 200 (no email enumeration)

**S02.5 - RBAC middleware**
- As the system, I enforce role-based access on every request.
- Acceptance criteria:
  - Middleware extracts user, role, campus from JWT
  - Each handler declares required role(s)
  - Unauthorized access returns 403
  - Unauthorized access creates audit log entry

**S02.6 - Campus data isolation**
- As the system, I ensure users only see data from their campus.
- Acceptance criteria:
  - All list/detail queries include `campus_id` filter from JWT
  - Admins can optionally query across campuses
  - Cross-campus access attempts are logged

**S02.7 - React login page**
- As a user, I see a clean login form on my mobile phone.
- Acceptance criteria:
  - Email and password fields with validation
  - Error message on failed login
  - Redirect to dashboard on success
  - Token stored securely (httpOnly cookie preferred, fallback to secure storage)
  - "Forgot password" link

**S02.8 - React auth context**
- As the frontend, I manage authentication state across the app.
- Acceptance criteria:
  - Auth context provides current user, role, campus
  - Automatic token refresh before expiry
  - Redirect to login on 401
  - Protected route wrapper component

---

## E03: Person Management

### Stories

**S03.1 - Create person**
- As a secretary or volunteer, I can register a new person.
- Acceptance criteria:
  - `POST /api/v1/persons` with name, document_type, document_number, birth_date, phone, address, campus_id
  - UUID generated server-side (or accepted from client for offline-created records)
  - Returns created person with ID
  - Audit log entry created

**S03.2 - Duplicate detection**
- As the system, I prevent duplicate person registrations.
- Acceptance criteria:
  - Check document_number uniqueness before creation
  - Fuzzy name + birth_date match suggests potential duplicates
  - `GET /api/v1/persons/check-duplicate?document_number=X` returns matches
  - Client shows warning before allowing duplicate save

**S03.3 - Search persons**
- As a user, I can quickly find a person by name or document.
- Acceptance criteria:
  - `GET /api/v1/persons?q=search_term` searches name and document_number
  - Full-text search on name (PostgreSQL tsvector)
  - Paginated results (20 per page)
  - Results scoped to user's campus

**S03.4 - Update person**
- As a secretary, I can update a person's information.
- Acceptance criteria:
  - `PUT /api/v1/persons/:id` updates allowed fields
  - Audit log captures old and new values
  - Cannot change document_number without admin role

**S03.5 - Person detail with history**
- As a professional, I can view a person's complete profile and service history.
- Acceptance criteria:
  - `GET /api/v1/persons/:id` returns person data
  - `GET /api/v1/persons/:id/history` returns triages and attendances in chronological order
  - Includes role information

**S03.6 - Person role assignment**
- As a coordinator, I can assign roles (volunteer, assisted, professional) to a person.
- Acceptance criteria:
  - `POST /api/v1/persons/:id/roles` adds a role
  - `PATCH /api/v1/persons/:id/roles/:role_id` activates/deactivates
  - Audit log entry for role changes

**S03.7 - React person list page**
- Acceptance criteria:
  - Search bar with real-time filtering
  - Person cards showing name, document, phone, roles
  - "New Person" button
  - Infinite scroll or pagination
  - Works on 320px-wide screens

**S03.8 - React person form**
- Acceptance criteria:
  - Form with all person fields
  - Duplicate warning before save
  - Validation (required fields, CPF format)
  - Works offline (saves to IndexedDB)

**S03.9 - React person detail page**
- Acceptance criteria:
  - Profile card with person info
  - Attendance history timeline
  - Role badges
  - Edit button (authorized roles only)

---

## E04: Triage and Attendance

### Stories

**S04.1 - Create triage**
- As a volunteer, I can record an initial triage during an event.
- Acceptance criteria:
  - `POST /api/v1/triages` with person_id, main_complaint, requested_services, campaign_id (optional)
  - Auto-sets triage_date, triaged_by, campus_id
  - Returns created triage

**S04.2 - Create attendance from triage**
- As a coordinator, I can create an attendance record from a triage.
- Acceptance criteria:
  - `POST /api/v1/attendances` with person_id, triage_id, service_type_id, professional_id
  - Initial status: SCHEDULED
  - Transition record created

**S04.3 - Attendance workflow transitions**
- As a professional, I can move an attendance through workflow states.
- Acceptance criteria:
  - `PATCH /api/v1/attendances/:id/transition` with new_status
  - Validates transition is allowed (SCHEDULED→IN_PROGRESS→COMPLETED)
  - Creates AttendanceTransition record
  - Audit log entry

**S04.4 - Record attendance details**
- As a professional, I can add observations and recommendations to an attendance.
- Acceptance criteria:
  - `PATCH /api/v1/attendances/:id` updates observations, recommendations
  - Only assigned professional or coordinator can edit

**S04.5 - List attendances**
- As a professional, I can see my assigned attendances.
- Acceptance criteria:
  - `GET /api/v1/attendances?professional_id=me&status=SCHEDULED,IN_PROGRESS`
  - Paginated, sorted by date
  - Filter by status, service_type, date range

**S04.6 - React triage form**
- Acceptance criteria:
  - Person search/select (or inline creation)
  - Complaint text field
  - Service type multi-select
  - Campaign auto-selected if applicable
  - Offline-capable

**S04.7 - React attendance form**
- Acceptance criteria:
  - Service type, professional assignment
  - Observations and recommendations text areas
  - Status transition buttons
  - Offline-capable

**S04.8 - React attendance list**
- Acceptance criteria:
  - Filter by status, date, professional
  - Color-coded status badges
  - Tap to open detail/edit

---

## E05: Offline Sync

### Stories

**S05.1 - IndexedDB schema and stores**
- Acceptance criteria:
  - Dexie.js database with tables: persons, triages, attendances, syncQueue
  - Schema versioning for future migrations
  - Encryption at rest (if browser supports it)

**S05.2 - Offline record creation**
- Acceptance criteria:
  - Person, triage, and attendance forms save to IndexedDB when offline
  - Records get client-generated UUIDs
  - Visual indicator shows offline mode

**S05.3 - Sync queue and push**
- Acceptance criteria:
  - Offline records queued with timestamps
  - `POST /api/v1/sync/push` accepts batch of records
  - Server validates and persists; returns success/failure per record
  - Queue retries failed records

**S05.4 - Pull sync**
- Acceptance criteria:
  - `GET /api/v1/sync/pull?since=timestamp` returns records modified since last sync
  - Client merges into IndexedDB
  - Last sync timestamp persisted locally

**S05.5 - Online/offline status indicator**
- Acceptance criteria:
  - Visual badge shows online/offline/syncing status
  - Pending sync count visible
  - Manual "Sync Now" button

---

## E06: Basic Reports

### Stories

**S06.1 - Attendance count by period**
- Acceptance criteria:
  - `GET /api/v1/reports/attendances?start=YYYY-MM-DD&end=YYYY-MM-DD`
  - Returns total count, count by service type, count by status
  - Campus-scoped

**S06.2 - CSV export**
- Acceptance criteria:
  - `GET /api/v1/reports/attendances/export?start=...&end=...&format=csv`
  - Returns CSV file with attendance details
  - Respects RBAC (coordinator+ only)

**S06.3 - React report page**
- Acceptance criteria:
  - Date range picker
  - Summary table with counts
  - "Export CSV" button
  - Mobile-friendly layout

---

## E07: Campaign and Event Management (Phase 2)

**S07.1** - Create/edit/list campaigns
**S07.2** - Assign team members to campaigns
**S07.3** - Link triage/attendance to campaigns
**S07.4** - Campaign dashboard (metrics per campaign)
**S07.5** - React campaign pages

---

## E08: Document and Consent Management (Phase 2)

**S08.1** - Object storage integration (S3/MinIO)
**S08.2** - Upload documents to person or attendance
**S08.3** - Create consent with digital signature capture
**S08.4** - Revoke consent
**S08.5** - React document upload component
**S08.6** - React consent form with signature pad

---

## E09: Donation Tracking (Phase 2)

**S09.1** - Create/edit/list donations
**S09.2** - Link donations to campaigns
**S09.3** - React donation form and list

---

## E10: Advanced Reports and Dashboards (Phase 2)

**S10.1** - Reports by service type, team, campaign
**S10.2** - Statistical charts (Recharts)
**S10.3** - Dashboard with key metrics
**S10.4** - Report filter UI

---

## E11: Multi-Region and Compliance (Phase 3)

**S11.1** - Multi-campus data isolation with PostgreSQL RLS
**S11.2** - International document type support
**S11.3** - Consent revocation with data anonymization
**S11.4** - LGPD compliance reporting
**S11.5** - Donation receipt PDF generation

---

## E12: External Integrations (Phase 3)

**S12.1** - WordPress public API (campaign listings, volunteer signup)
**S12.2** - Advanced sync conflict resolution UI
**S12.3** - Automated backup and disaster recovery
**S12.4** - Email notifications (password recovery, event reminders)
