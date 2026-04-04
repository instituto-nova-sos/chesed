# Chesed Operating Model

This document is the **central index** for the AI-assisted delivery operating system. It shows how all artifacts connect across the delivery lifecycle.

---

## Delivery Lifecycle

```
┌─────────────────────────────────────────────────────────────────────┐
│                        FEATURE DELIVERY FLOW                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  0. REFINE (optional, if story needs refinement)                    │
│     ├── [agent] product-analyst                                     │
│     ├── [skill] refine-requirements                                 │
│     └── [skill] validate-acceptance-criteria                        │
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
│     ├── [skill] design-api-contract      (if new endpoints)         │
│     ├── [skill] design-database-schema   (if new tables)            │
│     ├── [skill] design-backend-feature   (if backend)               │
│     ├── [skill] design-frontend-feature  (if frontend)              │
│     ├── [skill] design-offline-support   (if offline needed)        │
│     ├── [skill] design-test-plan                                    │
│     ├── [skill] review-security          (if security-sensitive)    │
│     └── [rule] offline-first-assessment                             │
│                                                                     │
│  3. IMPLEMENT                                                       │
│     ├── [playbook] implement-backend-endpoint   (if backend)        │
│     ├── [playbook] implement-frontend-page      (if frontend)       │
│     ├── [playbook] add-database-table           (if new table)      │
│     ├── [playbook] add-offline-support          (if offline)        │
│     ├── [hook] pre-api-change                   (before API work)   │
│     ├── [hook] pre-migration                    (before migrations) │
│     ├── [rule] backlog-traceability             (commit messages)   │
│     └── [rule] dependency-management            (new dependencies)  │
│                                                                     │
│  3.5 POST-IMPLEMENT                                                 │
│     └── [hook] post-implement (tests, lint, quality, docs check)    │
│                                                                     │
│  4. VERIFY                                                          │
│     ├── [hook] post-api-change                                      │
│     ├── [hook] post-migration                                       │
│     ├── [skill] review-code                                         │
│     ├── [skill] review-api-contract                                 │
│     ├── [skill] review-migration          (if migration created)    │
│     ├── [agent] qa-engineer + [skill] execute-test-plan             │
│     ├── [playbook] conduct-security-review (if security-sensitive)  │
│     ├── [checklist] backend-feature-complete  (if backend)          │
│     ├── [checklist] frontend-feature-complete (if frontend)         │
│     ├── [checklist] api-review                (if API changed)      │
│     ├── [checklist] security-review           (if security-sens.)   │
│     ├── [rule] test-coverage-enforcement                            │
│     └── [hook] pre-review                                           │
│                                                                     │
│  4.5 QUALITY VALIDATE                                               │
│     ├── [skill] maintainability-analysis                            │
│     ├── [skill] reliability-validation                              │
│     ├── [agent] reviewer (quality gate verdict)                     │
│     ├── [checklist] pr-quality                                      │
│     ├── [hook] pre-merge (blocking quality gate)                    │
│     ├── [rule] quality-gates                                        │
│     └── [playbook] refactor-for-quality (if gate fails)             │
│                                                                     │
│  5. DOCUMENT                                                        │
│     ├── [skill] maintain-docs                                       │
│     └── [rule] documentation-first (update docs for deviations)     │
│                                                                     │
│  6. DELIVER (at sprint end)                                         │
│     ├── [playbook] prepare-sprint-delivery                          │
│     ├── [skill] assess-release-readiness                            │
│     ├── [skill] performance-analysis          (sprint boundary)     │
│     ├── [checklist] sprint-release                                  │
│     ├── [skill] prepare-handoff                                     │
│     ├── [hook] pre-release                                          │
│     ├── [hook] pre-deploy                     (before deployment)   │
│     └── [hook] post-deploy                    (after deployment)    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Agent-Skill Mapping

| Agent | Primary Skills | When Active |
|-------|---------------|-------------|
| **tech-lead** | review-code, review-api-contract, assess-release-readiness, analyze-requirements, maintainability-analysis, reliability-validation, performance-analysis | Architecture decisions, code review, sprint planning, quality oversight |
| **backend-engineer** | design-backend-feature, design-api-contract, design-database-schema, review-migration, design-test-plan | All backend implementation (handlers, services, repos, migrations) |
| **frontend-engineer** | design-frontend-feature, design-offline-support, design-test-plan | All frontend implementation (pages, components, hooks, PWA) |
| **security-engineer** | review-security, review-code | Auth changes, PII handling, RBAC, Keycloak config, LGPD compliance |
| **reviewer** | review-code, maintainability-analysis, reliability-validation | Every PR — quality gate enforcement, clean code assessment |
| **devops-engineer** | infrastructure-setup, assess-release-readiness | Sprint 1 bootstrap, CI/CD changes, deployment, releases |
| **qa-engineer** | design-test-plan, execute-test-plan, validate-acceptance-criteria | After implementation, sprint boundary regression, test failures |
| **product-analyst** | refine-requirements, validate-acceptance-criteria, analyze-requirements | Requirements refinement, backlog grooming, phase transitions |

---

## Hook Trigger Map

| Hook | Fires When | Blocking? | Key Actions |
|------|-----------|-----------|-------------|
| **pre-implement** | Before starting any story | Yes | Verify story, phase, sprint, dependencies |
| **pre-api-change** | Before creating/modifying handlers | Yes | Verify endpoint documented, schemas match |
| **post-api-change** | After API endpoint changes | No* | Run API contract review, update docs |
| **pre-migration** | Before creating migration files | Yes | Verify table documented, in Phase 1 |
| **post-migration** | After creating migration files | No* | Run migration review, verify domain structs |
| **post-implement** | After implementation, before review | No* | Run tests, linters, quality assessment, verify docs, dependency check |
| **pre-review** | Before marking work complete | Yes | Run tests, lint, quality gate validation, appropriate checklists |
| **pre-merge** | Before merging PR | Yes | Enforce New Code Quality Gate, validate complexity, reviewer verdict |
| **pre-release** | Before sprint release | Yes | Run release readiness, Overall Code Quality Gate, performance budget, tag git |
| **pre-deploy** | Before deployment to any environment | Yes | Verify tests, checklist, migrations reversible, Docker builds, no secrets, git tag |
| **post-deploy** | After deployment completes | No* | Smoke tests, health checks, Keycloak verification, log review |

*Non-blocking but mandatory before story/release completion.

---

## Rule Enforcement Matrix

| Rule | Applies To | Enforcement Point |
|------|-----------|-------------------|
| **documentation-first** | All features | pre-implement hook, maintain-docs skill |
| **backlog-traceability** | All commits | Commit message review in pre-review hook |
| **phase-boundary** | All features | pre-implement hook, pre-migration hook |
| **security-review-triggers** | Auth, PII, RBAC, Keycloak changes | pre-review hook (conditional) |
| **offline-first-assessment** | All React pages | design-frontend-feature skill |
| **quality-gates** | All code changes | pre-merge hook, pre-release hook, reviewer agent |
| **dependency-management** | New go.mod/package.json entries | pre-review hook, reviewer agent, tech-lead approval |
| **test-coverage-enforcement** | All PRs and sprint releases | pre-merge hook, qa-engineer agent |
| **performance-budget** | Sprint releases, new endpoints | performance-analysis skill, pre-release hook |
| **api-versioning-strategy** | Breaking changes to existing endpoints | pre-api-change hook, review-api-contract skill |

---

## Sprint Cadence

| Sprint | Focus | Key Artifacts Used |
|--------|-------|-------------------|
| **Sprint 1** (Auth & Infra) | Keycloak, OIDC middleware, RBAC, audit, Docker Compose, React shell | devops-engineer agent, bootstrap-project-infrastructure playbook, security-engineer agent, add-database-table playbook (12 tables), security-review checklist |
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

### Bug Fix
```
BUG REPORTED → REPRODUCE (failing test) → ROOT CAUSE → DESIGN FIX → IMPLEMENT + REGRESSION TEST → REVIEW → MERGE
```
See: `workflows/bug-fix-workflow.md`

### Hotfix (Production Emergency)
```
INCIDENT → ASSESS → HOTFIX BRANCH → MINIMAL FIX → ABBREVIATED REVIEW → STAGING → PRODUCTION → INCIDENT REPORT → BACKLOG STORY
```
See: `workflows/hotfix-workflow.md` + `playbooks/rollback-and-hotfix.md`

### Performance Optimization
```
IDENTIFY (analysis) → PROFILE (measure) → DESIGN FIX → IMPLEMENT → VERIFY (re-measure) → DOCUMENT (report)
```
See: `workflows/performance-optimization.md`

---

## Prompt Execution Chain

The `prompts/` directory contains 10 standalone, reusable prompts that drive each stage of the development lifecycle. Each prompt is independent but feeds into the next in sequence.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      PROMPT EXECUTION CHAIN                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  requirement-analysis ──→ architecture-design ──→ task-breakdown         │
│         │                        │                      │               │
│         │                        │                      ▼               │
│         │                        │              ┌───────────────┐       │
│         │                        │              │ PARALLEL IMPL │       │
│         │                        │              │ backend-impl  │       │
│         │                        │              │ frontend-impl │       │
│         │                        │              └───────┬───────┘       │
│         │                        │                      │               │
│         │                        │                      ▼               │
│         │                        │              test-generation         │
│         │                        │                      │               │
│         │                        │              ┌───────┴───────┐       │
│         │                        │              │ PARALLEL REVIEW│      │
│         │                        │              │ code-review    │      │
│         │                        │              │ security-review│      │
│         │                        │              └───────┬───────┘       │
│         │                        │                      │               │
│         │                        │              performance-review      │
│         │                        │                      │               │
│         │                        │              release-readiness       │
│         │                        │                      │               │
│         ▼                        ▼                      ▼               │
│   [REQUIREMENTS]          [ARCHITECTURE]          [DEPLOYED RELEASE]    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Blocking Dependencies Between Prompts

| Prompt | Requires Output From | Blocks |
|--------|---------------------|--------|
| requirement-analysis | — (entry point) | architecture-design, task-breakdown |
| architecture-design | requirement-analysis | task-breakdown, backend/frontend-implementation |
| task-breakdown | requirement-analysis, architecture-design | backend/frontend-implementation |
| backend-implementation | architecture-design, task-breakdown | code-review, test-generation |
| frontend-implementation | architecture-design, task-breakdown | code-review, test-generation |
| test-generation | backend/frontend-implementation | code-review |
| code-review | implementation + tests | release-readiness |
| security-review | implementation | release-readiness |
| performance-review | implementation | release-readiness |
| release-readiness | all above | deployment |

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
| `docs/quality/quality-profiles.md` | review-code skill, reviewer agent, all engineer agents |
| `docs/quality/clean-code-guidelines.md` | review-code skill, maintainability-analysis skill, reviewer agent |
| `docs/quality/quality-gates.md` | pre-merge hook, pre-release hook, reviewer agent, quality-gates rule |
| `docs/quality/complexity-guidelines.md` | review-code skill, maintainability-analysis skill, all engineer agents |

---

## Full Artifact Index

### Skills (20)
| File | Purpose |
|------|---------|
| `skills/analyze-requirements.md` | Break story into implementation tasks |
| `skills/assess-release-readiness.md` | Sprint release evaluation |
| `skills/design-api-contract.md` | Design REST API contracts from requirements |
| `skills/design-backend-feature.md` | Design Go handler→service→repository |
| `skills/design-database-schema.md` | Design database schemas from domain model |
| `skills/design-frontend-feature.md` | Design React page→components→hooks |
| `skills/design-offline-support.md` | Offline sync feature design |
| `skills/design-test-plan.md` | Design test cases per testing pyramid |
| `skills/execute-test-plan.md` | Validate test plan completeness against actual tests |
| `skills/infrastructure-setup.md` | Design Docker, Makefile, CI/CD, environment config |
| `skills/maintain-docs.md` | Documentation maintenance |
| `skills/maintainability-analysis.md` | Complexity, coupling, cohesion, duplication analysis |
| `skills/performance-analysis.md` | API, query, frontend, and sync performance analysis |
| `skills/prepare-handoff.md` | Session handoff preparation |
| `skills/refine-requirements.md` | Transform business requests into structured stories |
| `skills/reliability-validation.md` | Error handling, state consistency, fault tolerance validation |
| `skills/review-api-contract.md` | Validate implementation vs API docs |
| `skills/review-code.md` | Code review with project quality standards |
| `skills/review-migration.md` | Database migration review |
| `skills/review-security.md` | Security review for auth, PII, RBAC |
| `skills/validate-acceptance-criteria.md` | Evaluate acceptance criteria completeness and testability |

### Agents (8)
| File | Role |
|------|------|
| `agents/tech-lead.md` | Architecture, code review, quality oversight |
| `agents/backend-engineer.md` | Go implementation |
| `agents/frontend-engineer.md` | React/TypeScript implementation |
| `agents/security-engineer.md` | OIDC, RBAC, PII, LGPD |
| `agents/reviewer.md` | Quality gate enforcement, clean code assessment, PR review |
| `agents/devops-engineer.md` | Infrastructure, Docker, CI/CD, deployment |
| `agents/qa-engineer.md` | Test coverage, regression testing, acceptance validation |
| `agents/product-analyst.md` | Requirements refinement, acceptance criteria, backlog grooming |

### Hooks (11)
| File | Trigger |
|------|---------|
| `hooks/pre-implement.md` | Before starting any feature |
| `hooks/pre-api-change.md` | Before modifying API endpoints |
| `hooks/post-api-change.md` | After modifying API endpoints |
| `hooks/pre-migration.md` | Before creating migrations |
| `hooks/post-migration.md` | After creating migrations |
| `hooks/post-implement.md` | After implementation, before review |
| `hooks/pre-review.md` | Before marking work complete |
| `hooks/pre-merge.md` | Before merging PR (quality gate enforcement) |
| `hooks/pre-release.md` | Before sprint release |
| `hooks/pre-deploy.md` | Before deployment to any environment |
| `hooks/post-deploy.md` | After deployment completes |

### Rules (10)
| File | Enforces |
|------|----------|
| `rules/documentation-first.md` | Docs before code |
| `rules/backlog-traceability.md` | Story IDs in commits |
| `rules/phase-boundary.md` | Phase 1/2/3 enforcement |
| `rules/security-review-triggers.md` | When security review is mandatory |
| `rules/offline-first-assessment.md` | Offline behavior on every page |
| `rules/quality-gates.md` | Quality gate enforcement — complexity, duplication, coverage, ratings |
| `rules/dependency-management.md` | New dependency evaluation and approval |
| `rules/test-coverage-enforcement.md` | Layer-specific test coverage thresholds |
| `rules/performance-budget.md` | Response time and resource budgets |
| `rules/api-versioning-strategy.md` | Breaking change versioning policy |

### Playbooks (11)
| File | Guides |
|------|--------|
| `playbooks/implement-backend-endpoint.md` | End-to-end backend endpoint |
| `playbooks/implement-frontend-page.md` | End-to-end frontend page |
| `playbooks/add-database-table.md` | New table with migration |
| `playbooks/add-offline-support.md` | Offline capability |
| `playbooks/conduct-security-review.md` | Security review workflow |
| `playbooks/prepare-sprint-delivery.md` | Sprint completion |
| `playbooks/implement-with-quality.md` | Quality-aware feature implementation |
| `playbooks/refactor-for-quality.md` | Refactoring to meet quality gates |
| `playbooks/bootstrap-project-infrastructure.md` | Initial project infrastructure setup |
| `playbooks/rollback-and-hotfix.md` | Emergency rollback and hotfix procedures |
| `playbooks/onboard-new-feature-domain.md` | Introducing new domain areas (phase transitions) |

### Templates (8)
| File | For |
|------|-----|
| `templates/feature-spec.md` | Feature specification |
| `templates/adr.md` | Architecture decision record |
| `templates/api-change-proposal.md` | API change proposal |
| `templates/test-plan.md` | Test plan |
| `templates/security-review-report.md` | Security review report |
| `templates/performance-report.md` | Performance analysis findings |
| `templates/incident-report.md` | Post-incident documentation |
| `templates/sprint-retrospective.md` | Sprint metrics and process evaluation |

### Checklists (7)
| File | Gates |
|------|-------|
| `checklists/backend-feature-complete.md` | Backend quality gate (includes quality profile compliance) |
| `checklists/frontend-feature-complete.md` | Frontend quality gate (includes quality profile compliance) |
| `checklists/api-review.md` | API contract review |
| `checklists/security-review.md` | Security review |
| `checklists/sprint-release.md` | Sprint release readiness (includes overall quality gates) |
| `checklists/pr-quality.md` | PR-level quality gate enforcement |
| `checklists/refactoring.md` | Refactoring safety and correctness |

### Workflows (6)
| File | Shows |
|------|-------|
| `workflows/feature-delivery.md` | End-to-end feature flow |
| `workflows/architecture-change.md` | Architecture decision flow |
| `workflows/security-sensitive-change.md` | Security-sensitive change flow |
| `workflows/bug-fix-workflow.md` | Non-emergency bug fix flow |
| `workflows/hotfix-workflow.md` | Emergency production fix flow |
| `workflows/performance-optimization.md` | Performance issue resolution flow |

### Prompts (10)
| File | Lifecycle Stage |
|------|----------------|
| `prompts/requirement-analysis.md` | Stage 1: Product understanding → structured requirements |
| `prompts/architecture-design.md` | Stage 2: Requirements → API contracts, schemas, layer designs |
| `prompts/task-breakdown.md` | Stage 3: Design → ordered, agent-assignable implementation tasks |
| `prompts/backend-implementation.md` | Stage 4: Tasks → production-ready Go code |
| `prompts/frontend-implementation.md` | Stage 4: Tasks → production-ready React/TypeScript code |
| `prompts/code-review.md` | Stage 5: Implementation → quality-gated review verdict |
| `prompts/test-generation.md` | Stage 6: Implementation → comprehensive test suite |
| `prompts/security-review.md` | Stage 7: Code → security assessment with threat model mapping |
| `prompts/performance-review.md` | Stage 8: Code → performance budget evaluation |
| `prompts/release-readiness.md` | Stage 9: Sprint → go/no-go release decision |
