# HANDOFF.md - Session History and Next Steps

## Last Updated
2026-04-03 (Session 5)

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

## Open Questions

These need stakeholder input (but have documented defaults that allow implementation to proceed):

1. **International document types**: MVP supports CPF (Brazil) only. What types for USA/Europe? (Phase 3)
2. **Consent form content**: Legal team must provide template text. (Phase 2 prerequisite)
3. **Donation receipt format**: Legal requirements per country? (Phase 3)
4. **Hosting budget**: Confirmed budget? (Estimates: $10-17/month MVP with Keycloak, $55-70/month production)
5. **Volunteer testing**: Can real volunteers test the MVP during development?

---

## Next Recommended Steps

### Immediate (Next Session)

1. **Create the Go project skeleton** (`backend/`)
   - `cmd/server/main.go` with chi router
   - `internal/` package structure (config, domain, handler, service, repository, middleware)
   - Health check endpoint (`GET /api/v1/health`)
   - Makefile with standard commands
   - Docker Compose for Go + PostgreSQL + Keycloak

2. **Set up Keycloak**
   - Keycloak container in Docker Compose with `chesed` realm
   - Configure OIDC client for React PWA (public client, PKCE)
   - Configure OIDC client for Go API (confidential, for Admin API access)
   - Add realm roles (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER)
   - Add custom protocol mappers for `campus_id` and `person_id`
   - Export realm configuration to `keycloak/realm-export.json`

3. **Create the React project skeleton** (`frontend/`)
   - Vite + React + TypeScript
   - Tailwind CSS configured
   - PWA manifest and service worker shell
   - keycloak-js adapter integration
   - Base layout component (responsive sidebar + header)
   - ESLint + Prettier configured

4. **Write the first database migrations**
   - `campus` table
   - `person` and `address` tables
   - `app_user` table (with `keycloak_subject_id`, no `password_hash`)
   - `audit_log` table
   - `service_type` table

5. **Set up CI/CD**
   - GitHub Actions: Go test + lint, React build + lint
   - PostgreSQL service container for integration tests
   - Trivy scanning for Keycloak container image

### Short-Term (Phase 1 Sprint 1)

6. Implement Go OIDC middleware using `coreos/go-oidc` (validate Keycloak tokens via JWKS)
7. Implement local user auto-provisioning (first login creates `app_user` from Keycloak `sub` claim)
8. Implement RBAC middleware (roles from Keycloak token claims)
9. Implement audit logging middleware
10. Build React OIDC integration (keycloak-js adapter, protected routes, auth context)
11. Build React layout shell
12. Configure MFA for ADMIN role in Keycloak

### Medium-Term (Phase 1 Sprints 2-4)

10. Person CRUD API + React pages
11. Triage and Attendance API + React forms
12. Offline sync (IndexedDB + push/pull endpoints)
13. Basic reports with CSV export

---

## Context for Future AI Sessions

When starting a new session on this repository:

1. Read `CLAUDE.md` (root) for architecture rules
2. Read `docs/07-mvp-scope.md` to understand current priorities
3. Read `docs/09-backlog.md` to find the next story to implement
4. Check `HANDOFF.md` (this file) for the latest status
5. Check `git log` for what has been implemented since this handoff

The documentation suite is designed to be self-sufficient — an AI agent should be able to pick up any story from the backlog and implement it correctly by following the architecture, data model, and API design docs.
