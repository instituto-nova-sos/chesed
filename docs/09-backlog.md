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

**Target**: Phase 0 / Sprint 0

### Stories

**S01.1 - Go project skeleton**
- As a developer, I need a Go project with standard layout so I can begin implementing features.
- Acceptance criteria:
  - `cmd/server/main.go` with HTTP server startup
  - `internal/` package structure (config, domain, handler, service, repository, middleware)
  - `go.mod` with initial dependencies (chi, pgx, coreos/go-oidc, golang-migrate)
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

**S01.6 - Service type seed data**
- As a developer, I need predefined service types loaded into the database.
- Acceptance criteria:
  - Migration seeds: LEGAL, MEDICAL, NUTRITIONAL, PHYSIOTHERAPY, SOCIAL, EDUCATIONAL, PSYCHOLOGICAL, OTHER
  - `GET /api/v1/service-types` returns the list (available to all authenticated users)
  - Service types are read-only in Phase 1

---

## E02: Authentication and Authorization

**Target**: Sprint 1

### Stories

**S02.1 - User registration (admin-initiated)**
- As an admin, I can create system users associated with registered persons so they can access the system.
- Acceptance criteria:
  - `POST /api/v1/users` creates a user in Keycloak via Admin API and a local `app_user` record
  - Requires admin role
  - Sets access profile as Keycloak realm role (admin, coordinator, professional, secretary, volunteer)
  - Sets `campus_id` and `person_id` as Keycloak user attributes
  - Sets required action: `UPDATE_PASSWORD` (user sets password on first login)
  - Sends password setup email if SMTP is configured in Keycloak

**S02.2 - Keycloak OIDC integration**
- As a user, I am redirected to the Keycloak login page to authenticate.
- Acceptance criteria:
  - React app uses `keycloak-js` adapter to redirect to Keycloak
  - Authorization Code Flow with PKCE
  - Keycloak handles credential validation, account lockout, and brute-force protection
  - On successful auth, user is redirected back with OIDC tokens
  - Successful login is logged by Keycloak event listener
  - On first login, Go API auto-creates local `app_user` record from `sub` claim

**S02.3 - Token refresh (Keycloak-managed)**
- As a user, my session stays active without re-entering credentials.
- Acceptance criteria:
  - `keycloak-js` adapter handles silent token refresh automatically
  - No custom refresh endpoint in the Go API
  - Expired refresh token redirects to Keycloak login page
  - For field workers, `offline_access` scope provides 14-day offline tokens

**S02.4 - Password recovery (Keycloak-managed)**
- As a user, I can reset my password if I forget it.
- Acceptance criteria:
  - Keycloak built-in "Forgot Password" flow handles password reset
  - No custom password reset endpoints in the Go API
  - Requires SMTP configuration in Keycloak
  - Keycloak sends reset email with secure link

**S02.5 - RBAC middleware**
- As the system, I enforce role-based access on every request.
- Acceptance criteria:
  - Go middleware validates Keycloak token via JWKS endpoint (`coreos/go-oidc`)
  - Extracts `realm_access.roles` from Keycloak token claims
  - Extracts `campus_id` from custom token claim
  - Each handler declares required role(s)
  - Unauthorized access returns 403
  - Unauthorized access creates audit log entry

**S02.6 - Campus data isolation**
- As the system, I ensure users only see data from their campus.
- Acceptance criteria:
  - All list/detail queries include `campus_id` filter from Keycloak token claims
  - Admins can optionally query across campuses
  - Cross-campus access attempts are logged

**S02.7 - React OIDC integration**
- As a user, I am redirected to the Keycloak login page (themed with NGO branding).
- Acceptance criteria:
  - `keycloak-js` adapter initialized on app startup
  - Login redirects to Keycloak (no custom login form in React)
  - OIDC callback route handles token receipt
  - Logout calls Keycloak end-session endpoint
  - Token stored and managed by `keycloak-js` adapter

**S02.8 - React auth context**
- As the frontend, I manage authentication state across the app.
- Acceptance criteria:
  - Auth context wraps `keycloak-js` adapter
  - Provides current user (email, role, campus_id) from token claims
  - Auto-attaches access token to API calls via interceptor
  - Silent token refresh handled by `keycloak-js`
  - Redirect to Keycloak login on 401
  - Protected route wrapper component

**S02.9 - Keycloak realm configuration as code**
- As the team, we have reproducible Keycloak configuration.
- Acceptance criteria:
  - Realm configuration exported to `keycloak/realm-export.json`
  - Includes realm roles, client configs, protocol mappers, password policy, brute-force settings
  - Import on Docker Compose startup via `--import-realm`
  - Changes to realm config reviewed in pull requests

**S02.10 - MFA for admin accounts**
- As an admin, I am required to use TOTP two-factor authentication.
- Acceptance criteria:
  - Keycloak conditional authentication flow requires TOTP for ADMIN role
  - Zero application code needed
  - TOTP enrollment prompted on first admin login

---

## E03: Person Management

**Target**: Sprint 2

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

**Target**: Sprint 2-3

> Sprint 3-4 metadata convention: each story below carries `status`, `depends_on`,
> `covers_requirements`, `parallel_with`, `size`, and `offline` fields. `status`
> is sourced from `docs/09-backlog.md` (single source of truth — never hand-edit
> `tasks/STATUS.md`). Run `make validate-backlog` to check field integrity and
> `make status` to regenerate the board.

### Stories

**S04.1 - Create triage**
- status: done
- depends_on: [S02.5, S03.1]
- covers_requirements: [RF-22, RF-23, RF-24, RF-26]
- parallel_with: [S04.2, S06.1]
- size: M
- offline: Triage form persists to IndexedDB with a client-generated UUID; the record is pushed via the sync queue when connectivity returns (see S05.3).
- As a volunteer, I can record an initial triage during an event.
- Acceptance criteria:
  - **Given** an authenticated volunteer with a valid `campus_id` claim **when** they `POST /api/v1/triages` with a valid `person_id`, `main_complaint`, and at least one requested service **then** the API returns `201` with the created triage, and `triage_date`, `triaged_by`, and `campus_id` are auto-populated from the token.
  - **Given** a triage payload referencing a `person_id` from another campus **when** the request is processed **then** the API returns `403` and writes a cross-campus access audit entry.
  - **Given** a triage payload missing `main_complaint` **when** it is submitted **then** the API returns `400` with a field-level validation error and creates no triage row.
  - **Given** a successful triage creation **when** the transaction commits **then** an `audit_log` entry is written capturing the new triage and the acting user.
  - **Given** an offline-created triage carrying a client `sync_id` **when** it is later pushed via `/api/v1/sync/push` **then** the server persists it idempotently (a duplicate `sync_id` is a no-op, not a second row).

**S04.2 - Create attendance from triage**
- status: done
- depends_on: [S04.1, S03.1]
- covers_requirements: [RF-27, RF-28, RF-33]
- parallel_with: [S04.1, S06.1]
- size: M
- offline: Attendance can be created offline against a triage already present in IndexedDB; client UUID is assigned and the record is queued for sync.
- As a coordinator, I can create an attendance record from a triage.
- Acceptance criteria:
  - **Given** an existing triage in the user's campus **when** the coordinator `POST /api/v1/attendances` with `person_id`, `triage_id`, `service_type_id`, and `professional_id` **then** the API returns `201` with status `SCHEDULED`.
  - **Given** a successful attendance creation **when** the transaction commits **then** an initial `attendance_transition` row (`NULL → SCHEDULED`) and an `audit_log` entry are created.
  - **Given** an attendance payload referencing a `triage_id` from another campus **when** the request is processed **then** the API returns `403` and logs the cross-campus attempt.
  - **Given** an attendance payload with an unknown `service_type_id` **when** it is submitted **then** the API returns `400` and creates no attendance row.

**S04.3 - Attendance workflow transitions**
- status: done
- depends_on: [S04.2]
- covers_requirements: [RF-32, RF-33, RF-34]
- parallel_with: [S04.4, S04.5]
- size: M
- offline: Status transitions made offline are queued; the server re-validates the transition against current state on push and rejects illegal transitions per record.
- As a professional, I can move an attendance through workflow states.
- Acceptance criteria:
  - **Given** an attendance in `SCHEDULED` **when** a professional `PATCH /api/v1/attendances/:id/transition` to `IN_PROGRESS` **then** the API returns `200`, the status updates, and an `attendance_transition` row records `from`, `to`, actor, and timestamp.
  - **Given** an attendance in `SCHEDULED` **when** a transition to `COMPLETED` is requested directly **then** the API returns `409` (illegal transition) and the status is unchanged.
  - **Given** any successful transition **when** it commits **then** an `audit_log` entry is created.
  - **Given** an attendance in another campus **when** a transition is attempted **then** the API returns `403` and logs the attempt.

**S04.4 - Record attendance details**
- status: done
- depends_on: [S04.2]
- covers_requirements: [RF-29]
- parallel_with: [S04.3, S04.5]
- size: S
- offline: Observations/recommendations edited offline are queued and pushed; last-write-wins on the server per the Phase 1 sync strategy.
- As a professional, I can add observations and recommendations to an attendance.
- Acceptance criteria:
  - **Given** the assigned professional **when** they `PATCH /api/v1/attendances/:id` with `observations` and `recommendations` **then** the API returns `200` and the fields are persisted.
  - **Given** a user who is neither the assigned professional nor a coordinator **when** they attempt the same `PATCH` **then** the API returns `403` and no fields change.
  - **Given** a successful detail update **when** it commits **then** an `audit_log` entry captures old and new values.

**S04.5 - List attendances**
- status: done
- depends_on: [S04.2]
- covers_requirements: [RF-31]
- parallel_with: [S04.3, S04.4]
- size: S
- offline: List reads served from the IndexedDB cache when offline; the badge indicates cached/stale data (see S05.5).
- As a professional, I can see my assigned attendances.
- Acceptance criteria:
  - **Given** an authenticated professional **when** they `GET /api/v1/attendances?professional_id=me&status=SCHEDULED,IN_PROGRESS` **then** only their campus's matching attendances are returned, paginated and sorted by date.
  - **Given** more results than one page **when** the next page is requested **then** pagination metadata (page size, has-more) is returned and no rows are duplicated across pages.
  - **Given** a `status` filter value outside the allowed enum **when** the request is made **then** the API returns `400`.
  - **Given** a user from campus A **when** they list attendances **then** no campus B attendance ever appears in the response.

**S04.6 - React triage form**
- status: done
- depends_on: [S04.1, S03.8]
- covers_requirements: [RF-22, RF-23, RF-24, RNF-19]
- parallel_with: [S04.7, S04.8]
- size: M
- offline: Form submits to IndexedDB when offline (client UUID + queue entry); an offline badge is shown and submission succeeds without a network round-trip.
- As a volunteer, I can fill a triage form on a phone, online or offline.
- Acceptance criteria:
  - **Given** the triage form **when** the user searches for a person and selects one **then** the selected person is bound to the form and their name is displayed.
  - **Given** a completed form **when** the user submits while online **then** `POST /api/v1/triages` is called and a success state is shown on `201`.
  - **Given** a completed form **when** the user submits while offline **then** the record is written to IndexedDB with a client UUID and an enqueued sync entry, and the UI confirms the local save.
  - **Given** the form rendered at 320px width **when** the user interacts **then** all fields and the service-type multi-select remain usable without horizontal scroll.

**S04.7 - React attendance form**
- status: done
- depends_on: [S04.2, S04.6]
- covers_requirements: [RF-27, RF-29, RNF-19]
- parallel_with: [S04.6, S04.8]
- size: M
- offline: Attendance edits and status changes are persisted to IndexedDB and queued when offline; the form does not block on connectivity.
- As a professional, I can record an attendance and move its status on a phone.
- Acceptance criteria:
  - **Given** an attendance form **when** the professional sets service type, observations, and recommendations and submits online **then** `PATCH /api/v1/attendances/:id` is called and the saved state is reflected.
  - **Given** status transition buttons **when** the professional taps an allowed transition **then** the transition endpoint is called and the new status badge is shown.
  - **Given** an illegal transition response (`409`) from the API **when** it is received **then** the UI shows a clear error and leaves the status unchanged.
  - **Given** the form used offline **when** the professional saves **then** changes persist locally and queue for sync.

**S04.8 - React attendance list**
- status: done
- depends_on: [S04.5]
- covers_requirements: [RF-31, RNF-13]
- parallel_with: [S04.6, S04.7]
- size: S
- offline: Renders from the IndexedDB cache when offline with a stale-data indicator; filters apply to cached rows.
- As a professional, I can browse and filter my attendances on a phone.
- Acceptance criteria:
  - **Given** the attendance list **when** the user applies a status, date, or professional filter **then** the visible rows update to match the filter.
  - **Given** color-coded status badges **when** the list renders **then** each attendance shows a badge whose color maps to its status.
  - **Given** a row **when** the user taps it **then** the attendance detail/edit view opens.
  - **Given** offline mode **when** the list is opened **then** cached attendances render and a stale/offline indicator is visible.

---

## E05: Offline Sync

**Target**: Sprint 3-4

> This epic is the current Phase 1 critical path. The backend push/pull contract
> is implemented and integration-tested (commit #28); the remaining work is
> frontend-heavy: a Dexie v2 schema, the `useOnlineSync` drainer hook, pull-merge,
> and conflict surfacing. See `docs/08-roadmap.md` → "Parallelization Model".

### Stories

**S05.1 - IndexedDB schema and stores (Dexie v2)**
- status: in_progress
- depends_on: [S03.8]
- covers_requirements: [RF-47, RNF-04, RNF-18]
- parallel_with: [S05.5]
- size: M
- offline: This story *is* the offline substrate — it defines the local stores that hold unsynced records and the queue that drives the drainer.
- As the frontend, I have a versioned local database for offline records and a durable sync queue.
- Acceptance criteria:
  - **Given** the app starts for the first time **when** Dexie initializes **then** a database is created with tables `persons`, `triages`, `attendances`, and `syncQueue`, each keyed by client UUID.
  - **Given** an existing v1 database **when** the app upgrades to the v2 schema **then** the Dexie version migration runs without data loss and existing offline records remain readable.
  - **Given** a browser that supports encryption at rest **when** the database is opened **then** local data is stored encrypted; **given** a browser without support **then** the app degrades gracefully and still functions.
  - **Given** a record written to a store **when** the page is reloaded **then** the record is still present (durability across sessions).

**S05.2 - Offline record creation**
- status: done
- depends_on: [S05.1, S03.8]
- covers_requirements: [RF-46, RF-47]
- parallel_with: [S05.5]
- size: M
- offline: This is the write path for all offline-capable forms; it must never depend on the network being present.
- As a field user, I can create persons, triages, and attendances with no connectivity.
- Acceptance criteria:
  - **Given** the device is offline **when** the user submits a person, triage, or attendance form **then** the record is written to the matching IndexedDB store with a client-generated UUID and a corresponding `syncQueue` entry.
  - **Given** an offline-created record **when** the user navigates to the relevant list **then** the local record appears immediately (read-your-writes from cache).
  - **Given** the device is offline **when** any record is created **then** a visible offline-mode indicator is shown.
  - **Given** an offline-created record **when** connectivity returns **then** the queued entry is eligible for the drainer (S05.3) without further user action.

**S05.3 - Sync queue and push (drainer)**
- status: ready
- depends_on: [S05.1, S05.2]
- covers_requirements: [RF-48, RF-49, RNF-09]
- parallel_with: [S05.4]
- size: L
- offline: This is the drain side — `useOnlineSync` flushes the queue when the browser reports online; backend `/sync/push` is already implemented and idempotent per `sync_id`.
- As the system, I flush queued offline records to the server when connectivity returns.
- Acceptance criteria:
  - **Given** queued records and a transition from offline to online **when** `useOnlineSync` runs **then** it batches the queue and calls `POST /api/v1/sync/push` (batch cap 50 per request).
  - **Given** a push response with per-record results **when** it is processed **then** succeeded records are removed from `syncQueue` and their local rows are marked synced; failed records remain queued.
  - **Given** a transient failure on a record **when** the next drain runs **then** the record is retried with backoff and is not lost.
  - **Given** the same `sync_id` is pushed twice (e.g., a retry after a dropped response) **when** the server processes it **then** no duplicate row is created (idempotency holds end-to-end).
  - **Given** a batch-level error (oversize or missing campus) **when** the server responds **then** the client surfaces the error and does not silently drop the batch.

**S05.4 - Pull sync (pull-merge)**
- status: ready
- depends_on: [S05.3]
- covers_requirements: [RF-48, RF-49, RNF-09]
- parallel_with: [S05.3]
- size: L
- offline: Pull-merge reconciles server changes into the local cache; conflicts are resolved last-write-wins in Phase 1, with the conflicting field surfaced to the user.
- As the system, I pull server-side changes since the last sync and merge them locally.
- Acceptance criteria:
  - **Given** a stored last-sync cursor **when** the client calls `GET /api/v1/sync/pull?since=<cursor>` **then** it receives only records modified since the cursor plus a `next_since` cursor and `has_more` flag.
  - **Given** `has_more` is true **when** the client continues paging **then** it advances the cursor until the server reports no more pages, then persists the final cursor locally.
  - **Given** a pulled record that does not exist locally **when** merge runs **then** it is inserted into the matching store.
  - **Given** a pulled record that conflicts with an unsynced local edit **when** merge runs **then** last-write-wins is applied and the conflict is recorded so the UI can surface it.
  - **Given** an interrupted pull (tab closed mid-page) **when** the app restarts **then** the merge resumes from the persisted cursor without data corruption.

**S05.5 - Online/offline status indicator**
- status: done
- depends_on: [S05.1]
- covers_requirements: [RF-48, RNF-18]
- parallel_with: [S05.1, S05.2]
- size: S
- offline: Purely a presentation of sync state (`useOfflineStatus` already exists); reflects online/offline/syncing and the pending queue depth.
- As a user, I can see whether I am online, offline, or syncing, and how many records are pending.
- Acceptance criteria:
  - **Given** the browser reports offline **when** the badge renders **then** it shows the offline state.
  - **Given** queued records **when** the badge renders **then** it shows the pending sync count, updating as the queue drains.
  - **Given** an active drain **when** it is running **then** the badge shows a syncing state.
  - **Given** the user taps "Sync Now" **when** connectivity is available **then** a drain is triggered immediately; **when** offline **then** the action is disabled or clearly indicates it will run once online.

---

## E06: Basic Reports

**Target**: Sprint 4

> Phase 1 reports are intentionally minimal: attendance counts by period, service
> type, and status, plus CSV export. Charts (Recharts) and campaign/team reports
> are Phase 2 (RF-42, RF-45). Reports are an online-only read surface in Phase 1.

### Stories

**S06.1 - Attendance count by period**
- status: done
- depends_on: [S04.2, S02.5]
- covers_requirements: [RF-40, RF-41]
- parallel_with: [S06.2]
- size: M
- offline: Online-only. Reports query live server aggregates; when offline the report page shows a clear "requires connection" state rather than stale numbers.
- As a coordinator, I can see attendance totals for a date range.
- Acceptance criteria:
  - **Given** an authenticated coordinator **when** they `GET /api/v1/reports/attendances?start=YYYY-MM-DD&end=YYYY-MM-DD` **then** the API returns total count, count by service type, and count by status, all scoped to their campus.
  - **Given** a `start` later than `end` (or a malformed date) **when** the request is made **then** the API returns `400` with a validation error.
  - **Given** a coordinator in campus A **when** the report runs **then** no campus B attendance is counted.
  - **Given** a range with no attendances **when** the report runs **then** the API returns zeroed counts (not an error).

**S06.2 - CSV export**
- status: done
- depends_on: [S06.1]
- covers_requirements: [RF-44]
- parallel_with: [S06.1]
- size: S
- offline: Online-only. Export streams a server-generated file; unavailable offline.
- As a coordinator, I can export attendance details as a CSV file.
- Acceptance criteria:
  - **Given** a coordinator **when** they `GET /api/v1/reports/attendances/export?start=...&end=...&format=csv` **then** the API responds with `text/csv` and a CSV body of attendance details for the range.
  - **Given** a user below coordinator access profile **when** they request the export **then** the API returns `403` and logs the attempt.
  - **Given** the exported CSV **when** it is opened **then** it contains a header row and one row per attendance, campus-scoped.
  - **Given** an invalid date range **when** the export is requested **then** the API returns `400` and produces no file.

**S06.3 - React report page**
- status: done
- depends_on: [S06.1, S06.2]
- covers_requirements: [RF-40, RF-44, RNF-13]
- parallel_with: []
- size: S
- offline: Online-only. When offline, the page disables the run/export actions and shows a "connect to view reports" message.
- As a coordinator, I can pick a date range and view/export the report on a phone.
- Acceptance criteria:
  - **Given** the report page **when** the coordinator selects a start and end date and runs the report **then** a summary table of counts is shown.
  - **Given** report results **when** the user clicks "Export CSV" **then** the CSV download is triggered from the export endpoint.
  - **Given** the page rendered at 320px width **when** it is viewed **then** the date pickers, table, and export button remain usable without horizontal scroll.
  - **Given** the device is offline **when** the page is opened **then** the run and export actions are disabled and a clear offline message is shown.

---

## E07: Campaign and Event Management (Phase 2)

**Phase**: 2 | **Priority**: P1 | **Prerequisite**: Phase 1 complete

> Detailed acceptance criteria deferred to phase kickoff (phase-boundary rule).

**S07.1** - Create/edit/list campaigns
**S07.2** - Assign team members to campaigns
**S07.3** - Link triage/attendance to campaigns
**S07.4** - Campaign dashboard (metrics per campaign)
**S07.5** - React campaign pages

---

## E08: Document and Consent Management (Phase 2)

**Phase**: 2 | **Priority**: P1 | **Prerequisite**: Phase 1 complete

> Detailed acceptance criteria deferred to phase kickoff (phase-boundary rule).

**S08.1** - Object storage integration (S3/MinIO)
**S08.2** - Upload documents to person or attendance
**S08.3** - Create consent with digital signature capture
**S08.4** - Revoke consent
**S08.5** - React document upload component
**S08.6** - React consent form with signature pad

---

## E09: Donation Tracking (Phase 2)

**Phase**: 2 | **Priority**: P1 | **Prerequisite**: Phase 1 complete

> Detailed acceptance criteria deferred to phase kickoff (phase-boundary rule).

**S09.1** - Create/edit/list donations
**S09.2** - Link donations to campaigns
**S09.3** - React donation form and list

---

## E10: Advanced Reports and Dashboards (Phase 2)

**Phase**: 2 | **Priority**: P1 | **Prerequisite**: Phase 1 complete

> Detailed acceptance criteria deferred to phase kickoff (phase-boundary rule).

**S10.1** - Reports by service type, team, campaign
**S10.2** - Statistical charts (Recharts)
**S10.3** - Dashboard with key metrics
**S10.4** - Report filter UI

---

## E11: Multi-Region and Compliance (Phase 3)

**Phase**: 3 | **Priority**: P2 | **Prerequisite**: Phase 2 complete

> Detailed acceptance criteria deferred to phase kickoff (phase-boundary rule).

**S11.1** - Multi-campus data isolation with PostgreSQL RLS
**S11.2** - International document type support
**S11.3** - Consent revocation with data anonymization
**S11.4** - LGPD compliance reporting
**S11.5** - Donation receipt PDF generation

---

## E12: External Integrations (Phase 3)

**Phase**: 3 | **Priority**: P2 | **Prerequisite**: Phase 2 complete

> Detailed acceptance criteria deferred to phase kickoff (phase-boundary rule).

**S12.1** - WordPress public API (campaign listings, volunteer signup)
**S12.2** - Advanced sync conflict resolution UI
**S12.3** - Automated backup and disaster recovery
**S12.4** - Email notifications (password recovery, event reminders)
