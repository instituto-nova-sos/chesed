# Critical Review — feat/sprint10-lgpd-compliance

**Scope**: `git diff main...HEAD` — Sprint 10 (LGPD & Compliance): consent-driven
anonymization (S11.3), compliance report + CSV (S11.4), donation receipt PDF
(S11.5), audit log viewer (S11.6), retention sweep (S11.7). 71 files, ~4.9k insertions.

**Method**: three independent adversarial reviewers over the anonymization/retention,
audit/compliance, and receipt/frontend surfaces, each verifying findings against the
code and tests. BLOCKER/MAJOR findings were fixed and re-verified before this verdict.

---

## Quality Gate

- **Bugs / correctness**: 1 MAJOR found and **fixed** (retention "last activity"); 0 remaining.
- **Vulnerabilities**: 0. Audit campus-scoping, RLS interaction, SQL-injection surface, and RBAC all verified safe.
- **Security hotspots reviewed**: 100%. PII-in-audit checked on every mutation path (clean — only IDs/timestamps recorded); `audit_log` immutability preserved (read-only viewer, `Create` connection semantics unchanged); Keycloak-only auth untouched.
- **Coverage**: every new endpoint has backend integration tests (real Postgres + MinIO) and every new API-client surface has a frontend MSW integration test; new-code unit coverage meets the layer thresholds. Backend + frontend suites green.
- **Duplication**: within threshold — reused `parseDateRange`/`parseReportRange`, the `base`/Querier RLS pattern, `PersonAnonymizer` across consent + retention, and the existing CSV/blob download helper.
- **Ratings**: Maintainability A, Reliability A, Security A.

## Clean Code

- **Consistency**: new handlers/services/repositories follow the established layering (handler→service→repository→domain), error mapping (`writeReportError`/`writeDonationError`), audit-at-service-layer, and campus-from-context conventions verbatim.
- **Intentionality**: names reveal purpose; comments explain the non-obvious (anonymization sentinel width, audit non-RLS scoping, storage-first ordering, retention "last activity").
- **Adaptability**: dependencies point inward; the audit read path was added additively without disturbing the write path; the receipt renderer and object storage are injected behind interfaces.
- **Responsibility**: functions do one thing and stay within the complexity/length thresholds.

## Complexity

Within thresholds (Go ≤40 lines / cyclomatic 10 / nesting 3; React component ≤300 / JSX ≤80). PDF rendering is split into header/body/footer helpers; the audit/compliance dynamic-WHERE builders are flat and parameterized.

---

## Issues by Severity

### BLOCKER
None.

### MAJOR — fixed
- **Retention keyed expiry off `person.updated_at`, not real last activity.** A subject with recent triage/attendance/donation but an old profile row would be irreversibly anonymized while still being assisted — an erasure-of-active-subject harm.
  **Fix (commit on this branch)**: `ListExpiredPersonIDs` now derives last activity from `GREATEST(person.updated_at, max triage/attendance/donation activity)`; a new integration test (`TestRetention_KeepsSubjectsWithRecentActivity`) proves a recent donation keeps the subject intact. Docs updated.

### LOW — fixed
- **Concurrent first-call to the receipt endpoint could return a spurious 409** to the race loser. **Fixed**: `ErrDuplicate` on stamp is now treated as already-issued and re-presigns the (already-written, deterministic-key) object. Unit test updated to assert the idempotent success path.

### LOW / informational — acknowledged, no change (by design)
- Compliance report has no date-span cap (intentional — multi-year retention windows; Coordinator+ gated; bounded by campus consent volume; documented).
- Compliance `active/revoked` counts are period-scoped while subject/document counts are point-in-time — a defensible reading of "posture for a period"; documented, internally consistent.
- `user_email` join couples to `keycloak_subject_id` being a canonical lowercase UUID (true for Keycloak today; LEFT JOIN degrades to NULL email, never a wrong email, if a future IdP differs).

### SUGGESTION (non-blocking, deferred)
- An end-to-end rollback integration test (forced Anonymize failure → revoke rolls back). The mechanism is proven at the unit level and fully traced through `CampusTx` (rollback on ≥400); an integration variant would harden the regression net.
- A collision-proof anonymization sentinel (current 25-hex truncation is a near-certainty, not a guarantee, of uniqueness).
- Add `CampusTx` to the integration harness router so the compliance repo's in-RLS-tx path is regression-covered (today only the explicit `WHERE campus_id` is exercised).

---

## Positive Practices

- Erasure is in-place (PII overwritten, row kept) so referential integrity and aggregate reporting survive; the audit trail is PII-free by construction (only IDs + `anonymized_at`), satisfying CLAUDE MUST-NOT #7.
- Atomicity of revoke+anonymize leans on the existing per-request `CampusTx` (rollback on ≥400/panic) instead of a bespoke transaction — no new code path, fail-closed.
- The audit viewer keeps `audit_log` append-only and non-RLS, applying strict campus scoping in SQL from the token (never a query param); integration test seeds foreign-campus and NULL-campus rows and proves only own-campus rows return.
- Receipt issuance is storage-first with orphan-key logging and idempotent re-presign; the receipt number fits `VARCHAR(50)` and the DB-stamped and PDF-rendered numbers are the same value.
- Every new endpoint is covered by a real-stack integration test; the integration suite caught two contract bugs the unit mocks could not (VARCHAR(30) sentinel overflow, the 366-day cap mismatch) — both fixed before this review.
- Frontend: no `any`; RBAC route guards and sidebar nav gating are consistent with the backend role tiers; read surfaces handle loading/empty/error/offline.

---

### Verdict: APPROVE
