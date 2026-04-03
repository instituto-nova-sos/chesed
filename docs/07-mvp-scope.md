# 07 - MVP Scope

## MVP Definition

The Minimum Viable Product delivers the core loop: **register a person → record triage → record attendance → view history → generate basic reports**. It must work on mobile devices and support offline data entry.

---

## MVP Features (Phase 1)

### Must Have

| Feature | Requirements Covered | Description |
|---------|---------------------|-------------|
| **Person registration** | RF-01, RF-02, RF-03, RF-04 | Create and update person records with basic data, CPF deduplication |
| **Person search** | RF-03 | Search by name, CPF, or document number with fuzzy matching |
| **Triage form** | RF-22, RF-23, RF-24, RF-26 | Record initial assessment with complaint, requested services, date/location |
| **Attendance recording** | RF-27, RF-28, RF-29, RF-31 | Record service performed, professional, observations |
| **Attendance workflow** | RF-32, RF-33, RF-34 | Basic status flow: triage → scheduled → in_progress → completed/cancelled |
| **Person history** | RF-21, RF-31 | View complete attendance timeline for a person |
| **User authentication** | RF-14, RF-15, RF-16 | Keycloak OIDC authentication with refresh tokens |
| **MFA for admin accounts** | RF-16 | Multi-factor authentication for admin accounts (via Keycloak) |
| **RBAC** | RF-17, RF-18 | Role-based access: admin, coordinator, professional, secretary, volunteer |
| **Service type catalog** | RF-28 | Predefined list of service types delivered as seed data |
| **Basic reports** | RF-40, RF-44 | Attendance count by period with CSV export |
| **Offline data entry** | RF-46, RF-47, RF-48 | Create persons and triages offline; sync when online |
| **Mobile-responsive UI** | RNF-11, RNF-13, RNF-19 | Mobile-first responsive design |
| **Audit logging** | RF-50, RF-51, RF-52 | Log all data access and mutations |
| **Campus scoping** | RNF-07, RNF-15 | Data segregation by campus |

> **MVP attendance states**: SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED (4 states). The FOLLOW_UP state is introduced in Phase 2.

> **Service type catalog**: Service type catalog is delivered as fixed seed data in MVP. The list of service types (LEGAL, MEDICAL, NUTRITIONAL, PHYSIOTHERAPY, SOCIAL, EDUCATIONAL, PSYCHOLOGICAL, OTHER) is predefined. Admin-configurable service types are a Phase 2 feature.

> **Campus scoping**: Each person and user belongs to exactly one campus in MVP. Multi-campus assignment is a Phase 2 feature.

### Won't Have in MVP

| Feature | Phase | Reason for Deferral |
|---------|-------|-------------------|
| Campaign management | Phase 2 | Not blocking for basic operations |
| Donation tracking | Phase 2 | Independent workflow |
| Document attachments | Phase 2 | Requires object storage setup |
| Consent capture with signature | Phase 2 | Complex mobile UX |
| FOLLOW_UP attendance state | Phase 2 | MVP uses linear flow only |
| Report charts/dashboards | Phase 2 | CSV export covers MVP needs |
| Team assignment to campaigns | Phase 2 | Depends on campaign management |
| Multi-campus per user | Phase 2 | Single campus assignment sufficient for MVP |
| Admin-configurable service types | Phase 2 | Fixed seed data sufficient for MVP |
| Multi-region support | Phase 3 | Brazil-only for MVP |
| WordPress API integration | Phase 3 | External integration |
| Donation receipts | Phase 3 | Depends on donation tracking |
| Advanced conflict resolution | Phase 3 | Last-write-wins is sufficient for MVP |

---

## MVP User Flows

### Flow 1: Register a Person (Secretary/Volunteer)
```
1. Open app → Person list
2. Tap "New Person"
3. Fill: name, CPF, birth date, phone, neighborhood
4. System checks for duplicates by CPF
5. Save (locally if offline)
6. Confirmation shown
```

### Flow 2: Record Triage (Volunteer at Event)
```
1. Open app → "New Triage"
2. Search or create person
3. Select: main complaint, requested services
4. Location auto-filled (from campaign if applicable)
5. Save (locally if offline)
6. Generates attendance record in "scheduled" state
```

### Flow 3: Record Attendance (Professional)
```
1. Open app → "My Attendances" (filtered by professional)
2. Select scheduled attendance
3. Record: service type, observations, recommendations
4. Mark as completed
5. Save (locally if offline)
```

### Flow 4: View Person History (Coordinator/Professional)
```
1. Open app → Search person
2. View person profile
3. See attendance timeline (all triages and attendances)
4. Tap any entry for details
```

### Flow 5: Generate Report (Coordinator/Admin)
```
1. Open app → Reports
2. Select date range
3. View attendance count summary
4. Export as CSV
```

---

## MVP Technical Scope

### Backend
- Go API server with chi router
- PostgreSQL database with core tables (see Phase 1 database tables below)
- Keycloak OIDC authentication with refresh tokens
- RBAC middleware
- Audit logging middleware
- Sync endpoints (push/pull)
- Report endpoint with CSV export
- Database migrations
- Seed data (service types, default permissions, test campus)

> **Phase 1 database tables**: campus, person, address, person_role, assisted_profile, app_user, service_type, triage, triage_requested_service, attendance, attendance_transition, audit_log.
>
> Tables deferred to Phase 2 migrations: campaign, campaign_team, document, consent, donation.

### Frontend
- React + TypeScript + Vite
- React OIDC integration (redirect to Keycloak login page)
- PWA with Service Worker (app shell caching)
- IndexedDB with Dexie.js (offline person, triage, attendance storage)
- Sync queue with background sync
- Pages: Dashboard, Person List, Person Form, Person Detail, Triage Form, Attendance Form, Attendance List, Report
- Tailwind CSS responsive layout
- React Hook Form for form handling

### Infrastructure
- Docker Compose (Go API + PostgreSQL + Keycloak + React dev server)
- GitHub Actions CI (Go tests + React build)
- Environment-based configuration

---

## MVP Acceptance Criteria

1. A volunteer can register a new person on a mobile phone in under 60 seconds
2. A volunteer can record a triage in under 90 seconds
3. A professional can record an attendance in under 2 minutes
4. Data created offline appears in the system within 30 seconds of connectivity restoration
5. Duplicate CPF registration is prevented with a clear error message
6. Users can only see data from their assigned campus
7. An admin can generate a CSV report of all attendances in a date range
8. All data mutations are recorded in the audit log
9. The application loads and is interactive within 3 seconds on a 3G connection (after first load)
10. The application is usable on screens as small as 320px wide

---

## Phase 2 Scope (Post-MVP)

| Feature | Requirements |
|---------|-------------|
| Campaign and event management | RF-36 to RF-39 |
| Team assignment | RF-25, RF-38 |
| Document attachments | RF-06, RF-30 |
| Consent capture with digital signature | RF-07, RF-08, RF-57, RF-58 |
| Donation tracking | RF-54, RF-55, RF-56 |
| Follow-up workflow state (FOLLOW_UP) | RF-35 |
| Report charts and dashboards | RF-41, RF-42, RF-43, RF-45 |
| Assisted person extended profile | RF-19, RF-20 |
| Person role management UI | RF-09 to RF-13 |
| Multi-campus per user | RNF-07 |
| Admin-configurable service types | RF-28 |

## Phase 3 Scope

| Feature | Requirements |
|---------|-------------|
| Multi-region support | RNF-14 |
| WordPress API integration | RNF-20 |
| Advanced conflict resolution | RF-49 |
| Donation receipts (PDF) | RF-55 |
| Consent revocation and data erasure | RF-58 |
| LGPD compliance reporting | RNF-01, RNF-02 |
| Backup and disaster recovery automation | RNF-16 |
