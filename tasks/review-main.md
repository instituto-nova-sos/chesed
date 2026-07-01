# Critical Review (re-review) — working-tree changes (branch: main)

Reviewed scope: re-review after remediation of the prior REQUEST_CHANGES verdict
(2 MAJOR + 3 MINOR) on the frontend offline **sync drainer** feature
(Dexie v2, syncEngine, useOnlineSync, SyncStatusBanner, offline-create fallback,
activated E2E offline slice) and the backend golangci-lint cleanup + `toPullRecord`
error handling. Each prior finding was verified against the current source, not
taken on faith.
Reviewer: autonomous critical-review gate.

## Quality Gate: PASS

### Conditions
| Condition | Status | Details |
|-----------|--------|---------|
| Bugs (reliability) | PASS | The two MAJOR reliability defects are genuinely fixed (see Remediation Verification). No new bug introduced by the remediation: no timer leak, no broken happy path, retry loop converges (dead-letter terminates it). |
| Vulnerabilities | PASS | 0. No injection / auth bypass; sync endpoints remain behind RBAC + `auth.CampusIDFromContext` -> `ErrForbidden`. No change to the security surface. |
| Security Hotspots | PASS | Reviewed: Bearer token via apiClient, campus scoping on push/pull, per-create audit logging. The new `slog.Error` calls in `toPullRecord` log only entity type + UUID — no PII. |
| Coverage (new code) | PASS | Remediation is test-driven: `backoffDelay` schedule test, conflict-preservation test, dead-letter test, conflict/retry helper tests (syncEngine.test.ts), and a seam-driven retry-scheduling test (useOnlineSync.test.tsx). Assertions verify the corrected behavior, not the old lossy behavior. |
| Integration tests | PASS | Frontend MSW integration test for api/sync.ts remains; no new client-server contract added by the remediation. |
| Duplication | PASS | MINOR-1 resolved: `isNetworkError` extracted to `frontend/src/api/errors.ts` and imported by usePersons.ts + personOffline.ts. No remaining duplicate. |
| Maintainability | PASS | No changed function exceeds funlen 60 / cyclop 10 / nesting 3. New functions (flagConflict, bumpRetry, scheduleRetry) are small and single-purpose. |
| Reliability | PASS | Rating A. Conflict data is now preserved + recoverable; failed records auto-retry with a real backoff and a dead-letter ceiling. |
| Security | PASS | Rating A. |

### Clean Code Assessment
| Category | Status | Findings |
|----------|--------|----------|
| Consistency | PASS | Sync code matches sibling patterns; shared `isNetworkError` now used uniformly. New helpers follow existing naming/structure. |
| Intentionality | PASS | `backoffDelay` is now consumed at runtime (useOnlineSync.scheduleRetry) — the prior dead-behavioral-code finding (MINOR-2) is resolved. One residual: the `drainQueue` docstring still describes the OLD "drop the queue item (no retry)" conflict semantics (see MINOR-4) — comment drift, non-blocking. |
| Adaptability | PASS | `PushFn`/`PullRecord` keep syncEngine decoupled; the `retryDelayMs` option and `syncImplRef` indirection are clean test/runtime seams; Dexie schema is unchanged (new `conflicted`/`deadLettered` are optional fields, no version bump needed). |
| Responsibility | PASS | resolveCreated / flagConflict / bumpRetry / setCachedStatus each do one thing; scheduleRetry isolates the timer concern from syncNow. |

## Remediation Verification

**MAJOR-1 (conflict data loss) — RESOLVED.**
`flagConflict` (syncEngine.ts:142-147) now calls `db.syncQueue.update(id, { conflicted: true, ... })`
instead of `db.syncQueue.delete(...)`. The field-captured queue item is preserved in IndexedDB,
so the offline-entered record is no longer destroyed — it is recoverable. `drainQueue`
(lines 74-76) excludes `conflicted` items from subsequent batches, so the conflicted record is
neither re-pushed (no thrash/duplicate) nor silently lost. `getConflicts()` / `discardConflict()`
expose a programmatic resolution surface, and `SyncQueueItem` gained the documented `conflicted?`
field (db.ts:30-35). The test (syncEngine.test.ts:79-106) now asserts the item is PRESERVED
(`items[0].conflicted === true`) and that a second drain does NOT re-push it — the inverse of the
prior lossy assertion. Data-loss path is closed.

**MAJOR-2 (unused backoff / no auto-retry) — RESOLVED.**
`useOnlineSync.scheduleRetry` (useOnlineSync.ts:80-90) now schedules a `setTimeout` keyed off
`retryDelayMs(minRetries)` where `retryDelayMs` defaults to the real `backoffDelay`, so the
5s->30s->2m->10m schedule actually drives runtime behavior. It is invoked from `syncNow`'s
`finally` (line 109) when `autoStart` is on. A retry ceiling exists: `bumpRetry` dead-letters an
item once `retryCount > MAX_RETRIES` (5), and dead-lettered items are excluded from both
`drainQueue` and `getRetryable`, so the loop converges (a permanently-failing record can no
longer poison the queue or spin timers forever). No timer leak: an unmount-cleanup effect
(lines 137-141) clears `retryTimer.current`, and `scheduleRetry` clears any prior timer before
scheduling. The `syncImplRef` indirection correctly breaks the `syncNow <-> scheduleRetry`
dependency cycle while always firing the latest `syncNow`. Tests cover both the scheduling path
(useOnlineSync.test.tsx:101-124, seam delay 10ms) and the dead-letter terminal state
(syncEngine.test.ts:147-166).

**MINOR-1 (isNetworkError duplication) — RESOLVED.** Extracted to `api/errors.ts`; both call
sites import it. No residual copy.

**MINOR-2 (dead exported backoffDelay) — RESOLVED.** Now consumed by scheduleRetry (see MAJOR-2).

**MINOR-3 (unchecked errors in toPullRecord) — RESOLVED.** sync_service.go:517-533 now branches on
`json.Marshal` / `json.Unmarshal` errors and emits `slog.Error` (entity + id only, no PII)
instead of discarding with `_`. Consistent with the no-ignored-errors rule.

### Issues
(by severity: BLOCKER > MAJOR > MINOR > SUGGESTION)

No BLOCKER or MAJOR issues remain.

#### MINOR-4 — `drainQueue` docstring describes the old (pre-remediation) conflict semantics
File: frontend/src/offline/syncEngine.ts:62-70. The doc comment still reads
"conflict -> mark the cached entity 'conflict', drop the queue item (no retry)", which now
contradicts the implementation (the item is preserved + flagged, not dropped). Comment drift on
the exact behavior the remediation changed; update the comment to match `flagConflict`. Non-blocking.

#### MINOR-5 — Dead-lettered items are counted as "pending" in the banner
Files: frontend/src/hooks/useOnlineSync.ts:66-74 (refreshCounts), SyncStatusBanner.tsx.
`pendingCount = db.syncQueue.count() - conflicts.length` subtracts conflicted items but NOT
dead-lettered ones. A dead-lettered record therefore keeps inflating `pendingCount` and keeps the
"Sincronizar agora" button enabled, yet `drainQueue` excludes it — so the manual action can never
clear it, leaving a permanently-stuck count. Recommend subtracting dead-lettered items from the
pending tally (or surfacing them in their own state). Reliability/UX inconsistency, low impact
(MAX_RETRIES must be exceeded first); non-blocking.

#### MINOR-6 — Conflict/dead-letter resolution helpers have no UI consumer yet
Files: syncEngine.ts:160-175 (getConflicts/discardConflict), no caller in components/.
The remediation correctly closed the data-LOSS path (data is preserved + programmatically
recoverable), but the operator-facing resolution surface is still only a `conflictCount` badge in
SyncStatusBanner — `getConflicts`/`discardConflict` and the dead-letter state are not wired to any
view/merge/resubmit/discard UI. This is a follow-up feature gap, not data loss, so it does not
block; recommend tracking a dedicated "sync conflicts" resolution screen as a separate backlog item.

#### SUGGESTION-1 — funlen 40->60 relaxation (carried over) — acceptable, no action.

#### SUGGESTION-2 — person_repository.go::CheckDuplicate cross-campus query (pre-existing) —
Track separately; not touched by this PR; not blocking.

### Complexity Report
No function in the changed set exceeds thresholds (Go: cognitive <=25, cyclomatic <=10, length
<=60, nesting <=3; React/TS: cyclomatic <=10, length <=50, nesting <=3). The new frontend helpers
are small and flat: drainQueue ~16 lines (cyclo ~3), applyResults ~24 (cyclo ~5), bumpRetry ~10,
scheduleRetry ~10, syncNow ~18. Backend `toPullRecord` ~16 lines (cyclo ~3). All within limits.

### Positive observations
- Remediation is genuinely test-first: each fixed behavior has a test asserting the CORRECTED
  outcome (preservation, dead-letter, scheduled retry), replacing the prior tests that locked in
  the lossy/no-retry behavior. No rubber-stamp assertions.
- The retry loop is provably bounded: dead-lettering removes an item from both the drain set and
  the retry-eligible set, so there is no poison loop and no unbounded timer churn.
- Timer lifecycle is correct (clear-before-schedule + unmount cleanup); the `syncImplRef` pattern
  resolves the callback cycle without stale closures.
- MINOR fixes are real: shared `isNetworkError`, runtime-consumed `backoffDelay`, and checked
  JSON round-trip errors in the backend pull path.

### Verdict: APPROVE
Both MAJOR reliability defects are genuinely remediated and verified against source: MAJOR-1's
data-loss path is closed (conflicted queue items are preserved + recoverable, excluded from
re-push), and MAJOR-2's retry is real (backoffDelay drives a bounded auto-retry with a
dead-letter ceiling, no timer leak, happy path intact). All three MINORs are resolved. The
remediation introduces no new BLOCKER/MAJOR issue. Three low-impact MINORs (docstring drift,
dead-lettered items counted as pending, and the still-absent conflict-resolution UI) and two
carried-over SUGGESTIONs remain — none blocking; recommend tracking MINOR-5/MINOR-6 in the
backlog. Quality gate passes.
