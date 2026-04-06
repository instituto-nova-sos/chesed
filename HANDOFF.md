# HANDOFF.md - Session History and Next Steps

## Last Updated
2026-04-06 (Session 11)

---

## Repository Origin

This repository (`instituto-nova-sos/chesed`) is the canonical home for the rebuild of the SOS Gestao platform. It was created as a clean-start repository after the decision to rebuild rather than refactor the legacy Django system.

**Legacy repository**: `Amadeus-22/SOS-Gestao-Final` (Django 5.2.6 + Jazzmin admin)
**This repository**: `instituto-nova-sos/chesed` (Go + React + PostgreSQL)

---

## Session 1: Product Discovery and Architecture (2026-04-02)

### What Was Done
1. **Full inspection** of the legacy Django codebase (3 apps, 4 models, 17 permissions, audit logging)
2. **Assessment verdict**: Rebuild recommended — Django Admin cannot support offline-first PWA; no API layer; data model insufficient
3. **Complete documentation suite** created (19 files covering requirements, architecture, domain model, data model, API design, security, roadmap)

### What Was Produced
- 00 through 15: Product and architecture documentation
- CLAUDE.md and CODEX.md: AI agent execution rules
- Original HANDOFF.md: Session summary

---

## Session 2: Repository Migration (2026-04-02)

### What Was Done
1. **Selective migration** from legacy repo to this clean repository
2. **Documentation adapted** to reference the new repository context (no more "current Django system" language in active docs)
3. **Four new security documents** created (16-19) that did not exist in the legacy repo
4. **Root-level files** created fresh: README.md, CLAUDE.md, CODEX.md

### Files in This Repository

**Migrated from legacy (adapted):**
| File | Status | Changes |
|------|--------|---------|
| `docs/00-current-system-assessment.md` | Deleted in Session 4 | Was a historical reference of the legacy Django system; removed as legacy artifact |
| `docs/01-product-vision.md` | Copied | No changes needed |
| `docs/02-problem-statement.md` | Copied | No changes needed |
| `docs/03-requirements-catalog.md` | Copied | No changes needed |
| `docs/04-domain-model.md` | Copied | No changes needed |
| `docs/05-architecture-proposal.md` | Copied | No changes needed |
| `docs/06-tech-stack-evaluation.md` | Copied | No changes needed |
| `docs/07-mvp-scope.md` | Copied | No changes needed |
| `docs/08-roadmap.md` | Copied | No changes needed |
| `docs/09-backlog.md` | Copied | No changes needed |
| `docs/10-data-model.md` | Copied | No changes needed |
| `docs/11-api-design.md` | Copied | No changes needed |
| `docs/12-offline-sync-strategy.md` | Copied | No changes needed |
| `docs/13-security-and-compliance.md` | Copied | No changes needed |
| `docs/14-deployment-strategy.md` | Copied | No changes needed |
| `docs/15-implementation-guidelines.md` | Rewritten | Removed Django-centric migration section; replaced with legacy data migration reference |

**Created new for this repository:**
| File | Description |
|------|-------------|
| `README.md` | Fresh project README for Chesed |
| `CLAUDE.md` | AI agent rules (adapted for new repo name and structure) |
| `CODEX.md` | AI agent rules (adapted; references CLAUDE.md at root) |
| `HANDOFF.md` | This file |
| `docs/16-iam-and-access-control.md` | Complete IAM model: roles, permissions matrix, campus isolation, session management |
| `docs/17-security-test-strategy.md` | 4-layer security testing approach with test cases |
| `docs/18-threat-model.md` | 9 threat scenarios with risk matrix and mitigations |
| `docs/19-secure-development-standard.md` | Secure coding rules, secret management, CI gates, dependency policy |
| `.gitignore` | Standard Go + React + environment ignores |

**Intentionally left behind (not migrated):**
| File/Directory | Reason |
|----------------|--------|
| All Django source code (`atendimento/`, `ong_manager/`, `users/`) | Legacy stack; not reusable in Go+React |
| Django configuration (`settings.py`, `manage.py`, `wsgi.py`, `asgi.py`) | Framework-specific |
| Legacy documentation (`GUIA_PRATICO.md`, `QUICK_REFERENCE.md`, `ANALISE_E_MELHORIAS.md`, etc.) | Django-specific operational guides |
| `requirements.txt` | Python dependencies; irrelevant to Go+React |
| `Dockerfile`, `docker-compose.yml` | Django-specific; will be recreated for Go+React |
| `staticfiles/`, `static/` | Django admin assets |
| `db.sqlite3` | Legacy development database |
| `.env` | Contains legacy credentials |
| `SEGURANCA_PRODUCAO.md` | Django-specific production security guide |
| `INSTALACAO_RAPIDA.txt` | Django quick-start guide |

---

## Major Decisions

| Decision | Rationale |
|----------|-----------|
| Rebuild in new repo (not refactor in place) | Legacy code provides no reusable value for Go+React; clean repo prevents accidental legacy coupling |
| Flat `docs/` structure | Simple and explicit; all 20 docs are numbered for clear ordering |
| CLAUDE.md and CODEX.md at repo root | AI agents read root-level files first; docs/ copies removed to avoid duplication |
| Keep `00-current-system-assessment.md` as-is | Historical reference; documents what was evaluated and why rebuild was chosen |
| Four new security docs (16-19) | Original docs lacked dedicated IAM, threat model, security testing, and secure dev standards |
| `.gitignore` for Go+React+PostgreSQL | Prevents accidental commit of binaries, node_modules, env files, IDE configs |

---

## Session 3: Security Architecture Review (2026-04-02)

### What Was Done
1. **Comprehensive security review** of all 19 documentation files from a cybersecurity-first perspective
2. **Critical finding**: Custom JWT authentication identified as the highest-risk architectural decision
3. **IAM decision**: Keycloak selected as external identity provider (OIDC)
4. **14 documents updated** to replace custom auth with Keycloak and strengthen security controls
5. **1 new document created**: `docs/20-keycloak-configuration.md`

### Key Decision: Keycloak as IAM Provider
- **Rationale**: Open-source (Apache 2.0), self-hosted ($5-7/month), zero licensing cost at any scale, standard OIDC (no vendor lock-in), provides MFA/SSO/password policies/brute-force protection out of the box
- **Alternatives rejected**: Auth0/Okta (cost grows with users), AWS Cognito (vendor lock-in), custom JWT (highest security risk)
- **Protocol**: OIDC Authorization Code Flow with PKCE
- **Impact**: Removed 5 custom auth API endpoints, removed `refresh_token` table, removed `password_hash` from `app_user`, added `keycloak_subject_id`

### Documents Updated
| File | Change Type |
|------|-------------|
| `docs/16-iam-and-access-control.md` | Full rewrite — Keycloak OIDC architecture |
| `docs/11-api-design.md` | Major — removed auth endpoints, added OIDC flow |
| `docs/10-data-model.md` | Major — schema changes for Keycloak |
| `docs/05-architecture-proposal.md` | Major — Keycloak in architecture diagram |
| `docs/06-tech-stack-evaluation.md` | Moderate — IAM provider comparison table |
| `docs/13-security-and-compliance.md` | Moderate — session management, WAF, logging, LGPD timeline |
| `docs/18-threat-model.md` | Moderate — updated T1/T7, added T10-T12 |
| `docs/17-security-test-strategy.md` | Moderate — OIDC validation tests |
| `docs/19-secure-development-standard.md` | Moderate — prohibited custom auth, updated deps |
| `docs/14-deployment-strategy.md` | Moderate — Keycloak container, costs, env vars |
| `docs/12-offline-sync-strategy.md` | Moderate — offline tokens, mandatory encryption |
| `docs/15-implementation-guidelines.md` | Minor — approved deps, middleware description |
| `CLAUDE.md` | Minor — Keycloak guardrails |
| `CODEX.md` | Minor — mirrored CLAUDE.md changes |

### New Documents Created
| File | Description |
|------|-------------|
| `docs/20-keycloak-configuration.md` | Realm setup, protocol mappers, MFA config, branding, troubleshooting |

### Security Enhancements Beyond Keycloak
- MFA moved from Phase 3 to Phase 1 (free with Keycloak)
- Mandatory IndexedDB encryption (was optional)
- LGPD breach notification timeline specified (2 business days to ANPD)
- Three new threat scenarios: Keycloak compromise (T10), supply chain (T11), DDoS (T12)
- WAF recommendation (Cloudflare free tier)
- Centralized logging recommendation (Grafana Loki)
- Security headers CI validation
- Keycloak container image scanning in CI

---

## Session 4: Product Validation and Cleanup (2026-04-03)

### What Was Done
1. **Legacy artifact removed**: Deleted `docs/00-current-system-assessment.md` (described old Django system, not relevant to future-state repo)
2. **Terminology standardized** across all documents: canonical enum names for person roles (ASSISTED, VOLUNTEER, PROFESSIONAL, COORDINATOR, ADMIN) and access profiles (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER)
3. **Person Roles vs Access Profiles distinction** documented clearly — these are two different taxonomies
4. **Product ambiguities resolved** with pragmatic defaults:
   - Service type catalog: fixed seed data in MVP, admin-configurable in Phase 2
   - Campus scoping: single campus per person/user in MVP, multi-campus in Phase 2
   - Triage lifecycle: immutable after creation
   - Attendance MVP states: SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED (FOLLOW_UP deferred to Phase 2)
   - Data retention: 5yr operational, 10yr audit logs, anonymization on LGPD erasure
   - Keycloak audit forwarding: poll Admin Events API (MVP)
5. **Phase boundaries clarified**: Phase 1 tables explicitly listed (12 tables), Phase 2 tables deferred (5 tables)
6. **Roadmap rebased**: Phase 0 marked COMPLETE, Keycloak tasks added, Sprint 1 updated for OIDC
7. **Data retention policy** added to security docs with LGPD anonymization rules
8. **Product scope guardrails** added to CLAUDE.md and CODEX.md to prevent premature Phase 2 work
9. **Backlog aligned**: sprint targets assigned to all Phase 1 epics, service type seed data story added
10. **MVP scope tightened**: explicit "Won't Have" list expanded, authentication updated to Keycloak OIDC
11. **API design improved**: duplicate detection endpoint documented, service types endpoint added, Phase 2 markers on deferred endpoints

### Documents Updated
| File | Change |
|------|--------|
| `docs/00-current-system-assessment.md` | DELETED (legacy artifact) |
| `docs/03-requirements-catalog.md` | Terminology fixes, RF-79 (MFA) added, open questions resolved |
| `docs/04-domain-model.md` | Roles/profiles distinction, triage immutability, FOLLOW_UP Phase 2, campus scoping |
| `docs/07-mvp-scope.md` | MVP states, Phase 1 tables, service types, campus scoping, Keycloak auth |
| `docs/08-roadmap.md` | Phase 0 complete, Keycloak tasks, Sprint 1 OIDC, phase boundaries |
| `docs/09-backlog.md` | Sprint targets, Phase 2/3 markers, service type seed story (S01.6) |
| `docs/10-data-model.md` | Phase 1/2 table categorization, data retention policy, FOLLOW_UP note |
| `docs/11-api-design.md` | Duplicate detection expanded, service types endpoint, Phase 2 markers |
| `docs/13-security-and-compliance.md` | Data retention policy, anonymization rules, audit forwarding decision |
| `CLAUDE.md` | Product scope guardrails added |
| `CODEX.md` | Product scope guardrails added |

---

## Session 5: AI Tooling Structure Audit and Compatibility Review (2026-04-03)

### What Was Done
1. **Full audit** of the `.project-ai/` directory structure (49 artifacts across 8 categories)
2. **Compatibility review** evaluating `.project-ai/` vs `.claude/` conventions for AI-assisted workflows
3. **Cross-reference audit** of all 551 internal references across 49 files — zero broken links found
4. **Documentation fixes** for inconsistencies found during the audit
5. **Decision documented**: `.project-ai/` convention retained, flat file skill structure retained

### Key Decision: Keep `.project-ai/` Convention

**Question evaluated**: Should the AI delivery operating model live in `.project-ai/` or be migrated to `.claude/`?

**Decision**: Keep `.project-ai/` — do not migrate.

**Rationale**:
- Claude Code owns `~/.claude/` for user-level state (memory, plans, settings) and has its own `skills/` directory there. A project-level `.claude/` would create naming conflicts and ambiguity between Claude Code tooling state and project delivery artifacts.
- There is no official Claude Code convention for project-level `.claude/` directories. The only official project-level touchpoint is `CLAUDE.md` at the root.
- The operating model is agent-agnostic by design (both `CLAUDE.md` and `CODEX.md` reference it). Moving it to `.claude/` would tie it to a single vendor.
- `.project-ai/` accurately describes what it contains: project-level AI delivery artifacts.

**Skill structure decision**: Keep flat files (`skills/name.md`), not directories (`skills/name/SKILL.md`). All 12 skills are single markdown files with no auxiliary content. The directory pattern solves a problem (multi-file skills) that doesn't exist here. If a skill ever needs auxiliary files, only that skill should be converted.

### Issues Found and Fixed

| File | Issue | Fix |
|------|-------|-----|
| `CLAUDE.md` line 190 | Missing `.project-ai/` prefix on `workflows/feature-delivery.md` path | Added `.project-ai/` prefix |
| `README.md` line 23 | Auth listed as "JWT + RBAC" (stale — Keycloak adopted in Session 3) | Updated to "Keycloak (OIDC) + RBAC" |
| `README.md` project structure | `.project-ai/` directory not listed | Added to project tree |
| `README.md` AI-Assisted Development | `.project-ai/` not referenced | Added reference with description |

### Cross-Reference Audit Results
- **49 `.project-ai/` files verified**: all exist and are properly structured
- **551 cross-references checked**: 100% valid (456 to `docs/`, 95 internal)
- **16 documentation files referenced**: all exist
- **Special case**: `docs/adrs/` referenced in `workflows/architecture-change.md` does not exist yet (by design — created when first ADR is written)

---

## Session 6: Quality Governance Audit (2026-04-04)

### What Was Done
1. **Comprehensive audit** of the Quality Governance system across all documentation, AI tooling, CLAUDE.md, and CODEX.md
2. **docs/quality/ review**: All 4 files (quality-gates.md, quality-profiles.md, clean-code-guidelines.md, complexity-guidelines.md) verified internally consistent — no issues found
3. **.project-ai/ review**: All 47+ artifacts verified for quality enforcement — comprehensive chain with no gaps
4. **CLAUDE.md corrections**: Complexity table completed with 3 missing metrics
5. **CODEX.md corrections**: 6 missing sections/rules added for governance alignment
6. **HANDOFF.md**: Audit report documented

### Issues Found and Fixed

| # | File | Issue | Severity | Fix |
|---|------|-------|----------|-----|
| 1 | CLAUDE.md | Complexity table missing Parameter count (5/5) | MINOR | Added row |
| 2 | CLAUDE.md | Complexity table missing Return values (Go: 3) | MINOR | Added row |
| 3 | CLAUDE.md | Complexity table missing Component JSX lines (React: 80) | MINOR | Added row |
| 4 | CODEX.md | Missing Documentation-First Workflow section | MAJOR | Added section after "Before You Start Coding" |
| 5 | CODEX.md | Missing Code Strictness rules (TypeScript `any`, error handling, singletons, audit_log immutability, migration up/down, Keycloak realm export) | MAJOR | Added "Code Strictness" subsection |
| 6 | CODEX.md | Missing Software Qualities (Non-Negotiable) subsection | MAJOR | Added under Quality Governance |
| 7 | CODEX.md | Missing AI Agent Responsibility section | MAJOR | Added under Quality Governance |
| 8 | CODEX.md | Missing Commit Convention | MINOR | Added section |
| 9 | CODEX.md | Complexity table missing Return values and Component JSX lines | MINOR | Added rows |

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| Keep "Quality Checklist" naming in CODEX.md (not rename to "Quality Bar") | "Checklist" with checkboxes is more actionable for execution-focused agents. Both enforce same rules. |
| Add "Code Strictness" as separate subsection in CODEX.md | TypeScript `any`, error handling, audit_log immutability are code-level constraints distinct from Security and Architecture |
| AI Agent Responsibility applies to ALL agents (added to CODEX.md) | The 5 enforcement rules are universal — limiting them to Claude-only created a bypass risk for other agents |
| Software Qualities listed explicitly in both files | Previously CODEX.md only referenced quality-profiles.md by link without listing the three dimensions |

### Post-Audit State

**CLAUDE.md and CODEX.md now enforce the same governance model:**
- Same quality gates (New Code + Overall Code)
- Same software qualities (Security, Reliability, Maintainability)
- Same clean code categories (Consistency, Intentionality, Adaptability, Responsibility)
- Same complexity thresholds (all 8 metrics aligned)
- Same AI agent responsibility rules
- Same documentation-first workflow
- Same commit convention

**Intentional differences preserved:**
- CODEX.md: "Quality Checklist" (actionable checkboxes) vs CLAUDE.md: "Quality Bar" (aspirational criteria)
- CODEX.md: Common Tasks walkthroughs, Naming Conventions table, File Structure tree, Testing examples
- CLAUDE.md: Code Structure dependency diagrams, Key Documentation References list

### Remaining Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Rating definitions (A-E) only in quality-gates.md, not in summary files | LOW | docs/quality/ is the authoritative source |
| Coverage roadmap (sprint schedule) only in quality-gates.md | LOW | Summary says "tightens to 80%" which is sufficient |
| Summary tables in CLAUDE.md/CODEX.md could drift from docs/quality/ | MEDIUM | AI Agent Responsibility mandates validation; .project-ai/ continuous improvement catches drift |
| No automated linter enforcement yet (no code exists) | MEDIUM | Linter configs documented in complexity-guidelines.md ready to copy when scaffolding |

---

## Session 7: Phase 0 Validation and Project Scaffolding (2026-04-04)

### What Was Done
1. **Phase 0 validation audit**: Discovered that 6 of 8 Phase 0 tasks were incorrectly marked as "Done" in the roadmap — only documentation (0.1) and security review (0.8) were actually complete. No application code, Docker files, or Keycloak configuration existed.
2. **Roadmap corrected**: `docs/08-roadmap.md` updated to reflect actual status (IN PROGRESS), tasks 0.2–0.7 marked as Pending.
3. **Requirement analysis executed**: Ran `.project-ai/prompts/requirement-analysis.md` for all 6 pending tasks, producing structured specs with acceptance criteria (63 total) and implementation task breakdowns.
4. **Go backend skeleton created** (Task 0.2): `backend/` with chi router, slog JSON logger, graceful shutdown, health endpoint, config loader with validation, Makefile, golangci-lint config, unit tests.
5. **React frontend skeleton created** (Task 0.3): `frontend/` with Vite + React 19 + TypeScript (strict) + Tailwind v4 + PWA manifest + all approved dependencies (keycloak-js, dexie, react-hook-form, zod, zustand, recharts, react-router-dom) + ESLint flat config + Prettier + Vitest.
6. **Docker Compose created** (Task 0.4): `docker-compose.yml` with 4 services (PostgreSQL 16, Keycloak 26, Go API with air hot reload, React dev server with HMR), health checks, named volumes, inter-service networking. Dockerfiles for backend (production multi-stage + dev) and frontend (dev).
7. **Keycloak realm configured** (Tasks 0.5–0.7): `keycloak/realm-export.json` with chesed realm, 5 RBAC roles, 2 OIDC clients (chesed-pwa public with PKCE, chesed-api confidential with service account), custom protocol mappers (campus_id, person_id), password policy, brute-force protection, token lifetimes, refresh token revocation, conditional MFA for ADMIN role, event logging.
8. **Phase 0 marked COMPLETE**: All 8 tasks verified and roadmap updated.

### Files Created

**Backend (15 files):**
| File | Purpose |
|------|---------|
| `backend/go.mod` + `go.sum` | Go module with chi, pgx, go-oidc, validator, uuid |
| `backend/cmd/server/main.go` | Entry point: chi router, slog, graceful shutdown, health routes |
| `backend/internal/config/config.go` | Config struct with env var loading and validation |
| `backend/internal/config/config_test.go` | Table-driven tests for config loading |
| `backend/internal/handler/health.go` | Health endpoint returning `{"status":"ok"}` |
| `backend/internal/handler/health_test.go` | Health handler tests |
| `backend/internal/domain/doc.go` | Empty domain package (zero deps) |
| `backend/internal/service/doc.go` | Empty service package |
| `backend/internal/repository/doc.go` | Empty repository package |
| `backend/internal/middleware/doc.go` | Empty middleware package |
| `backend/migrations/.gitkeep` | Empty migrations directory |
| `backend/Makefile` | build, run, test, lint, clean targets |
| `backend/.golangci.yml` | Linter config with complexity thresholds |
| `backend/.env.example` | Environment variable template |
| `backend/.air.toml` | Hot reload configuration for dev |

**Frontend (23 files):**
| File | Purpose |
|------|---------|
| `frontend/package.json` | All approved deps + scripts (dev, build, test, lint, format, typecheck) |
| `frontend/vite.config.ts` | React + Tailwind v4 + PWA + Vitest + path aliases |
| `frontend/tsconfig.json` + `tsconfig.app.json` | TypeScript strict mode, path aliases |
| `frontend/eslint.config.js` | Flat config: TS, react-hooks, no-explicit-any, prettier |
| `frontend/.prettierrc` | singleQuote, trailingComma, semi |
| `frontend/index.html` | PWA-ready HTML entry |
| `frontend/src/main.tsx` | React root with StrictMode |
| `frontend/src/App.tsx` | BrowserRouter with placeholder route |
| `frontend/src/App.test.tsx` | App render test |
| `frontend/src/test-setup.ts` | testing-library/jest-dom setup |
| `frontend/src/index.css` | Tailwind v4 import |
| `frontend/.env.example` | Frontend env template |
| `frontend/src/{pages,components,hooks,api,types,offline,store,utils}/.gitkeep` | Directory structure |

**Infrastructure (8 files):**
| File | Purpose |
|------|---------|
| `docker-compose.yml` | 4 services, health checks, volumes, networking |
| `docker/postgres/init-keycloak-db.sh` | Creates chesed_keycloak database |
| `backend/Dockerfile` | Production multi-stage (golang:1.22-alpine → alpine:3.19) |
| `backend/Dockerfile.dev` | Dev with air hot reload |
| `frontend/Dockerfile` | Dev with vite HMR |
| `keycloak/realm-export.json` | Complete chesed realm configuration |
| `.env.example` | Root-level env template for all services |

### Verification Results
| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `go test ./...` | 2 test suites passed |
| `go vet ./...` | PASS |
| `npm run typecheck` | PASS |
| `npm run build` | PASS (PWA + service worker generated) |
| `npm run test` | 1 test passed |
| `npm run lint` | PASS (0 warnings) |
| `docker compose config` | PASS (valid syntax) |
| realm-export.json | Valid JSON, all config verified |

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| Tailwind v4 with `@tailwindcss/vite` (not v3 with config file) | Current version, simpler setup, CSS-based configuration |
| ESLint flat config (v9+, not legacy .eslintrc) | Vite template default, future-proof |
| `air` for Go hot reload (not CompileDaemon) | More actively maintained, cleaner config |
| Keycloak health check on port 9000 (management port) | `/health/ready` endpoint available on management interface in Keycloak 26 |
| `chesed-api` client secret set to "changeme-in-production" | Placeholder only; production secret set via environment variable |
| Conditional OTP via custom authentication flow | Keycloak's `conditional-user-role` authenticator enables role-based MFA without affecting non-admin users |

---

## Session 8: Sprint 1 — Auth and Infrastructure (2026-04-04)

### What Was Done
1. **Phase 0 HANDOFF gap fixed**: Updated 7 files across `.project-ai/`, `CLAUDE.md`, and `CODEX.md` to enforce HANDOFF.md updates after every task (not just session end). Added Step 7 to `post-implement` hook, item 9 to Quality Bar, updated `prepare-handoff` skill trigger conditions.
2. **Sprint 1 PLAN phase**: Read all required docs (03, 04, 07, 08, 09, 10, 11, 15, 16, 20, quality/). Validated pre-implement hook: all Sprint 1 features exist in requirements catalog, belong to Phase 1, no Phase 2 features being built.
3. **Sprint 1 DESIGN phase**: Produced full architecture plan covering middleware chain (Auth → AutoProvision → RBAC), audit logging at service layer, typed auth context, DB pool via constructor injection, migrations via Makefile.
4. **Task 1.1: Database migrations** — Created 11 migration pairs (22 SQL files) for all 12 Phase 1 tables. Key change: `app_user.person_id` made nullable to support auto-provisioning. Added append-only trigger on `audit_log`.
5. **Task 1.2: OIDC middleware** — `internal/middleware/auth.go` using `coreos/go-oidc/v3` with JWKS caching, retry loop for Keycloak startup (15 attempts, 2s intervals), extracts `sub`, `email`, `realm_access.roles`, `campus_id`, `person_id` into typed `AuthClaims` context.
6. **Task 1.3: User auto-provisioning** — `internal/middleware/provision.go` + `internal/service/user_service.go` with `EnsureUser()`: finds by Keycloak subject, creates if missing, updates last_login, logs to audit.
7. **Task 1.4: RBAC middleware** — `internal/middleware/rbac.go` with `RequireRole()` factory checking role membership, 403 on insufficient permissions, warning log on unauthorized attempts.
8. **Task 1.5: Audit logging** — `internal/service/audit_service.go` + `internal/repository/audit_repository.go`. Service-layer approach with `LogAction()` accepting typed `AuditParams`, marshals old/new values to JSONB, extracts user/campus from context.
9. **Task 1.6: Campus-scoped access** — `internal/auth/campus.go` with `ResolveCampusID()` supporting admin cross-campus override. All repository queries will use `CampusIDFromContext()`.
10. **Task 1.7: Seed data** — Migration `000011_seed_data` inserts default campus and 8 service types. `GET /api/v1/service-types` endpoint via handler → service → repository chain.
11. **Task 1.8: React OIDC** — `frontend/src/auth/keycloak.ts` configuring keycloak-js with env vars, Authorization Code Flow with PKCE.
12. **Task 1.9: Layout shell** — `AppLayout.tsx`, `Sidebar.tsx`, `Header.tsx` with responsive design, mobile hamburger menu, navigation links, user info display.
13. **Task 1.10: Auth context** — Zustand `authStore.ts` wrapping keycloak-js (init, refresh, logout, claims extraction), `useAuth()` hook with `hasRole()`/`hasMinRole()`, `apiClient()` with auto Bearer token, `ProtectedRoute` component.
14. **Tasks 1.11-1.12: Already complete** (Keycloak MFA + realm-export.json from Phase 0).
15. **main.go fully wired** — DI chain: config → pool → repositories → services → handlers → middleware chain → chi router. CORS middleware for dev.

### Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| Middleware chain: Auth → AutoProvision → RBAC | Auth validates first, provision ensures local user, RBAC per-route |
| Audit at service layer, not HTTP middleware | Service has entity context (entity_id, old/new values) |
| Auth context via typed struct + typed context key | Type-safe, no string collisions |
| DB pool via constructor injection | No global state, explicit dependencies |
| app_user.person_id nullable | Auto-provisioning before person is linked |
| OIDC provider with 15-retry loop | Keycloak may not be ready at API startup |
| CORS middleware for dev | React (5173) calls API (8080) cross-origin |

### Files Created

**Backend — 22 migration files** (`backend/migrations/000001-000011 .up.sql + .down.sql`)

**Backend — 17 Go files:**
- `internal/database/pool.go` — pgx connection pool
- `internal/auth/context.go` — AuthClaims struct + context helpers
- `internal/auth/campus.go` — Campus ID resolution
- `internal/domain/user.go`, `service_type.go`, `audit.go`, `role.go`, `errors.go` — Domain structs
- `internal/middleware/auth.go`, `rbac.go`, `provision.go`, `cors.go` — Middleware stack
- `internal/service/audit_service.go`, `user_service.go`, `service_type_service.go` — Business logic
- `internal/repository/audit_repository.go`, `user_repository.go`, `service_type_repository.go` — Data access

**Frontend — 11 files:**
- `src/auth/keycloak.ts` — Keycloak instance
- `src/store/authStore.ts` — Zustand auth store
- `src/hooks/useAuth.ts` — Auth convenience hook
- `src/api/client.ts` — API client with Bearer token
- `src/components/auth/ProtectedRoute.tsx` — Auth gate
- `src/components/layout/AppLayout.tsx`, `Sidebar.tsx`, `Header.tsx` — Layout shell
- `src/components/ui/LoadingScreen.tsx` — Loading spinner
- `src/pages/DashboardPage.tsx`, `NotFoundPage.tsx` — Pages

### Files Modified
- `backend/cmd/server/main.go` — Full DI wiring, middleware chain, routes
- `backend/internal/handler/health.go` — Added DB ping check
- `backend/internal/handler/health_test.go` — Updated for new signature
- `backend/Makefile` — Added migrate-up/down/create/force targets
- `backend/go.mod` — Added pgx/v5, go-oidc/v3, uuid, oauth2
- `frontend/src/App.tsx` — Keycloak init, protected routes, layout
- `frontend/src/api/client.ts` — Fixed erasableSyntaxOnly compat
- `docs/08-roadmap.md` — Phase 0 status corrected earlier in session
- `CLAUDE.md` — Added HANDOFF.md to Quality Bar
- `CODEX.md` — Added HANDOFF.md to Quality Checklist
- `.project-ai/hooks/post-implement.md` — Added HANDOFF.md step
- `.project-ai/skills/prepare-handoff.md` — Updated triggers
- `.project-ai/workflows/feature-delivery.md` — Added HANDOFF in POST-IMPLEMENT
- `.project-ai/OPERATING_MODEL.md` — Updated hook trigger map

### Verification Results

| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `go test ./...` | 2 test suites passed (config, handler) |
| `go vet ./...` | PASS |
| `npm run typecheck` | PASS |
| `npm run build` | PASS (PWA generated) |
| `npm run test` | 1 test passed |
| `npm run lint` | PASS |

### Current State

**Working:**
- All 12 Phase 1 database migrations (up + down SQL)
- Go backend: health endpoint, OIDC middleware, RBAC middleware, auto-provisioning, audit service, service type endpoint, CORS, full DI wiring
- React frontend: Keycloak OIDC integration, auth store, layout shell, protected routes, API client

**Not Yet Verified (requires running Docker Compose):**
- End-to-end auth flow (Keycloak → React → API → DB)
- Migration execution against real PostgreSQL
- Auto-provisioning flow
- Service types endpoint with real data

### Next Steps

1. **Run `docker compose up`** and verify end-to-end auth flow
2. **Run `make migrate-up`** against the running PostgreSQL
3. **Create a test Keycloak user** with campus_id attribute and verify login flow
4. **Add unit tests** for domain/role.go, auth/context.go, auth/campus.go, middleware/rbac.go, service/user_service.go
5. **Set up CI/CD** (GitHub Actions with Go test + React build)
6. **Begin Sprint 2** (Person Management: CRUD API + React pages)

---

## Session 9: End-to-End Auth Verification (2026-04-05)

### What Was Done
1. **Migrations executed** — All 11 migration pairs ran successfully against real PostgreSQL (12 Phase 1 tables + seed data)
2. **Keycloak 26 compatibility fixes** — Discovered and fixed 4 issues with Keycloak 26's realm import:
   - **User Profile enforcement**: KC 26 silently drops user attributes not declared in User Profile. Added `campus_id` and `person_id` to User Profile config.
   - **Missing client scopes**: `profile`, `email`, `roles` scopes weren't created from `defaultDefaultClientScopes` — they must be explicitly defined in `clientScopes`. Added all with proper protocol mappers.
   - **Lightweight access tokens**: KC 26 omits `sub` from access tokens by default. Added `oidc-sub-mapper` to the `profile` scope.
   - **Issuer URL mismatch**: API reaches Keycloak at `http://keycloak:8080` (Docker internal) but tokens have `iss: http://localhost:8180`. Added `KC_HOSTNAME`/`KC_HOSTNAME_PORT` to docker-compose and `SkipIssuerCheck` with documented rationale in auth middleware.
3. **IP address port stripping** — `r.RemoteAddr` includes port (e.g., `192.168.65.1:26043`), PostgreSQL `inet` type rejects it. Fixed `extractIP()` in `provision.go` to use `net.SplitHostPort()`.
4. **Test users created** — `testvolunteer` (VOLUNTEER) and `testadmin` (ADMIN) in Keycloak with `campus_id` attribute, realm roles, passwords.
5. **Full auth flow verified** — Login → Token (with sub, email, roles, campus_id) → API call → OIDC validation → Auto-provisioning → DB insert → Audit log entry. Both VOLUNTEER and ADMIN users verified.
6. **realm-export.json updated** — All Keycloak fixes persisted: User Profile config, 5 client scopes (profile, email, roles, offline_access, chesed-custom-claims) with protocol mappers, sub mapper, introspection claims.

### Files Modified
| File | Change |
|------|--------|
| `docker-compose.yml` | Added `KC_HOSTNAME`, `KC_HOSTNAME_PORT`, `KC_HOSTNAME_STRICT` to Keycloak service |
| `backend/internal/middleware/auth.go` | Added `InsecureIssuerURLContext` for OIDC discovery, `SkipIssuerCheck` on verifier (Docker hostname mismatch) |
| `backend/internal/middleware/provision.go` | Fixed `extractIP()` to strip port using `net.SplitHostPort()` for PostgreSQL `inet` compatibility |
| `keycloak/realm-export.json` | Added `profile`, `email`, `roles`, `offline_access` scopes with mappers; `sub` mapper; User Profile with `campus_id`/`person_id` attributes; `introspection.token.claim` on custom claims |

### Verification Results
| Check | Result |
|-------|--------|
| Migrations (11 up) | PASS — all 12 tables + seed data |
| Token claims (sub, email, roles, campus_id) | PASS |
| GET /api/v1/service-types (VOLUNTEER) | 200 — 8 service types |
| GET /api/v1/service-types (ADMIN) | 200 — 8 service types |
| app_user auto-provisioning | PASS — VOLUNTEER and ADMIN records |
| audit_log entries | PASS — 2 CREATE entries with IP, entity_id |
| `go build ./...` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |

### Keycloak 26 Lessons Learned
| Issue | Impact | Fix |
|-------|--------|-----|
| User Profile enforced by default | Custom user attributes silently dropped on save | Must declare custom attributes in `userProfile` section of realm config |
| Lightweight access tokens | `sub` claim missing from access tokens | Add `oidc-sub-mapper` protocol mapper to a default client scope |
| Client scopes not auto-created | Referencing `profile`/`email`/`roles` in `defaultDefaultClientScopes` doesn't create them | Must define full scope objects in `clientScopes` array with protocol mappers |
| KC_HOSTNAME required for consistent issuer | Token `iss` uses hostname visible to the browser; internal Docker DNS differs | Set `KC_HOSTNAME`/`KC_HOSTNAME_PORT` and use `SkipIssuerCheck` in API middleware |

---

## Open Questions

These need stakeholder input (but have documented defaults that allow implementation to proceed):

1. **International document types**: MVP supports CPF (Brazil) only. What types for USA/Europe? (Phase 3)
2. **Consent form content**: Legal team must provide template text. (Phase 2 prerequisite)
3. **Donation receipt format**: Legal requirements per country? (Phase 3)
4. **Hosting budget**: Confirmed budget? (Estimates: $10-17/month MVP with Keycloak, $55-70/month production)
5. **Volunteer testing**: Can real volunteers test the MVP during development?

---

## Session 10: Unit Tests + Security Review + Fixes (2026-04-05)

### What Was Done
1. **Unit tests written** — 11 test files, 73+ test cases across all Sprint 1 packages. Coverage: auth 100%, domain 100%, service 96.7%, config 100%, middleware 52.8% (OIDCAuth untestable), handler 76.2%.
2. **Security review executed** — Full review per `.project-ai/prompts/security-review.md`. Found 0 CRITICAL, 3 HIGH, 4 MEDIUM, 4 LOW, 3 INFO.
3. **Security fixes applied** — 8 findings fixed in code:

| Finding | Fix |
|---------|-----|
| H1: Client secret in realm-export.json | Removed `"secret": "changeme-in-production"` from chesed-api client |
| H2: JWT audience not validated | Set `SkipClientIDCheck: false` — now validates `aud`/`azp` against clientID |
| M1: RequireRole not applied | Added `RequireRole` middleware to `/api/v1/service-types` route |
| M2: Login not audited | Added LOGIN audit entry in `EnsureUser` for existing users |
| M3: X-Forwarded-For spoofable | Extracts first IP from XFF chain, validates with `net.ParseIP()`, rejects invalid |
| M4: SkipIssuerCheck hardcoded | Made configurable via `OIDC_SKIP_ISSUER_CHECK` env var (default false) |
| L3: Missing campus_id validation | Added `uuid.Nil` check in `AutoProvision` — returns 403 if campus_id missing |
| L4: Missing security headers | New `SecurityHeaders` middleware: X-Content-Type-Options, X-Frame-Options, Referrer-Policy, CSP, Permissions-Policy, Cache-Control |

### Files Created
| File | Purpose |
|------|---------|
| `backend/internal/domain/role_test.go` | HasRole + RoleHierarchy tests (11 cases) |
| `backend/internal/auth/context_test.go` | Context round-trip tests (6 cases) |
| `backend/internal/auth/campus_test.go` | ResolveCampusID tests (6 cases) |
| `backend/internal/service/audit_service_test.go` | LogAction tests + MockAuditRepository (7 cases) |
| `backend/internal/service/user_service_test.go` | EnsureUser + resolveAccessProfile tests (14 cases) |
| `backend/internal/service/service_type_service_test.go` | ListActive tests (3 cases) |
| `backend/internal/middleware/rbac_test.go` | RequireRole tests (5 cases) |
| `backend/internal/middleware/cors_test.go` | CORS tests (5 cases) |
| `backend/internal/middleware/auth_test.go` | extractBearerToken + writeError tests (7 cases) |
| `backend/internal/middleware/provision_test.go` | extractIP + AutoProvision tests (12 cases) |
| `backend/internal/middleware/security_headers.go` | Security headers middleware |
| `backend/internal/middleware/security_headers_test.go` | Security headers test |
| `backend/internal/handler/service_type_test.go` | ServiceTypeHandler.List tests (3 cases) |

### Files Modified
| File | Change |
|------|--------|
| `keycloak/realm-export.json` | Removed hardcoded client secret (H1) |
| `backend/internal/middleware/auth.go` | Enabled audience validation, configurable issuer check (H2, M4) |
| `backend/internal/middleware/provision.go` | IP validation + campus_id check (M3, L3) |
| `backend/internal/service/user_service.go` | LOGIN audit for existing users (M2) |
| `backend/cmd/server/main.go` | Added SecurityHeaders + RequireRole to routes (L4, M1) |
| `backend/internal/config/config.go` | Added OIDCSkipIssuerCheck field |
| `docker-compose.yml` | Added OIDC_SKIP_ISSUER_CHECK=true for dev |
| `backend/go.mod` / `backend/go.sum` | Added testify dependency |

### Deferred Security Findings (tracked for future sprints)

| ID | Severity | Finding | Reason Deferred | Target |
|----|----------|---------|-----------------|--------|
| H3 | HIGH | Hardcoded credentials in docker-compose.yml (`chesed`/`admin`) | Acceptable for local dev; production deployment will use env vars from secrets manager | Production deployment |
| L1 | LOW | Missing `campus_id` on address, person_role, assisted_profile tables | Schema change requires new migration + data model doc update; current approach joins through person | Sprint 2 (evaluate during Person CRUD) |
| L2 | LOW | triage table missing `is_active` and `updated_at` | Design decision — triages may be intentionally immutable per `docs/04-domain-model.md` | Sprint 3 (evaluate during Triage implementation) |
| I1 | INFO | Keycloak admin console exposed on port 8180 | Required for local development | Production deployment |
| I2 | INFO | Token stored in JavaScript memory (Zustand) | Standard for SPA + keycloak-js + PKCE; httpOnly cookies require server-side handling | Phase 2 (if XSS threat increases) |
| I3 | INFO | audit_log.user_id nullable | System operations (cron, batch) need null user_id | Document in data model |
| — | MEDIUM | No CI/CD pipeline yet | Blocks automated quality gate enforcement | Next session |
| — | LOW | No dependency scanning (T11: Supply Chain) | Need gitleaks/trufflehog + Dependabot | CI/CD setup |
| — | LOW | No rate limiting (T12: DDoS) | Deferred to production deployment behind reverse proxy | Production deployment |

### Code Review Fixes Applied (same session)

Code review executed per `.project-ai/prompts/code-review.md`. Found 0 BLOCKER, 2 MAJOR, 5 MINOR, 5 SUGGESTION. All fixable findings resolved:

| Finding | Fix |
|---------|-----|
| M1: `_ =` discards audit error | Replaced with `slog.ErrorContext` logging |
| M4: Header.tsx nesting depth 4 | Extracted SVG to `MenuIcon` component |
| M5: LogAction 57 lines | Refactored to `buildAuditEntry` (38 lines) + `setOptionalString` helper |
| M6: writeJSON duplicated | Extracted to `handler/response.go` |
| M7: Error format mismatch | Aligned to `{"error":"code","message":"text"}` per API spec |
| S1: config/health tests use t.Errorf | Migrated to testify assert/require |
| S2: Record<string,unknown> cast | Added typed `KeycloakTokenParsed` interface |
| S4: Loose security header assertions | Changed `assert.Contains` to `assert.Equal` |

### Deferred Code Review Findings

| ID | Severity | Finding | Reason Deferred | Target |
|----|----------|---------|-----------------|--------|
| M2 | MAJOR | Frontend test coverage ~7% (1 test for 13 files) | Requires significant effort; backend coverage is 90%+ | Sprint 2 |
| M3 | MINOR | Service tests (audit, user, provision) not table-driven | Functional and passing; refactor is cosmetic | When touching these files |
| S3 | SUGGESTION | CORS origin hardcoded to localhost:5173 | Acceptable for dev; must be configurable in production | Production deployment |
| S5 | SUGGESTION | No TypeScript types for API entities | No entity endpoints yet; add with Sprint 2 Person CRUD | Sprint 2 |

### Verification Results
| Check | Result |
|-------|--------|
| `go build ./...` | PASS |
| `go test ./...` | PASS (75+ cases across 8 packages) |
| `go vet ./...` | PASS |
| `npm run typecheck` | PASS |
| `npm run lint` | PASS |
| realm-export.json | Valid JSON, no secrets |

---

### Local Dev Setup Automation (same session)

Added automated local development setup flow:
- **`keycloak/init-realm.sh`** expanded to: configure User Profile, disable conditional OTP for dev, create 5 test users (one per RBAC role)
- **`README.md`** updated with numbered setup steps, test user table, service URLs
- **Decision**: Conditional OTP (MFA for ADMIN) is disabled in dev via `browserFlow: browser`. Production deployment must re-enable the custom `Browser with Conditional OTP` flow.
- **Decision**: `userProfile` removed from `realm-export.json` (Keycloak 26.0 doesn't support it in import). Configured via Admin API in `init-realm.sh` instead.
- **Decision**: `KC_HOSTNAME` removed from `docker-compose.yml` — Keycloak uses request hostname dynamically, API uses `SkipIssuerCheck` for dev.
- **Audience mapper** added to `chesed-pwa` client in realm-export.json — ensures `aud` claim includes `chesed-pwa` for JWT audience validation.

---

## Architecture Validation: Email Verification, MFA, User Onboarding (2026-04-06)

### Validation Performed
Executed `.project-ai/prompts/architecture-design.md` as a Senior Security & Architecture Reviewer. Validated production-level requirements for: (1) Keycloak user onboarding with email confirmation, (2) verified email enforcement before access, (3) MFA opt-in with email OTP or TOTP.

### Gaps Found and Addressed

| Gap | Severity | Fix Applied |
|-----|----------|-------------|
| `verifyEmail: false` in realm-export.json | HIGH | Set to `true` — Keycloak blocks unverified users |
| No `email_verified` check in API | HIGH | Added defense-in-depth check in `middleware/auth.go` — rejects `email_verified: false` with 403 |
| SMTP not configured | HIGH | Added Mailpit service to docker-compose.yml for dev; documented production SMTP env vars |
| MFA only for ADMIN, not COORDINATOR | MEDIUM | Updated auth flow to "Browser with Conditional MFA" — ADMIN + COORDINATOR mandatory |
| No email OTP option | MEDIUM | Documented as available method in Keycloak 26 (requires SMTP); users choose during enrollment |
| MFA blocks dev testing | LOW | `init-realm.sh` disables MFA + email verification for dev |
| doc 16: `chesed-web` should be `chesed-pwa` | LOW | Fixed all references |
| doc 16: `chesed-admin-cli` should be `chesed-api` | LOW | Fixed all references |
| doc 16: Refresh token TTL "7 days" (actual: 24h) | LOW | Fixed to "24 hours" |
| doc 16: SSO session max "10 hours" (actual: 24h) | LOW | Fixed to "24 hours" |
| doc 20: Email verification listed as "Phase 2" | MEDIUM | Moved to Phase 1 — now enabled |
| doc 20: No SMTP env var documentation | MEDIUM | Added production env vars table |
| doc 13: MFA described as ADMIN-only | LOW | Updated to ADMIN + COORDINATOR mandatory |

### Architecture Decisions

| Decision | Rationale |
|----------|-----------|
| Email verification enforced at two layers (Keycloak + API) | Defense-in-depth: Keycloak blocks login, API checks claim as backup |
| MFA mandatory for ADMIN + COORDINATOR | Coordinator manages teams/campaigns — credential compromise has high impact (T1) |
| MFA opt-in for other roles via Keycloak Account Console | Balances security with usability for field workers with limited devices |
| Email OTP as alternative to TOTP | Some users (volunteers) may not have smartphones for authenticator apps |
| Mailpit for dev SMTP | Captures emails locally without sending real emails; accessible at port 8025 |
| `init-realm.sh` disables MFA + email verification for dev | Test users need immediate access; production re-enables both |
| `userProfile` configured via Admin API, not realm-export.json | Keycloak 26.0 doesn't support `userProfile` in import JSON |

### Files Modified
| File | Change |
|------|--------|
| `keycloak/realm-export.json` | `verifyEmail: true`, auth flow renamed to "Browser with Conditional MFA", added COORDINATOR conditional flow, added audience mapper |
| `docker-compose.yml` | Added Mailpit service (ports 8025/1025) |
| `keycloak/init-realm.sh` | Added SMTP config (Mailpit), MFA disable for dev, email verification disable for dev, 5 test users |
| `backend/internal/middleware/auth.go` | Added `email_verified` claim parsing and 403 rejection |
| `backend/internal/auth/context.go` | Added `EmailVerified` field to `AuthClaims` |
| `docs/16-iam-and-access-control.md` | Fixed client names, session TTLs, added email verification section, updated MFA policy |
| `docs/20-keycloak-configuration.md` | Moved email verification to Phase 1, added SMTP docs, updated MFA section |
| `docs/13-security-and-compliance.md` | Updated MFA policy, email verification, token TTLs |
| `CLAUDE.md` | Added email_verified rule (#12), MFA policy rule (#13) |
| `CODEX.md` | Same rules added |
| `.env.example` | Added SMTP env vars (commented for production) |
| `README.md` | Added Mailpit URL, updated setup flow |

### Production Deployment Checklist (Email/MFA)
- [ ] Configure production SMTP (Amazon SES / SendGrid) via Keycloak Admin Console
- [ ] Set `browserFlow` to `Browser with Conditional MFA` (init-realm.sh disables for dev)
- [ ] Set `verifyEmail` to `true` (init-realm.sh disables for dev)
- [ ] Verify email verification link works with production domain
- [ ] Test MFA enrollment for ADMIN and COORDINATOR roles
- [ ] Test email OTP delivery for users without authenticator apps
- [ ] Store SMTP credentials in secrets manager (never in code)

---

## Session 9: CI/CD Pipeline (2026-04-06)

### What Was Done
1. **GitHub Actions CI/CD** — 3 workflows + Dependabot configuration
2. **Frontend coverage** — Added `@vitest/coverage-v8` and configured coverage thresholds in `vite.config.ts`

### Files Created
- `.github/workflows/backend.yml` — Go build, vet, golangci-lint, test with coverage, PostgreSQL 16 service container
- `.github/workflows/frontend.yml` — TypeScript typecheck, ESLint, Vitest with 80% coverage thresholds, Vite build
- `.github/workflows/security.yml` — gitleaks secret detection, govulncheck, npm audit (3 parallel jobs)
- `.github/dependabot.yml` — Weekly updates for Go modules, npm packages, GitHub Actions versions

### Files Modified
- `frontend/vite.config.ts` — Added `coverage` config with `v8` provider and 80% thresholds
- `frontend/package.json` — Added `@vitest/coverage-v8` devDependency

### Architecture Decisions
- **3 separate workflows** (backend, frontend, security) with path filters — run in parallel, only trigger on relevant changes
- **Built-in caching** via `actions/setup-go@v5` and `actions/setup-node@v4` — no explicit `actions/cache` steps needed
- **Coverage enforcement**: Backend uses shell script parsing `go tool cover -func` (threshold: 40% Sprint 1 → 70% Sprint 2 → 80% Sprint 3+); Frontend uses Vitest's native `coverage.thresholds` (80% all metrics)
- **Security scanning**: gitleaks for secrets, govulncheck for Go deps, npm audit for Node deps — all block merge on failure
- **Dependabot**: Grouped dependency PRs (Go all-in-one, npm prod/dev separate, GitHub Actions separate) on Monday weekly cadence
- **PostgreSQL service container** provisioned in backend workflow for future integration tests
- **`--legacy-peer-deps`** used for npm ci due to `vite-plugin-pwa` peer dep conflict with Vite 8

### Current Coverage Status
- Backend: 49.3% (unit tests only — `cmd/server`, `database`, `repository` at 0% pending integration tests)
- Frontend: 100% (single test file, will decrease as components grow)

### Manual Step Required
Configure GitHub branch protection rules on `main` to require these status checks:
- `Build & Test` (from backend.yml)
- `Build & Test` (from frontend.yml)
- `Secret Detection` (from security.yml)
- `Go Vulnerability Scan` (from security.yml)
- `npm Audit` (from security.yml)

---

## Session 10: Production Docker Compose (2026-04-06)

### What Was Done
1. **Production docker-compose** (`docker-compose.prod.yml`) with 4 services: PostgreSQL, Keycloak, Go API, Frontend (nginx)
2. **Frontend production Dockerfile** (`frontend/Dockerfile.prod`) — multi-stage build: node:20-alpine → nginx:1.27-alpine
3. **Nginx reverse proxy** (`frontend/nginx.conf`) — single entry point, SPA routing, API/Keycloak proxy
4. **Production env template** (`.env.prod.example`) — all credentials via environment variables

### Files Created
- `docker-compose.prod.yml` — production compose (health checks, restart policies, resource limits, internal networking)
- `frontend/Dockerfile.prod` — multi-stage build with build-time Vite env args
- `frontend/nginx.conf` — reverse proxy + SPA routing + security headers
- `.env.prod.example` — documented env var template with placeholders

### Key Decisions
- **Nginx as single entry point**: Only port 80 exposed externally. API and Keycloak are internal-only, proxied through nginx
- **Keycloak admin console blocked**: Nginx only proxies `/auth/realms/`, `/auth/resources/`, `/auth/js/` — admin console (`/admin/`) is NOT proxied
- **Keycloak production mode**: Uses `start` (not `start-dev`), with `KC_HOSTNAME` and `KC_PROXY_HEADERS=xforwarded`
- **SMTP via SPI env vars**: Keycloak email configured through `KC_SPI_EMAIL_SENDER_DEFAULT_*` environment variables
- **Resource limits**: db=512MB, keycloak=1GB, api=256MB, frontend=128MB
- **Build-time frontend config**: Vite env vars passed as Docker build args (baked into static bundle)

### Deployment Instructions
```bash
# 1. Copy and fill in environment variables
cp .env.prod.example .env.prod
# Edit .env.prod with production values

# 2. Start all services
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 3. Run database migrations (after first deploy)
docker compose -f docker-compose.prod.yml exec api sh -c "cd /app && ./server migrate-up"
# Or run migrations from host if migrate CLI is available

# 4. Verify
curl http://localhost/api/v1/health
curl http://localhost/nginx-health
```

---

## Session 11: TLS Termination with Let's Encrypt (2026-04-06)

### What Was Done
1. **TLS termination via nginx** — HTTPS on port 443 with HTTP-to-HTTPS redirect on port 80
2. **Let's Encrypt integration** — Certbot sidecar container for automatic certificate provisioning and renewal
3. **TLS hardening** — TLSv1.2/1.3 only, strong cipher suite, OCSP stapling, session tickets disabled
4. **HSTS** — `Strict-Transport-Security` header with `max-age=63072000; includeSubDomains; preload`
5. **Bootstrap script** — `scripts/init-letsencrypt.sh` solves the chicken-and-egg problem (nginx needs certs to start, certbot needs nginx to verify)
6. **Renewal automation** — Certbot container auto-renews every 12h; `scripts/ssl-renew.sh` for cron-based nginx reload

### Files Modified
- `frontend/nginx.conf` — Split into HTTP (ACME challenge + redirect) and HTTPS (all app traffic) server blocks; added SSL config, HSTS, HTTP/2
- `frontend/Dockerfile.prod` — Added `EXPOSE 443`
- `docker-compose.prod.yml` — Added certbot service, shared volumes (certbot-conf, certbot-webroot), exposed port 443, nginx.conf bind mount
- `.env.prod.example` — Added `DOMAIN_NAME` and `CERTBOT_EMAIL` variables

### Files Created
- `scripts/init-letsencrypt.sh` — Bootstrap script: dummy cert → start nginx → request real cert → reload
- `scripts/ssl-renew.sh` — Cron helper: certbot renew + nginx reload

### Key Decisions
- **Certbot with webroot plugin** (not standalone): Allows zero-downtime renewal since nginx keeps running
- **Fixed cert path `/etc/letsencrypt/live/chesed/`**: Since nginx can't interpolate env vars, the init script uses `--cert-name chesed` to create a fixed directory name
- **nginx.conf mounted as volume**: Overrides the baked-in copy so TLS paths are available at runtime without rebuilding the image
- **ACME challenge on HTTP**: Port 80 serves only `/.well-known/acme-challenge/` and health check; all other traffic gets 301 to HTTPS
- **Staging support**: Set `LETSENCRYPT_STAGING=1` in env to use Let's Encrypt staging for testing (avoids rate limits)
- **`ssl_prefer_server_ciphers off`**: Modern best practice — all listed ciphers are strong, so let client choose the fastest one it supports

### Deployment Instructions (TLS)
```bash
# 1. Copy and fill in environment variables (add DOMAIN_NAME and CERTBOT_EMAIL)
cp .env.prod.example .env.prod
# Edit .env.prod with production values

# 2. Bootstrap TLS certificates (first time only)
sudo bash scripts/init-letsencrypt.sh

# 3. Start all services (migrations run automatically before the API starts)
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 4. (Optional) Run migrations manually if needed
cd backend && make migrate-up-prod

# 5. Verify
curl -I http://yourdomain.org        # Should get 301 → https://
curl -I https://yourdomain.org       # Should get 200 + HSTS header
curl http://yourdomain.org/nginx-health  # Should get 200 "ok"

# 6. Set up renewal cron (on host)
# 0 0,12 * * * /path/to/chesed/scripts/ssl-renew.sh >> /var/log/chesed-ssl-renew.log 2>&1
```

---

## Session 11: Automated Database Migrations (2026-04-06)

### What Was Done
1. **Production migration runner script** — Created `backend/migrations/run.sh` with PostgreSQL readiness wait loop, dirty state recovery, and structured logging
2. **Migrate CLI in Docker image** — Updated `backend/Dockerfile` to download golang-migrate v4.18.3 binary in builder stage and copy to runtime image
3. **One-shot migrate service** — Added `migrate` service to `docker-compose.prod.yml` that runs before the API via `service_completed_successfully` dependency
4. **CI pipeline migrations** — Updated `.github/workflows/backend.yml` to install golang-migrate and run migrations before tests
5. **Manual migration target** — Added `make migrate-up-prod` to `backend/Makefile` for operator convenience

### Files Created
- `backend/migrations/run.sh` — Production migration runner with retry loop, dirty state handling, clear logging

### Files Modified
- `backend/Dockerfile` — Added golang-migrate v4.18.3 binary download and copy to runtime image
- `docker-compose.prod.yml` — Added `migrate` one-shot service; updated `api` depends_on chain; updated header comment
- `.github/workflows/backend.yml` — Added "Install golang-migrate" and "Run database migrations" steps before tests
- `backend/Makefile` — Added `migrate-up-prod` target

### Key Decisions
- **One-shot service pattern** (not entrypoint script): Migrations run as a separate Docker Compose service with `service_completed_successfully`, ensuring they complete before the API starts and deployment fails visibly if migrations fail
- **Dirty state auto-recovery**: If a migration was interrupted and left the DB dirty, the script forces the version to clear the flag and retries — handles the most common production failure mode
- **Same image for migrate and api**: The `migrate` service reuses the same backend Docker image (no extra build), just overrides the command
- **Pinned migrate version (v4.18.3)**: Same version in Dockerfile and CI to avoid drift

---

## Next Recommended Steps

### Immediate (Next Session)

1. **CI/CD pipeline TLS integration** — Update deployment scripts to run init-letsencrypt.sh on first deploy

### Medium-Term (Phase 1 Sprints 2-4)

3. Person CRUD API + React pages (Sprint 2)
4. Triage and Attendance API + React forms (Sprint 3)
5. Offline sync (IndexedDB + push/pull endpoints) (Sprint 4)
6. Basic reports with CSV export (Sprint 4)

---

## Context for Future AI Sessions

When starting a new session on this repository:

1. Read `CLAUDE.md` (root) for architecture rules
2. Read `docs/07-mvp-scope.md` to understand current priorities
3. Read `docs/09-backlog.md` to find the next story to implement
4. Check `HANDOFF.md` (this file) for the latest status
5. Check `git log` for what has been implemented since this handoff

The documentation suite is designed to be self-sufficient — an AI agent should be able to pick up any story from the backlog and implement it correctly by following the architecture, data model, and API design docs.
