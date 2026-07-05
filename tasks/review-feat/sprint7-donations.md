# Critical Review — E09 Donation Tracking (branch: feat/sprint7-donations)

Reviewer: reviewer (autonomous critical-review gate)
Scope: `git diff main...HEAD` — 4 commits (RED→GREEN per story), 27 files, +2626 lines.
Surface: backend donation domain/repository/service/handler + migration 000026 +
main.go/harness wiring; frontend types/api/hooks/3 pages + App/Sidebar/barrel;
backend unit+integration tests, frontend MSW integration, Playwright smoke; docs
(backlog E09 GWT + API bodies in docs/11).

## Quality Gate: PASS

### Conditions
| Condition | Status | Notes |
|-----------|--------|-------|
| Bugs (reliability) | PASS | 0. Every error handled and wrapped with `%w`; `23505`→`ErrDuplicate`, `pgx.ErrNoRows`→`ErrNotFound`; `rows.Err()` checked in `List`; stale-response guards in `useDonation`/`useDonations`. |
| Vulnerabilities (security) | PASS | 0. All SQL parameterized; campus_id enforced in every query and in Update's WHERE; T3 non-disclosure honored for donor and campaign refs. |
| Security hotspots reviewed | PASS | 100%. Campus scoping, RBAC, audit, T3 disclosure, PII-in-logs, secrets — all traced to code (see Issues). |
| Coverage on new code | PASS | ≥80%. Service unit tests cover create/update/get/list happy + forbidden + validation + donor/campaign in-and-out-of-campus; integration covers CRUD w/ DB assertions, campus boundary, RBAC 403+audit, campaign link, anonymous donor; frontend MSW covers list/filter/error/create(FIN+GOODS)/update/detail/404; E2E smoke asserts Postgres row. |
| Duplication on new code | PASS | ≤3%. `resolveCampaignRef` and `parseOptionalUUID`/`parseOptionalTime` reused from existing service package (not re-implemented); donor resolver mirrors the campaign resolver deliberately, sharing the T3 contract. |
| Maintainability rating | PASS | A. All files within thresholds (Go max 318≤400; React/TS max 269≤300). Functions decomposed (`buildDonationFromInput`, `applyDonationUpdate`, `validateDonationBusinessRules`, extracted React subcomponents). |
| Reliability rating | PASS | A. Immutable fields (`campus_id`, `registered_by`, `created_at`) carried forward on update and never in the UPDATE SET; audit failure logged, never fails the request. |
| Security rating | PASS | A. Keycloak-delegated auth, RBAC middleware on every route, campus scoping, audit on every mutation, generic validation errors. |
| High severity issues | PASS | 0. |

### Clean Code Assessment
| Category | Status | Findings |
|----------|--------|----------|
| Consistency | PASS | Mirrors sibling aggregates (campaign/consent): same service/repository/handler layering, same `CampaignRefRepository`/person-repo lookup pattern, same `writeXError` sentinel mapping, same `//go:build integration` harness convention. main.go and harness_test.go register identical route/role tuples. |
| Intentionality | PASS | Names reveal purpose (`validateDonationBusinessRules`, `resolveDonor`, `parseDonationDate`). Comments explain *why* (T3 non-disclosure, immutable fields, online-only hook, receipt_number Phase-3 deferral). No dead code. |
| Adaptability | PASS | Dependencies point inward: service depends on repository interfaces (`DonationRepository`, `DonationPersonRepository`, `CampaignRefRepository`); handler depends on service; repository depends on pgx+domain; domain has zero deps. Services never import pgx. |
| Responsibility | PASS | Each function does one thing; build/apply/validate/resolve split cleanly. Handler parses+delegates only. Repository is pure SQL+scan. |

### Issues

#### backend/internal/repository/donation_repository.go
- No BLOCKER/MAJOR/MINOR. Create/FindByID/List/Update all campus-scoped; Update WHERE includes `campus_id` so cross-campus update yields `ErrNotFound` (verified by `TestDonation_CampusBoundary`). `buildDonationWhere` uses positional args, no injection. `receipt_number` is SELECTed only, never in INSERT — stays NULL (Phase 3), no UNIQUE collision this sprint.

#### backend/internal/service/donation_service.go
- No BLOCKER/MAJOR/MINOR. CREATE and UPDATE both audit; campus stamped from `claims.CampusID`; `registered_by` from token subject; `resolveDonor` and `resolveCampaignRef` both collapse foreign-campus `ErrNotFound` into generic `ErrValidation` (T3). Conditional validation server-side. `audit` logs on failure without failing the request.

#### backend/internal/handler/donation.go
- No BLOCKER/MAJOR/MINOR. Sentinel→HTTP mapping correct (404/403/409/400/500); validation message kept generic ("invalid reference or field values") so foreign existence is never disclosed. `slog.ErrorContext` logs only action + `err.Error()` — no PII.

#### backend/cmd/server/main.go + backend/internal/integration/harness_test.go
- No BLOCKER/MAJOR. RBAC tuples are byte-faithful between production and harness: `POST`/`PUT` = `SECRETARY,PROFESSIONAL,COORDINATOR,ADMIN`; `GET`(list)/`GET`(detail) = `COORDINATOR,ADMIN`. Matches docs/11 rows 625–629. No drift.

#### backend/migrations/000026_create_donation.*.sql
- No BLOCKER/MAJOR. up+down present; UUID PK, `campus_id NOT NULL REFERENCES campus`, CHECK on donation_type, DECIMAL(12,2) amount, `receipt_number VARCHAR(50) UNIQUE`, indexes on campus/campaign/donor/date. down drops the table.

#### frontend (types/api/hooks/pages)
- No BLOCKER/MAJOR. No `any` (strict); `unknown` narrowed in catch. Query-string serialization matches API contract; POST omits `amount` for GOODS; conditional client validation mirrors server (defense in depth). Pages decomposed; no JSX block > 80 lines.

- SUGGESTION — `frontend/src/components/layout/Sidebar.tsx:14` and `frontend/src/App.tsx:179-182`
  The "Doações" nav link and the `/donations` list/detail routes are rendered for
  all authenticated roles, but the backend gates GET `/donations` and
  `/donations/:id` at Coordinator+. A SECRETARY/VOLUNTEER who opens "Doações"
  reaches a page that can only ever render the 403 error state. This is
  backend-authoritative (no security exposure — CLAUDE.md precedent #4; the 403 is
  enforced server-side and surfaced via the hook's `error` state) and it follows
  the pre-existing unfiltered-Sidebar pattern (`/campuses`, `/reports` behave the
  same). Unlike campaigns (list is `allRoles`), the donation list is Coordinator+,
  so the dead-end is more visible here. Non-blocking UX polish: consider
  role-gating the nav item / redirecting below-Coordinator roles. Recommend
  tracking as a follow-up issue rather than fixing in this PR.

### Complexity Report
No function or file exceeds thresholds.
- Go files: donation_service.go 318, donation_repository.go 179, handler/donation.go 145, domain/donation.go 64 — all ≤400. Longest function (`buildDonationFromInput`/`applyDonationUpdate`) ~35 lines ≤40; nesting ≤3; params ≤5; returns ≤3.
- React/TS files: DonationFormPage.tsx 269, DonationListPage.tsx 136, DonationDetailPage.tsx 68, useDonations.ts 59, useDonation.ts 40, donationLabels.ts 17 — all ≤300. Form page decomposed into `FinancialFields`/`CampaignSelect`/`DonationFields`; no JSX block > 80 lines; cognitive ≤15.

### Test Integrity Assessment
- Integration tests are REAL: assert Postgres state (campus_id, registered_by, donation_type, item_description, campaign_id, NULL donor), campus boundary (list omission + 404 on detail/update), RBAC (VOLUNTEER create → 403 with zero rows + audited ACCESS_DENIED; SECRETARY list → 403), cross-campus campaign rejection with non-disclosure assertion and no-row-persisted check.
- RED→GREEN order honest: commit `3df518c` (RED) adds only `*_test.go` files; `7411e47` (GREEN) adds production + docs. UI story: `f04ec49` (RED) precedes `1a07b39` (GREEN).
- E2E smoke asserts real Postgres: `SELECT campus_id, donation_type, amount FROM donation WHERE notes = $1`, rowCount==1, campus match, amount==175.5.
- No hollow/rubber-stamp tests found; assertions verify intended correct behavior, not merely current output.

### Verdict: APPROVE
All quality-gate conditions PASS, all four clean-code categories PASS, and there
are no BLOCKER, MAJOR, or MINOR issues. The single finding is a non-blocking
SUGGESTION (client-side nav gating for below-Coordinator roles), which is
backend-authoritative, consistent with the existing unfiltered-Sidebar pattern,
and recommended for follow-up tracking rather than a change in this PR. The E09
Donation Tracking feature meets the New Code Quality Gate and is READY-FOR-PR.
