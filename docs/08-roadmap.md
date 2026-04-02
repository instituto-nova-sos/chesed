# 08 - Roadmap

## Phased Implementation Plan

### Overview

```
Phase 0 ──── Phase 1 ──── Phase 2 ──── Phase 3
Documentation  MVP Core    Extended     Scale &
& Setup        Features    Features     Compliance
(1 week)       (6-8 weeks) (6-8 weeks)  (4-6 weeks)
```

---

## Phase 0: Documentation and Project Setup (Week 1)

**Goal**: Establish project foundation, documentation, and development environment.

| # | Task | Dependencies | Status |
|---|------|-------------|--------|
| 0.1 | Complete architecture documentation (this deliverable) | None | In progress |
| 0.2 | Create Go project skeleton (`backend/`) | 0.1 |  |
| 0.3 | Create React project skeleton (`frontend/`) | 0.1 |  |
| 0.4 | Set up Docker Compose (Go + PostgreSQL + React dev) | 0.2, 0.3 |  |
| 0.5 | Set up GitHub Actions CI for Go + React | 0.4 |  |
| 0.6 | Create database migration framework | 0.2 |  |
| 0.7 | Set up linting and formatting (golangci-lint, eslint, prettier) | 0.2, 0.3 |  |

**Milestone**: Development environment fully functional; `docker compose up` starts all services.

---

## Phase 1: MVP Core (Weeks 2-9)

### Sprint 1: Auth and Infrastructure (Weeks 2-3)

| # | Task | Dependencies |
|---|------|-------------|
| 1.1 | Design and run database migrations (person, user, campus, service_type, audit_log) | 0.6 |
| 1.2 | Implement user registration and authentication (JWT) | 1.1 |
| 1.3 | Implement RBAC middleware | 1.2 |
| 1.4 | Implement audit logging middleware | 1.2 |
| 1.5 | Implement campus-scoped data access | 1.1, 1.3 |
| 1.6 | Create seed data (service types, default campus, admin user) | 1.1 |
| 1.7 | React project: auth pages (login, password recovery) | 0.3 |
| 1.8 | React: layout shell (navbar, sidebar, responsive) | 0.3 |
| 1.9 | React: auth context and API client with JWT handling | 1.7 |

**Milestone**: Users can log in; API enforces RBAC; audit log captures events.

### Sprint 2: Person Management (Weeks 4-5)

| # | Task | Dependencies |
|---|------|-------------|
| 2.1 | Person API: CRUD endpoints | 1.1 |
| 2.2 | Person API: search (name, CPF, fuzzy matching) | 2.1 |
| 2.3 | Person API: duplicate detection | 2.1 |
| 2.4 | Person role management API | 2.1 |
| 2.5 | React: Person list page (search, filter, paginate) | 1.8, 2.1 |
| 2.6 | React: Person form (create/edit) | 2.5 |
| 2.7 | React: Person detail page (profile + history timeline) | 2.5 |
| 2.8 | Set up IndexedDB with Dexie.js (person store) | 1.8 |
| 2.9 | Implement offline person creation | 2.6, 2.8 |

**Milestone**: Persons can be registered, searched, and viewed. Offline creation works.

### Sprint 3: Triage and Attendance (Weeks 6-7)

| # | Task | Dependencies |
|---|------|-------------|
| 3.1 | Triage API: create, list, detail | 1.1, 2.1 |
| 3.2 | Attendance API: CRUD with workflow transitions | 1.1, 2.1 |
| 3.3 | Attendance transition API: status change with audit | 3.2 |
| 3.4 | React: Triage form (complaint, services, person search) | 2.5, 3.1 |
| 3.5 | React: Attendance form (service type, observations, status) | 3.4, 3.2 |
| 3.6 | React: Attendance list (filter by status, professional, date) | 3.5 |
| 3.7 | React: Dashboard (counts, recent activity) | 3.2 |
| 3.8 | IndexedDB: triage and attendance offline stores | 2.8, 3.4, 3.5 |
| 3.9 | Offline triage and attendance creation | 3.8 |

**Milestone**: Complete triage → attendance workflow. Offline creation of both.

### Sprint 4: Sync, Reports, and Polish (Weeks 8-9)

| # | Task | Dependencies |
|---|------|-------------|
| 4.1 | Sync API: push endpoint (batch upload of offline records) | 2.1, 3.2 |
| 4.2 | Sync API: pull endpoint (fetch updates since last sync) | 4.1 |
| 4.3 | React: sync engine (queue, retry, conflict detection) | 2.8, 4.1 |
| 4.4 | PWA setup: Service Worker, manifest, install prompt | 1.8 |
| 4.5 | Report API: attendance count by period with CSV export | 3.2 |
| 4.6 | React: Report page (date range, table, CSV download) | 4.5 |
| 4.7 | End-to-end testing (critical flows) | All above |
| 4.8 | Performance optimization (lazy loading, caching) | All above |
| 4.9 | Security review (OWASP checklist, penetration basics) | All above |
| 4.10 | Deploy to staging environment | All above |

**Milestone**: MVP is feature-complete, tested, and deployed to staging.

---

## Phase 2: Extended Features (Weeks 10-17)

### Sprint 5: Campaigns and Teams (Weeks 10-11)

| # | Task | Dependencies |
|---|------|-------------|
| 5.1 | Campaign API: CRUD, status management | Phase 1 |
| 5.2 | Campaign team API: assign persons to campaigns | 5.1 |
| 5.3 | Link triage and attendance to campaigns | 5.1, 3.2 |
| 5.4 | React: Campaign list and detail pages | 5.1 |
| 5.5 | React: Campaign form and team assignment | 5.4 |
| 5.6 | React: Campaign dashboard (metrics per campaign) | 5.4 |

### Sprint 6: Documents and Consent (Weeks 12-13)

| # | Task | Dependencies |
|---|------|-------------|
| 6.1 | Object storage setup (S3/MinIO integration) | Phase 1 |
| 6.2 | Document upload API (person and attendance) | 6.1 |
| 6.3 | Consent API: create, revoke, list | Phase 1 |
| 6.4 | React: Document upload component | 6.2 |
| 6.5 | React: Consent form with signature capture | 6.3 |
| 6.6 | React: Consent history view | 6.5 |

### Sprint 7: Donations and Extended Profiles (Weeks 14-15)

| # | Task | Dependencies |
|---|------|-------------|
| 7.1 | Donation API: CRUD, link to campaign | Phase 1 |
| 7.2 | Assisted person profile API | Phase 1 |
| 7.3 | React: Donation form and list | 7.1 |
| 7.4 | React: Assisted person extended profile form | 7.2 |
| 7.5 | Person role management UI | Phase 1 |

### Sprint 8: Reports and Dashboards (Weeks 16-17)

| # | Task | Dependencies |
|---|------|-------------|
| 8.1 | Report API: by service type, by team, by campaign | Phase 1 |
| 8.2 | React: Report filters (type, team, campaign, period) | 8.1 |
| 8.3 | React: Chart components (Recharts) | 8.2 |
| 8.4 | React: Statistics dashboard | 8.3 |
| 8.5 | Follow-up workflow state (attendance reopen) | 3.2 |

**Phase 2 Milestone**: Full feature set operational except multi-region and compliance automation.

---

## Phase 3: Scale and Compliance (Weeks 18-23)

### Sprint 9: Multi-Region and Data Segregation (Weeks 18-19)

| # | Task | Dependencies |
|---|------|-------------|
| 9.1 | Multi-campus data isolation (PostgreSQL RLS or application-level) | Phase 2 |
| 9.2 | International document type support | Phase 2 |
| 9.3 | Multi-currency support for donations | 7.1 |
| 9.4 | Timezone handling for multi-region | Phase 2 |

### Sprint 10: LGPD and Compliance (Weeks 20-21)

| # | Task | Dependencies |
|---|------|-------------|
| 10.1 | Consent revocation with data anonymization | 6.3 |
| 10.2 | Data retention policy enforcement | Phase 2 |
| 10.3 | LGPD compliance report generation | 10.1 |
| 10.4 | Audit log viewer for compliance teams | Phase 1 |
| 10.5 | Donation receipt PDF generation | 7.1 |

### Sprint 11: Integration and Hardening (Weeks 22-23)

| # | Task | Dependencies |
|---|------|-------------|
| 11.1 | WordPress public API (campaigns, volunteer signup) | Phase 2 |
| 11.2 | Advanced sync conflict resolution UI | Phase 1 |
| 11.3 | Automated backup and disaster recovery | Phase 2 |
| 11.4 | Performance load testing (100 concurrent users) | Phase 2 |
| 11.5 | Security penetration testing | Phase 2 |
| 11.6 | Production deployment | All |

**Phase 3 Milestone**: Production-ready system with compliance, multi-region support, and integrations.

---

## Risk Mitigation

| Risk | Mitigation | Phase |
|------|-----------|-------|
| Offline sync complexity | Start with simple last-write-wins; defer advanced conflict resolution to Phase 3 | 1, 3 |
| Scope creep | Strict MVP definition; any new feature request goes to Phase 2+ backlog | 1 |
| Performance issues | Load test early (Sprint 4); optimize hot paths before Phase 2 | 1 |
| Data migration from Django | Write migration scripts in Phase 1; validate in Phase 2; execute in Phase 3 | 1-3 |
| Mobile UX quality | Involve real volunteers in testing at Sprint 4 milestone | 1 |
| Budget constraints | Use free-tier cloud services; single developer with AI assistance | All |
| Single point of failure (developer) | Comprehensive documentation (this suite); AI-executable specs | All |

---

## Technical Dependencies Graph

```
Docker + PostgreSQL (0.4)
    └── Database migrations (1.1)
        ├── Auth + RBAC (1.2, 1.3)
        │   └── All API endpoints
        ├── Person API (2.1)
        │   ├── Triage API (3.1)
        │   ├── Attendance API (3.2)
        │   ├── Campaign API (5.1)
        │   └── Donation API (7.1)
        └── Audit logging (1.4)
            └── All API endpoints

React shell (1.8)
    ├── Auth pages (1.7)
    ├── Person pages (2.5-2.7)
    │   ├── Triage form (3.4)
    │   └── Attendance form (3.5)
    └── IndexedDB setup (2.8)
        └── Offline creation (2.9, 3.8, 3.9)
            └── Sync engine (4.3)
```

---

## Suggested Development Cadence

- **Daily**: Commit working code; run tests
- **Per sprint (2 weeks)**: Demo to stakeholders; collect feedback
- **Per phase**: User acceptance testing with real volunteers
- **Continuous**: AI-assisted code generation following CLAUDE.md and CODEX.md guidelines
