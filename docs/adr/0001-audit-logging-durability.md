# 1. Audit logging durability: best-effort by default, fatal on LGPD/compliance paths

- Status: Accepted
- Date: 2026-07-07
- Deciders: Backend team

## Context

Every mutation in the backend records an audit entry through
`AuditService.LogAction` (`backend/internal/service/audit_service.go`). Across the
~16 mutation services (~25 call sites) the audit write is *best-effort*: a per-service
`audit(...)` helper calls `LogAction`, and on failure it logs via `slog` and returns
success anyway. The mutation therefore commits even if its audit record could not be
persisted.

Two facts constrain the decision:

1. `LogAction` runs on the **same per-request transaction connection** as the
   mutation. Repositories bind their queries to the `CampusTx` established in
   `backend/internal/middleware/campus_tx.go` (via `r.q(ctx)`), so a returned error
   rolls the entire request back. Making an audit write fatal — propagating its error
   up — is transactionally correct: a failed audit rolls back the whole mutation,
   guaranteeing there is never a mutation without its audit record. `audit_log` is
   append-only (a `BEFORE UPDATE/DELETE` trigger) and is not subject to RLS.

2. LGPD accountability (Art. 37) requires the controller to keep **provable records**
   of the processing operations it performs — most acutely the *erasure* of personal
   data (right to erasure, Art. 18) and the *export* of personal data. An erasure or
   an export that leaves no audit record is a compliance liability, not merely a
   missing log line.

The best-effort default exists on purpose: it protects write availability. If
`audit_log` has a transient problem, ordinary, high-frequency mutations (create a
person, transition an attendance, sync a batch) should still succeed rather than fail
the user's request. But that same trade-off is unacceptable for the small set of
deliberate, low-frequency, high-stakes operations that erase or export personal data.

## Decision

Keep the audit write **best-effort by default** for ordinary mutations, and make it
**fatal** on the enumerated LGPD/compliance-critical paths.

A dedicated method expresses the fatal contract at the call site rather than
scattering inline error handling:

`AuditService.LogRequired(ctx, params) error`
(`backend/internal/service/audit_service.go`) — a thin wrapper over `LogAction` whose
name documents that its failure is fatal and reserved for erasure/export of personal
data. The ~22 ordinary call sites are left unchanged and continue to use their
best-effort per-service helpers.

The three paths made fatal:

1. **Consent-revocation anonymization** —
   `ConsentService.anonymizePerson` in
   `backend/internal/service/consent_service.go`. Withdrawing the master
   `DATA_PROCESSING` consent triggers erasure of the person's PII; the
   `"person anonymized (LGPD consent revocation)"` audit now uses `LogRequired`, so a
   failed audit rolls the whole `RevokeConsent` back. The plain `"consent revoked"`
   audit in `RevokeConsent` stays best-effort — it is a normal mutation, not an
   erasure.

2. **Retention-sweep anonymization** —
   `RetentionService.audit` in
   `backend/internal/service/retention_service.go`, called per person from `Run`. Each
   `"person anonymized (LGPD retention policy)"` audit now uses `LogRequired`; a failed
   audit returns an error that stops the sweep and rolls that person's anonymization
   back. Already-anonymized records remain excluded by the repository query, so the
   sweep stays idempotent.

3. **Compliance-report EXPORT** —
   `ComplianceReportService.ExportCSV` in
   `backend/internal/service/compliance_report_service.go`. The `EXPORT` audit for the
   compliance report CSV now uses `LogRequired`, so an export of personal data that
   cannot be audited fails the export.

On all three paths the audit entry carries **only ids and timestamps** — the person
id and the `anonymized_at` timestamp for the anonymization paths; no entity id for the
export. It never carries scrubbed PII (CLAUDE.md MUST-NOT #7). This decision changes
only whether an audit failure is fatal, not what is logged.

Because all three run inside the per-request `CampusTx`, a fatal audit yields a clean
rollback of the accompanying mutation — never a half-applied erasure or a
partially-recorded export.

No configuration flag or global state governs this behavior: the distinction is
encoded structurally in which method a call site uses (`LogRequired` vs. the
best-effort helper).

## Consequences

- **Pro:** No erasure (consent revocation or retention sweep) and no compliance export
  can commit without a durable audit record. The LGPD accountability guarantee for
  these operations is enforced by the database transaction, not by best-effort logging.
- **Con:** `audit_log` availability now gates these specific compliance operations: if
  `audit_log` is unavailable, an erasure or export will fail rather than proceed
  unrecorded. This is acceptable because these are low-frequency, deliberate,
  high-stakes actions where correctness outweighs availability, unlike the
  high-frequency ordinary mutations that remain best-effort.
- The `LogRequired` method is the single, greppable anchor for the fatal contract;
  future compliance-critical paths opt in by calling it.

## Alternatives considered

- **Make every audit write fatal (all-fatal).** Rejected: it turns `audit_log` into a
  hard availability dependency for *every* write in the system. A transient `audit_log`
  problem would fail unrelated, high-frequency operations, degrading availability well
  beyond what the accountability requirement justifies.
- **Keep every audit write best-effort (all-best-effort — the status quo).** Rejected:
  it leaves a compliance gap precisely on erasure and export, the operations where a
  missing record is a liability, because a mutation could commit while its audit was
  silently dropped.
