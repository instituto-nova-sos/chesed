# Sprint 4 Security Review (Task 4.9)

**Date**: 2026-07-01
**Scope**: Defensive OWASP Top 10 review of the Chesed backend (`backend/`) and
frontend (`frontend/`) against the project threat model
(`docs/18-threat-model.md`) and the CLAUDE.md security rules.
**Method**: Read-only review of the production code paths, verified with a real
PostgreSQL integration test for the one confirmed HIGH finding.

This review covers the Phase 1 surface: auth, person, triage, attendance, sync
push/pull, and reports. It is a point-in-time audit; re-run when new endpoints
or data flows are added.

---

## Summary

| # | Finding | OWASP | Severity | Status |
|---|---------|-------|----------|--------|
| 1 | `GET /persons/check-duplicate` leaked cross-campus person data | A01 | **HIGH** | **Fixed** (this sprint) |
| 2 | Email (PII) written to logs on the email-not-verified path | A09 | LOW | **Fixed** (this sprint) |
| 3 | Sync push does not verify the referenced `person_id`/`triage_id` belongs to the caller's campus | A01 | MEDIUM | Deferred (hardening; TODO below) |
| 4 | RBAC 403 denials are logged but not written to `audit_log` | A09 | INFO | Deferred (hardening; TODO below) |

No CRITICAL findings. All A-rated controls below were verified present.

---

## A01 — Broken Access Control / Multi-Tenancy

**Finding 1 (HIGH, FIXED): `check-duplicate` cross-campus leak.**
`PersonRepository.CheckDuplicate` received the caller's `campusID` but never bound
it in the SQL `WHERE` clause, and joined `campus` to return the matched person's
campus name. A caller in campus A could probe any document and learn the
existence, full name, document number, and campus of the matching person in
**any** campus — violating CLAUDE.md rule #4 and threat model T3 (Campus
Isolation Breach).

- Reproduced with a real-Postgres integration test
  (`backend/internal/integration/person_test.go` → `TestCheckDuplicate_CampusScoped`),
  which failed against the unfixed query (returned `Campus B Person / Second
  Campus` to a campus-A caller).
- **Fix**: added `AND p.campus_id = $3` and bound `campusID`; removed the now-
  redundant `campus` field from the query, the scan, and the `DuplicateMatch`
  domain struct; updated `docs/11-api-design.md`. The test now passes, and a
  companion test (`TestCheckDuplicate_SameCampusMatch`) proves same-campus
  duplicates are still detected.
- Note: `(document_type, document_number)` carries a **global** unique constraint
  (`uq_person_document`, migration 000002), so at most one person holds a given
  document system-wide. Campus scoping therefore does not weaken duplicate
  prevention on insert (the DB still rejects a true global duplicate); it only
  closes the cross-campus information-disclosure channel.

**Verified present (A-rated):**
- Sync pull is campus-scoped in SQL and covered by a multi-tenant integration
  test (`sync_test.go` → `TestSyncPull_CampusScopedDelta`: campus B records must
  not leak into a campus A pull).
- Sync push stamps the server-resolved `campusID` on every created record and
  rejects a `uuid.Nil` campus.
- Reports/export read the campus from context and filter every query by
  `campus_id`; routes are COORDINATOR/ADMIN-only.
- Person/triage/attendance list and single-record reads all filter `campus_id`.
- Cross-campus overrides are restricted to the ADMIN role (`auth.ResolveCampusID`).

**Finding 3 (MEDIUM, DEFERRED)**: on sync push, a triage/attendance stamps its
own `campus_id` from the token but does not verify that the referenced
`person_id` (or `triage_id`) belongs to that campus. FK integrity holds and the
record's own campus is correct, so this is defense-in-depth, not a live bypass.
Implementing it requires extending `SyncPersonRepository`/`SyncTriageRepository`
with a campus-scoped `FindByID` and updating the unit-test mocks — see TODO below.

---

## A02 — Cryptographic Failures

- Secrets are read from environment variables (`internal/config`); no secrets are
  hardcoded in Go/TS source or committed compose files (prod compose uses
  `${...}` references). Dev/e2e compose credentials are non-prod placeholders.
- Offline-at-rest encryption (S05.1, this sprint) adds an AES-GCM field-encryption
  **capability** for the Dexie offline layer with Web Crypto feature-detection and
  graceful plaintext degradation (`frontend/src/offline/encryption.ts`). At-rest
  encryption activates once the store wraps the record `data` field through
  `encryptPayload`/`decryptPayload`; that wiring is a documented follow-up (Dexie's
  synchronous hooks cannot await AES-GCM, so it belongs in the async read/write
  helpers). Until then the module is tested but not yet on the live write path.
- TLS termination is assumed at the reverse proxy/Cloudflare edge (infra item).

---

## A03 — Injection

Verified: all SQL uses pgx positional placeholders. The dynamic query builders
(`buildPersonListQuery`, `buildTriageWhere`, `buildAttendanceWhere`) use
`fmt.Sprintf` only to compute placeholder indexes (`$%d`), never to interpolate
user data — values always flow through `pool.Query(ctx, q, args...)`. No
string-concatenated user input into SQL. Frontend has no `dangerouslySetInnerHTML`.

---

## A05 — Security Misconfiguration

- CORS is an allowlist (`middleware/cors.go`), not a wildcard; the origin is
  supplied at wiring time. **Infra item**: supply the production origin at
  `cmd/server/main.go` for deployment.
- Security headers are set globally (`X-Content-Type-Options: nosniff`,
  `X-Frame-Options: DENY`, `Referrer-Policy`, `Permissions-Policy`,
  `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`,
  `Cache-Control: no-store`).
- **Infra items**: HSTS + the SPA's own CSP are expected at the edge/asset server;
  `OIDC_SKIP_ISSUER_CHECK` must be `false` in production.

---

## A07 — Identification and Authentication Failures

Verified (strong): JWT signature validated via JWKS through `coreos/go-oidc`;
`email_verified` enforced with HTTP 403; expired/invalid/missing tokens rejected
401; RBAC derived from token claims (not request body) and applied on every
protected route. The compensating auth integration test
(`auth_middleware_test.go`) asserts valid→200, expired→401, unverified→403, bad
signature→401, missing header→401, and that rejected requests never reach the
downstream handler.

Frontend: keycloak-js uses PKCE S256; tokens live in the adapter's in-memory
object mirrored into Zustand state, not in `localStorage`/`sessionStorage`.

---

## A09 — Security Logging and Monitoring Failures

**Finding 2 (LOW, FIXED)**: the email-not-verified 403 path logged
`"email", claims.Email`. Email is PII (rule #7). **Fix**: dropped the email
attribute; the log keeps only the opaque `subject`. This was the only PII log
call in middleware/handlers. Other logs carry ID-based context (UUIDs) only.

**Verified present**: every mutating service path calls `auditSvc.LogAction`
(person/triage/attendance create/update/transition, campus create/update, sync
creates, onboarding, provisioning). Audit entries capture user, campus, IP,
user-agent, and success.

**Finding 4 (INFO, DEFERRED)**: RBAC 403 denials are emitted to application logs
(`middleware/rbac.go`) but not to `audit_log`. Threat model T2 wants 403s in the
audit trail. Wiring an `AuditService` into `RequireRole` touches all 8 call sites
in `main.go`; deferred as a follow-up rather than forced — see TODO below.

---

## A10 / Other

- Rate limiting is intentionally delegated to the Cloudflare/reverse-proxy edge
  (threat model T1/T5/T12) — no in-app middleware. Verify at the edge.
- PostgreSQL Row-Level Security is a documented **Phase 3** defense-in-depth
  layer (threat model T3); its absence in Phase 1 is expected, not a defect.

---

## Deferred hardening — follow-up TODOs

These are tracked for a later sprint; none is a live vulnerability.

1. **Sync reference campus check (Finding 3, MEDIUM)** — extend
   `SyncPersonRepository`/`SyncTriageRepository` with a campus-scoped `FindByID`
   and, in `buildSyncTriage`/`buildSyncAttendance`, reject a push whose referenced
   `person_id`/`triage_id` is not in the caller's campus. Requires updating the
   sync unit-test mocks. Add an integration test for the cross-campus-reference
   rejection.
2. **Audit 403 denials (Finding 4, INFO)** — inject `AuditService` into
   `RequireRole` (or a thin wrapper) so denials are written to `audit_log`, not
   just application logs.

## Infra / configuration checklist (out of code scope)

- [ ] Production CORS origin supplied at `cmd/server/main.go`.
- [ ] HSTS header and SPA CSP set at the reverse proxy / asset host.
- [ ] `OIDC_SKIP_ISSUER_CHECK` unset/false in production.
- [ ] Edge rate limiting configured (Cloudflare).
- [ ] Dev/e2e compose credentials never reused in any exposed environment.
