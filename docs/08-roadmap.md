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

## Phase 0: Documentation and Project Setup (Week 1) — COMPLETE

**Goal**: Establish project foundation, documentation, and development environment.

**Status**: All tasks completed. Documentation, project scaffolding, Docker Compose, and Keycloak realm configuration are in place.

| # | Task | Dependencies | Status |
|---|------|-------------|--------|
| 0.1 | Complete architecture documentation (docs 01-20 + quality/) | None | Done |
| 0.2 | Create Go project skeleton (`backend/`) | 0.1 | Done |
| 0.3 | Create React project skeleton (`frontend/`) | 0.1 | Done |
| 0.4 | Set up Docker Compose (Go + PostgreSQL + React dev + Keycloak) | 0.2, 0.3 | Done |
| 0.5 | Set up Keycloak container in Docker Compose with `chesed` realm | 0.4 | Done |
| 0.6 | Configure Keycloak realm (roles, protocol mappers, brute-force protection) | 0.5 | Done |
| 0.7 | Export Keycloak realm configuration to `keycloak/realm-export.json` | 0.6 | Done |
| 0.8 | Security architecture review (Keycloak IAM, threat model, security test strategy) | 0.6 | Done |

**Milestone**: Development environment fully functional; `docker compose up` starts all services. Keycloak realm configured and exported as code. Full documentation suite (docs 01-18) complete.

---

## Phase 1: MVP Core (Weeks 2-9)

### Phase 1 Scope Notes

**Database tables for Phase 1**: `campus`, `person`, `address`, `person_role`, `assisted_profile`, `app_user`, `service_type`, `triage`, `triage_requested_service`, `attendance`, `attendance_transition`, `audit_log`.

**Explicitly NOT in Phase 1** (deferred to Phase 2):
- Campaign management tables or endpoints
- Donation tracking tables or endpoints
- Document attachment tables or endpoints
- Consent capture tables or endpoints
- `FOLLOW_UP` attendance state

### Sprint 1: Auth and Infrastructure (Weeks 2-3) — DONE

| # | Task | Dependencies | Status |
|---|------|-------------|--------|
| 1.1 | Design and run database migrations (campus, person, address, person_role, assisted_profile, app_user, service_type, audit_log) | 0.4 | Done |
| 1.2 | Implement OIDC token validation middleware (`coreos/go-oidc` + Keycloak JWKS) | 1.1 | Done |
| 1.3 | Implement local user auto-provisioning (first login creates `app_user` from Keycloak `sub` claim) | 1.2 | Done |
| 1.4 | Implement RBAC middleware (roles from Keycloak token claims) | 1.2 | Done |
| 1.5 | Implement audit logging middleware | 1.2 | Done |
| 1.6 | Implement campus-scoped data access | 1.1, 1.4 | Done |
| 1.7 | Create seed data (service types, default campus) | 1.1 | Done |
| 1.8 | React OIDC integration (redirect to Keycloak via keycloak-js adapter) | 0.3 | Done |
| 1.9 | React: layout shell (navbar, sidebar, responsive) | 0.3 | Done |
| 1.10 | React: auth context wrapping keycloak-js | 1.8 | Done |
| 1.11 | MFA configuration for ADMIN role in Keycloak | 0.7 | Done |
| 1.12 | Keycloak realm configuration as code (S02.9) | 0.7 | Done |

**Milestone**: Users authenticate via Keycloak; API validates OIDC tokens and enforces RBAC; audit log captures events.

### Sprint 2: Person Management (Weeks 4-5) — DONE

| # | Task | Dependencies | Status |
|---|------|-------------|--------|
| 2.1 | Person API: CRUD endpoints | 1.1 | Done |
| 2.2 | Person API: search (name, CPF, fuzzy matching) | 2.1 | Done |
| 2.3 | Person API: duplicate detection | 2.1 | Done |
| 2.4 | Person role management API | 2.1 | Done |
| 2.5 | React: Person list page (search, filter, paginate) | 1.9, 2.1 | Done |
| 2.6 | React: Person form (create/edit) | 2.5 | Done |
| 2.7 | React: Person detail page (profile + history timeline) | 2.5 | Done |
| 2.8 | Set up IndexedDB with Dexie.js (person store) | 1.9 | Done |
| 2.9 | Implement offline person creation | 2.6, 2.8 | Done |

**Milestone**: Persons can be registered, searched, and viewed. Offline creation works.

### Sprint 3: Triage and Attendance (Weeks 6-7) — DONE

| # | Task | Dependencies | Status |
|---|------|-------------|--------|
| 3.1 | Triage API: create, list, detail | 1.1, 2.1 | Done |
| 3.2 | Attendance API: CRUD with workflow transitions | 1.1, 2.1 | Done |
| 3.3 | Attendance transition API: status change with audit | 3.2 | Done |
| 3.4 | React: Triage form (complaint, services, person search) | 2.5, 3.1 | Done |
| 3.5 | React: Attendance form (service type, observations, status) | 3.4, 3.2 | Done |
| 3.6 | React: Attendance list (filter by status, professional, date) | 3.5 | Done |
| 3.7 | React: Dashboard (counts, recent activity) | 3.2 | Done |
| 3.8 | IndexedDB: triage and attendance offline stores | 2.8, 3.4, 3.5 | Done |
| 3.9 | Offline triage and attendance creation | 3.8 | Done |

**Milestone**: Complete triage → attendance workflow. Offline creation of both.

### Sprint 4: Sync, Reports, and Polish (Weeks 8-9) — DONE

Backend sync (push/pull) and reports (with CSV export) are implemented and
integration-tested (commit #28). The offline sync drainer, conflict surfacing,
PWA setup, and end-to-end test infrastructure landed across commits #28–#31.
Performance optimization (route-level code splitting) and the security review
are complete; the two hardening follow-ups the review deferred are closed by the
`feat/phase1-hardening-followups` work (see note below). See "Parallelization
Model" below for the critical path that was followed.

| # | Task | Dependencies | Status |
|---|------|-------------|--------|
| 4.1 | Sync API: push endpoint (batch upload of offline records) | 2.1, 3.2 | Done |
| 4.2 | Sync API: pull endpoint (fetch updates since last sync) | 4.1 | Done |
| 4.3 | React: sync engine (Dexie v2, `useOnlineSync` drainer, pull-merge, conflict surfacing) | 2.8, 4.1 | Done |
| 4.4 | PWA setup: Service Worker, manifest, install prompt | 1.9 | Done |
| 4.5 | Report API: attendance count by period with CSV export | 3.2 | Done |
| 4.6 | React: Report page (date range, table, CSV download) | 4.5 | Done |
| 4.7 | End-to-end testing (critical flows) | All above | Done |
| 4.8 | Performance optimization (lazy loading, caching) | All above | Done |
| 4.9 | Security review (OWASP checklist, penetration basics) | All above | Done |
| 4.10 | Deploy to staging environment | All above | Todo (ops) |

> **Security review follow-ups (4.9)**: the two deferred hardening items from
> `docs/security-review-sprint4.md` are now closed — (1) sync push rejects
> cross-campus `person_id`/`triage_id` references (Finding 3), and (2) RBAC 403
> denials are written to `audit_log` as `ACCESS_DENIED` (Finding 4).

> **4.10** is a deployment/operations step, not application code; its
> prerequisites are tracked in the "Infra / configuration checklist" of
> `docs/security-review-sprint4.md`.

**Milestone**: MVP is feature-complete and tested; staging deployment (4.10) is
the remaining operational step.

---

## Parallelization Model

This section describes how the **remaining Phase 1 work** is sequenced and where
real concurrency exists. It supersedes the generic dependency graph at the bottom
of this document for day-to-day Sprint 4 planning.

### Remaining critical path (serial chain)

The offline sync drainer (task 4.3 / epic E05 frontend) is a strict serial chain
because each stage consumes the previous stage's output:

```
Dexie v2 schema (S05.1)
    └── useOnlineSync drainer (S05.3)        # flushes syncQueue on reconnect
            └── pull-merge (S05.4)           # reconciles server changes into cache
                    └── conflict surfacing UI # last-write-wins + show conflicted field
```

This chain cannot be meaningfully parallelized: the drainer needs the v2 stores,
pull-merge needs the drainer's synced-marking, and the conflict UI needs pull-merge
to detect conflicts. It is the binding constraint for Sprint 4 completion.

### Parallel tracks (independent of the chain)

These run concurrently with the serial chain because they touch disjoint files
and depend only on already-completed work:

| Track | Tasks / stories | Independent because |
|-------|-----------------|---------------------|
| E2E infrastructure | 4.7 (Playwright + real stack) | Exercises existing endpoints/UI; needs no new sync code to start scaffolding |
| Reports | 4.5 / 4.6 (S06.1–S06.3) | Already complete; isolated read surface, no offline coupling |
| PWA | 4.4 (Service Worker, manifest, install prompt) | Shell-level concern; orthogonal to the sync queue |
| Status indicator | S05.5 | Depends only on Dexie v2 (S05.1) and `useOfflineStatus`, not on the drainer |

### Where the parallelism lever actually is now

In early Phase 1, the dominant parallelism axis was **backend × frontend**: while
one stream built Go endpoints, another built React pages against the documented
contract. That lever is now largely **spent** — the backend Phase 1 surface (auth,
person, triage, attendance, sync, reports) is implemented and integration-tested,
so the remaining work is frontend-heavy and converges on a single serial chain.

The real lever for the remaining work is therefore:

1. **Process × feature concurrency** — run the delivery process (tests, lint,
   review gate, DoD) for one slice while authoring the next independent slice,
   rather than waiting for a backend counterpart that no longer exists.
2. **Independent frontend hooks** — the parallel tracks above are gated by file
   disjointness, not by a backend dependency. Hooks like `useOnlineSync`,
   `useOfflineStatus`, and the report hooks can be developed and tested in
   isolation as long as they do not edit the same files concurrently.

Coordinate concurrent work by the `depends_on` graph in `docs/09-backlog.md`
(`make status` renders the current board); never start a story whose `depends_on`
set is not yet `done` (see `.project-ai/rules/ready-definition.md`).

---

## Phase 2: Extended Features (Weeks 10-17)

### Sprint 5: Campaigns and Teams (Weeks 10-11) — DONE

| # | Task | Dependencies | Status |
|---|------|-------------|--------|
| 5.1 | Campaign API: CRUD, status management | Phase 1 | Done |
| 5.2 | Campaign team API: assign persons to campaigns | 5.1 | Done |
| 5.3 | Link triage and attendance to campaigns | 5.1, 3.2 | Done |
| 5.4 | React: Campaign list and detail pages | 5.1 | Done |
| 5.5 | React: Campaign form and team assignment | 5.4 | Done |
| 5.6 | React: Campaign dashboard (metrics per campaign) | 5.4 | Done |

> **5.3 note**: the API accepts and campus-validates `campaign_id` on triage
> and attendance creation; surfacing a campaign selector in the triage and
> attendance forms (and in the offline sync payload) is a documented follow-up
> for the next slice.

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
| Keycloak complexity | Use default themes, minimal realm customization, realm export for reproducibility | 1 |
| Offline sync complexity | Start with last-write-wins; manual conflict resolution deferred to Phase 3 | 1, 3 |
| Scope creep | Strict MVP definition; any new feature request goes to Phase 2+ backlog | 1 |
| Performance issues | Load test early (Sprint 4); optimize hot paths before Phase 2 | 1 |
| Mobile UX quality | Involve real volunteers in testing at Sprint 4 milestone | 1 |
| Budget constraints | Use free-tier cloud services; single developer with AI assistance | All |
| Single point of failure (developer) | Comprehensive documentation (this suite); AI-executable specs | All |

---

## Technical Dependencies Graph

```
Docker + PostgreSQL + Keycloak (0.4, 0.5)
    ├── Keycloak realm config (0.6, 0.7)
    │   └── OIDC middleware + RBAC (1.2, 1.4)
    │       └── All API endpoints
    └── Database migrations (1.1)
        ├── User auto-provisioning (1.3)
        ├── Person API (2.1)
        │   ├── Triage API (3.1)
        │   ├── Attendance API (3.2)
        │   ├── Campaign API (5.1)
        │   └── Donation API (7.1)
        └── Audit logging (1.5)
            └── All API endpoints

React shell (1.9)
    ├── Keycloak OIDC integration (1.8, 1.10)
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
