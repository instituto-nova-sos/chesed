# Chesed Operating Model

This document is the **central index** for the AI-assisted delivery operating system. It shows how all artifacts connect across the delivery lifecycle.

---

## Delivery Lifecycle

```
┌─────────────────────────────────────────────────────────────────────┐
│                        FEATURE DELIVERY FLOW                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. PLAN                                                            │
│     ├── Read story from docs/09-backlog.md                          │
│     ├── [hook] pre-implement                                        │
│     ├── [skill] analyze-requirements                                │
│     ├── [rule] phase-boundary (verify Phase 1)                      │
│     ├── [rule] documentation-first (verify docs exist)              │
│     └── [template] feature-spec                                     │
│                                                                     │
│  2. DESIGN                                                          │
│     ├── [skill] design-backend-feature    (if backend)              │
│     ├── [skill] design-frontend-feature   (if frontend)             │
│     ├── [skill] design-offline-support    (if offline needed)       │
│     ├── [skill] design-test-plan                                    │
│     ├── [skill] review-security           (if security-sensitive)   │
│     └── [rule] offline-first-assessment                             │
│                                                                     │
│  3. IMPLEMENT                                                       │
│     ├── [playbook] implement-backend-endpoint   (if backend)        │
│     ├── [playbook] implement-frontend-page      (if frontend)       │
│     ├── [playbook] add-database-table           (if new table)      │
│     ├── [playbook] add-offline-support          (if offline)        │
│     ├── [hook] pre-api-change                   (before API work)   │
│     ├── [hook] pre-migration                    (before migrations) │
│     └── [rule] backlog-traceability             (commit messages)   │
│                                                                     │
│  4. VERIFY                                                          │
│     ├── [hook] post-api-change                                      │
│     ├── [hook] post-migration                                       │
│     ├── [skill] review-code                                         │
│     ├── [skill] review-api-contract                                 │
│     ├── [skill] review-migration          (if migration created)    │
│     ├── [playbook] conduct-security-review (if security-sensitive)  │
│     ├── [checklist] backend-feature-complete  (if backend)          │
│     ├── [checklist] frontend-feature-complete (if frontend)         │
│     ├── [checklist] api-review                (if API changed)      │
│     ├── [checklist] security-review           (if security-sens.)   │
│     └── [hook] pre-review                                           │
│                                                                     │
│  5. DOCUMENT                                                        │
│     ├── [skill] maintain-docs                                       │
│     └── [rule] documentation-first (update docs for deviations)     │
│                                                                     │
│  6. DELIVER (at sprint end)                                         │
│     ├── [playbook] prepare-sprint-delivery                          │
│     ├── [skill] assess-release-readiness                            │
│     ├── [checklist] sprint-release                                  │
│     ├── [skill] prepare-handoff                                     │
│     └── [hook] pre-release                                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Agent-Skill Mapping

| Agent | Primary Skills | When Active |
|-------|---------------|-------------|
| **tech-lead** | review-code, review-api-contract, assess-release-readiness, analyze-requirements | Architecture decisions, code review, sprint planning, quality oversight |
| **backend-engineer** | design-backend-feature, review-migration, design-test-plan | All backend implementation (handlers, services, repos, migrations) |
| **frontend-engineer** | design-frontend-feature, design-offline-support, design-test-plan | All frontend implementation (pages, components, hooks, PWA) |
| **security-engineer** | review-security, review-code | Auth changes, PII handling, RBAC, Keycloak config, LGPD compliance |

---

## Hook Trigger Map

| Hook | Fires When | Blocking? | Key Actions |
|------|-----------|-----------|-------------|
| **pre-implement** | Before starting any story | Yes | Verify story, phase, sprint, dependencies |
| **pre-api-change** | Before creating/modifying handlers | Yes | Verify endpoint documented, schemas match |
| **post-api-change** | After API endpoint changes | No* | Run API contract review, update docs |
| **pre-migration** | Before creating migration files | Yes | Verify table documented, in Phase 1 |
| **post-migration** | After creating migration files | No* | Run migration review, verify domain structs |
| **pre-review** | Before marking work complete | Yes | Run tests, lint, appropriate checklists |
| **pre-release** | Before sprint release | Yes | Run release readiness, tag git |

*Non-blocking but mandatory before story completion.

---

## Rule Enforcement Matrix

| Rule | Applies To | Enforcement Point |
|------|-----------|-------------------|
| **documentation-first** | All features | pre-implement hook, maintain-docs skill |
| **backlog-traceability** | All commits | Commit message review in pre-review hook |
| **phase-boundary** | All features | pre-implement hook, pre-migration hook |
| **security-review-triggers** | Auth, PII, RBAC, Keycloak changes | pre-review hook (conditional) |
| **offline-first-assessment** | All React pages | design-frontend-feature skill |

---

## Sprint Cadence

| Sprint | Focus | Key Artifacts Used |
|--------|-------|-------------------|
| **Sprint 1** (Auth & Infra) | Keycloak, OIDC middleware, RBAC, audit, Docker Compose, React shell | security-engineer agent, add-database-table playbook (12 tables), security-review checklist |
| **Sprint 2** (Person Mgmt) | Person CRUD, duplicate detection, search, React pages | backend-engineer + frontend-engineer agents, implement-backend-endpoint + implement-frontend-page playbooks |
| **Sprint 3** (Triage & Attendance) | Triage form, attendance workflow, state machine | backend-engineer + frontend-engineer agents, design-test-plan skill (workflow transitions) |
| **Sprint 4** (Sync, Reports) | Offline sync, PWA, CSV reports, polish | frontend-engineer agent, add-offline-support playbook, prepare-sprint-delivery playbook |

---

## Specialized Workflows

### Architecture Change
```
IDENTIFY → RESEARCH → PROPOSE (ADR template) → REVIEW (tech-lead) → DECIDE → UPDATE DOCS → IMPLEMENT
```
See: `workflows/architecture-change.md`

### Security-Sensitive Change
```
TRIGGER (rule) → EARLY REVIEW (security-engineer) → IMPLEMENT → FULL REVIEW (playbook) → REPORT (template) → REMEDIATE → VERIFY → UPDATE DOCS
```
See: `workflows/security-sensitive-change.md`

---

## Document References

The `.project-ai/` artifacts frequently reference these project docs:

| Doc | Referenced By |
|-----|---------------|
| `docs/03-requirements-catalog.md` | analyze-requirements skill |
| `docs/04-domain-model.md` | maintain-docs skill |
| `docs/05-architecture-proposal.md` | tech-lead agent, architecture-change workflow |
| `docs/07-mvp-scope.md` | phase-boundary rule, pre-implement hook |
| `docs/08-roadmap.md` | pre-implement hook, sprint-release checklist |
| `docs/09-backlog.md` | analyze-requirements skill, pre-implement hook |
| `docs/10-data-model.md` | review-migration skill, pre-migration hook, add-database-table playbook |
| `docs/11-api-design.md` | review-api-contract skill, pre-api-change hook, all implementation playbooks |
| `docs/12-offline-sync-strategy.md` | design-offline-support skill, add-offline-support playbook |
| `docs/13-security-and-compliance.md` | review-security skill, security-engineer agent |
| `docs/15-implementation-guidelines.md` | review-code skill, backend-engineer + frontend-engineer agents |
| `docs/16-iam-and-access-control.md` | review-security skill, security-engineer agent |
| `docs/17-security-test-strategy.md` | design-test-plan skill (security tests) |
| `docs/18-threat-model.md` | review-security skill, conduct-security-review playbook |
| `docs/19-secure-development-standard.md` | review-security skill, security-review checklist |
| `docs/20-keycloak-configuration.md` | security-engineer agent, maintain-docs skill |

---

## Full Artifact Index

### Skills (12)
| File | Purpose |
|------|---------|
| `skills/analyze-requirements.md` | Break story into implementation tasks |
| `skills/design-backend-feature.md` | Design Go handler→service→repository |
| `skills/design-frontend-feature.md` | Design React page→components→hooks |
| `skills/review-api-contract.md` | Validate implementation vs API docs |
| `skills/review-security.md` | Security review for auth, PII, RBAC |
| `skills/design-test-plan.md` | Design test cases per testing pyramid |
| `skills/review-code.md` | Code review with project quality standards |
| `skills/review-migration.md` | Database migration review |
| `skills/design-offline-support.md` | Offline sync feature design |
| `skills/maintain-docs.md` | Documentation maintenance |
| `skills/assess-release-readiness.md` | Sprint release evaluation |
| `skills/prepare-handoff.md` | Session handoff preparation |

### Agents (4)
| File | Role |
|------|------|
| `agents/tech-lead.md` | Architecture, code review, quality oversight |
| `agents/backend-engineer.md` | Go implementation |
| `agents/frontend-engineer.md` | React/TypeScript implementation |
| `agents/security-engineer.md` | OIDC, RBAC, PII, LGPD |

### Hooks (7)
| File | Trigger |
|------|---------|
| `hooks/pre-implement.md` | Before starting any feature |
| `hooks/pre-api-change.md` | Before modifying API endpoints |
| `hooks/post-api-change.md` | After modifying API endpoints |
| `hooks/pre-migration.md` | Before creating migrations |
| `hooks/post-migration.md` | After creating migrations |
| `hooks/pre-review.md` | Before marking work complete |
| `hooks/pre-release.md` | Before sprint release |

### Rules (5)
| File | Enforces |
|------|----------|
| `rules/documentation-first.md` | Docs before code |
| `rules/backlog-traceability.md` | Story IDs in commits |
| `rules/phase-boundary.md` | Phase 1/2/3 enforcement |
| `rules/security-review-triggers.md` | When security review is mandatory |
| `rules/offline-first-assessment.md` | Offline behavior on every page |

### Playbooks (6)
| File | Guides |
|------|--------|
| `playbooks/implement-backend-endpoint.md` | End-to-end backend endpoint |
| `playbooks/implement-frontend-page.md` | End-to-end frontend page |
| `playbooks/add-database-table.md` | New table with migration |
| `playbooks/add-offline-support.md` | Offline capability |
| `playbooks/conduct-security-review.md` | Security review workflow |
| `playbooks/prepare-sprint-delivery.md` | Sprint completion |

### Templates (5)
| File | For |
|------|-----|
| `templates/feature-spec.md` | Feature specification |
| `templates/adr.md` | Architecture decision record |
| `templates/api-change-proposal.md` | API change proposal |
| `templates/test-plan.md` | Test plan |
| `templates/security-review-report.md` | Security review report |

### Checklists (5)
| File | Gates |
|------|-------|
| `checklists/backend-feature-complete.md` | Backend quality gate |
| `checklists/frontend-feature-complete.md` | Frontend quality gate |
| `checklists/api-review.md` | API contract review |
| `checklists/security-review.md` | Security review |
| `checklists/sprint-release.md` | Sprint release readiness |

### Workflows (3)
| File | Shows |
|------|-------|
| `workflows/feature-delivery.md` | End-to-end feature flow |
| `workflows/architecture-change.md` | Architecture decision flow |
| `workflows/security-sensitive-change.md` | Security-sensitive change flow |
