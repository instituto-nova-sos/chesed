# Critical Review — feat/sprint11-integration-hardening

**Scope**: `git diff main...HEAD` — Sprint 11 (E12 Integration & Hardening):
WordPress public API (S12.1: unauthenticated `GET /public/campaigns` +
`POST /public/volunteer-signup`), field-level sync conflict resolution
(S12.2: `server_data` enrichment + offline resolution UI), the
PersonRole/VolunteerAgreement repository transaction fix, and ops artifacts
(backup/DR, load test, pentest config, deploy templates, runbooks). 59 files,
~5.9k insertions.

**Method**: adversarial, independent review of the diff and full files. Three
surfaces reviewed in parallel — backend (public API + repo fix + tx safety +
sync), frontend (S12.2 conflict UI), ops/scripts — each finding traced through
the code before classification. Tooling run locally.

**Remediation status**: Cycle 1 fix (commit `5b80af3`, "fix: flag queued item on
pull-detected sync conflict (S12.2)") resolves the one MAJOR found in the first
pass. Re-verified below; verdict updated to APPROVE.

**Tooling results (local, GOTOOLCHAIN=go1.25.5)**:
- `go build ./...` — PASS (exit 0)
- `go vet ./...` — PASS (exit 0)
- `golangci-lint run ./...` — PASS (exit 0, no findings) [re-run after fix]
- frontend `npm run typecheck` — PASS (tsc --noEmit clean) [re-run after fix]
- frontend `npm run lint` — S12.2 files lint clean (0 errors; 1 pre-existing
  `react-hooks/set-state-in-effect` warning in `useSyncConflicts.ts:92`, not
  introduced by the fix and non-blocking)
- Integration suite (Docker) — relied on the orchestrator's green run;
  spot-verified `public_test.go` covers the documented contract by reading it.

---

## Quality Gate

| Condition | Status | Details |
|-----------|--------|---------|
| Bugs / reliability | PASS | The one MAJOR (pull-path sync conflict invisible + silently overwritten) is RESOLVED in commit `5b80af3` and re-verified. |
| Vulnerabilities | PASS | 0. Public unauth surface is campus-isolated (validated `campus_id` → GUC → RLS fail-closed) and PII-lean. |
| Security hotspots reviewed | PASS | 100%. Public campus validation, RLS backstop, rate-limit keying, PII projections (campaign + sync server_data), audit actor/IP/UA, HSTS/CORS all traced. |
| Coverage / integration mandate | PASS | Every new endpoint has a backend integration test (`public_test.go`: happy path w/ DB assertions, campus boundary, 400/404, 429, RLS fail-closed). New API surface has a frontend MSW integration test (`conflictResolution.integration.test.tsx`); the fix adds a unit test covering the pull-conflict path (flagged, visible in `getConflicts()`, excluded from the next drain). |
| Duplication | PASS | `campusTxWithSource` shared by auth + public tx; conflict projections factored into `sync_conflict.go`; reused `base`/Querier pattern. |
| Maintainability | PASS | golangci-lint clean; new files within complexity/length budgets. |
| Reliability | PASS | Data-integrity defect on the pull conflict path resolved and re-verified. |
| Security | PASS (A) | HSTS value pinned identically in middleware + check script; no secrets committed; gosec config hardened, not weakened. |

## Clean Code

- **Consistency**: the public surface follows the established
  handler→service→repository→domain layering, campus-from-context, generic-error
  mapping, and audit-at-service conventions. The public tx middleware is built by
  parameterizing the *existing* `campusTxWithSource` machinery rather than
  duplicating commit/rollback logic — the right extraction. The pull-conflict fix
  mirrors the existing push-path `flagConflict` semantics (a new
  `flagQueuedConflict` helper), keeping the two conflict origins consistent.
- **Intentionality**: names reveal purpose; comments explain the non-obvious
  (unspoofable rate-limit key, body read-once-then-restore, RLS-exempt audit,
  savepoint semantics of the nested `Begin`, why the queued item must be flagged).
- **Adaptability**: the repository fix makes PersonRole/VolunteerAgreement writes
  correctly resolve the request Querier (`base` + `r.q(ctx)`), so they join the
  per-request tx or the BypassRLS admin connection as the route dictates.
- **Responsibility**: functions do one thing; sync conflict enrichment was split
  into `sync_conflict.go` (112 lines) instead of growing `sync_service.go`.

## Complexity

Within thresholds. New Go files: `public_campus_tx.go` 141, `public_service.go`
205, `sync_conflict.go` 112, `handler/public.go` 92, `rate_limit.go` 44,
`security_headers.go` 38 — all under 400, and golangci-lint (gocognit/cyclop/
funlen where configured) reports no per-function breach. React: `ConflictDiff.tsx`
164 and `SyncConflictsPage.tsx` 151 are under the 300-line / 80-JSX / nesting-3
thresholds (both split into helper components). The new `flagQueuedConflict`
helper is a flat 15-line function.
Note: `sync_service.go` is 653 lines (over the 400 file budget) but it was already
643 on `main` — pre-existing debt, +10 lines this sprint; the new logic went into
`sync_conflict.go`. INFO, not a Sprint 11 regression.

---

## Issues by Severity

### BLOCKER
None.

### MAJOR — RESOLVED (commit 5b80af3)
- **Pull-originated sync conflicts were unreachable in the S12.2 UI and were
  silently overwritten (data loss).** `frontend/src/offline/syncEngine.ts`
  (`mergePullRecords`).
  Original defect: on a pull conflict (`local.syncStatus === 'pending'`) the code
  set the cached **entity** to `'conflict'` and captured a `db.conflicts`
  snapshot, but did **not** set `conflicted: true` on the corresponding
  `syncQueue` item. Consequences (both verified end-to-end in the first pass):
  (1) invisible in the resolution UI — `getConflicts()` filters on
  `item.conflicted`; (2) silent last-write-wins on the next drain —
  `drainQueue` re-pushed the unflagged item and overwrote the server change.

  **Fix (verified)**: `mergePullRecords` now calls a new `flagQueuedConflict(
  entityType, entityId)` helper in the pull-conflict branch. The helper queries
  `db.syncQueue.where('entityId').equals(entityId)` (— `entityId` is a Dexie
  index per `db.ts:79` `'++id, entityType, entityId, createdAt'`, so the query is
  valid, no `SchemaError`), filters on `item.entityType === entityType &&
  !item.conflicted`, and sets `conflicted: true` + `lastError: 'pull conflict'`.
  Re-traced: pull conflict → queue item flagged → `getConflicts()` surfaces it →
  `drainQueue`'s `!conflicted && !deadLettered` filter excludes it → no re-push,
  no overwrite. No new defect: the `entityType` match is belt-and-suspenders (the
  `entityId`/`sync_id` UUID is already unique per record), the `!item.conflicted`
  filter makes it idempotent, the push path (`flagConflict`) is untouched, and
  there is no `any` or swallowed error.
  A failing-first (RED) unit test was added
  (`syncEngine.conflict.test.ts`, "flags the queued item on a pull conflict so it
  is resolvable and not re-drained") asserting: `item.conflicted === true`, the
  entity appears in `getConflicts()`, and a subsequent `drainQueue(push)` does NOT
  call push. This is a correct behavioral test (it would have caught the original
  bug), not a rubber-stamp.

### MINOR
- `backend/internal/handler/public.go:57` — the `VolunteerSignup` doc comment says
  `POST /api/v1/public/volunteers`; the actual (and documented) route is
  `/public/volunteer-signup`. Stale comment only; the route in
  `cmd/server/main.go:232` and `docs/11-api-design.md` is correct.
- `frontend/src/pages/SyncConflictsPage.tsx:33,133` — two `as` type assertions
  (`item.id as number`, `snapshots[item.entityId] as ConflictSnapshot`). Both are
  currently safe (persisted Dexie key always present; guarded by a `!= null`
  filter) and are not `any`, but they bypass the checker; a narrowing helper is
  cleaner.
- `scripts/restore.sh:114` — the credential-redaction regex only redacts a
  `user:pass@` segment immediately after the URL scheme; best-effort log hygiene,
  low impact (not the primary credential path, which is `PGPASSWORD`/`.pgpass`).

### INFO
- `frontend/src/hooks/useSyncConflicts.ts:92` — `react-hooks/set-state-in-effect`
  lint warning (calling `refresh()` which `setState`s inside `useEffect`).
  Pre-existing S12.2 pattern, not touched by the fix; refactor to an event-driven
  refresh if desired.
- Rate limiting is one per-IP limiter over the whole `/public` group (default 60/min,
  `PUBLIC_RATE_LIMIT_RPM`), while `docs/11-api-design.md` lists "~60/min" for GET and
  "~10/min" for POST. The docs' "~" signals approximate and the single group limiter
  is a reasonable backstop, but doc and impl differ in granularity — reconcile the
  doc or add a POST-specific limiter if per-endpoint budgets are intended.
- `make security-scan` pipes the header-check failure into `|| echo ...`
  (`backend/Makefile:71-72`), so a real HSTS/CSP regression would not fail the
  target — it collapses "API unreachable" and "headers wrong" into the same soft
  warning. Consider a hard failure when the API is reachable.
- `sync_service.go` exceeds the 400-line file budget (653), pre-existing debt
  (+10 this sprint); consider a further split in a future refactor.
- The public audit IP comes from `X-Forwarded-For` (`extractIP`), which is
  spoofable — acceptable for an audit metadata field (not a security control), and
  the rate limiter correctly uses the unspoofable `RemoteAddr` instead.

---

## Verified Correct (adversarial checks that passed)

- **No cross-campus leak on the public surface.** `PublicCampusValidator` resolves
  `campus_id` (query for GET, body-peek for POST) against `campus.FindByID`
  (RLS-exempt table, looked up on the pool before the tx), stashes it in context,
  and `PublicCampusTx` sets it as the `app.current_campus` GUC on the **non-owner**
  pool. The campaign list is doubly guarded (`WHERE campus_id = $1` **and** RLS);
  the person insert is guarded by the RLS `WITH CHECK` (a campus_id ≠ GUC insert is
  rejected). `TestPublicCampaigns_CampusBoundary` and `TestPublicSignup_RLSFailsClosed`
  prove both directions (visible under own campus, invisible/rejected across
  campuses). Unknown/inactive campus → generic 404 (never discloses existence).
- **Body read-once-then-restore works.** `campusIDFromBody` buffers the body and
  resets `r.Body = io.NopCloser(bytes.NewReader(bodyBytes))`;
  `TestPublicCampusValidator_POST_RestoresBody` proves a downstream handler can
  re-read the full JSON (incl. `full_name`) after the peek.
- **Rate-limit key is unspoofable.** `keyByRemoteIP` uses `RemoteAddr` (TCP peer),
  never a forwarded header, canonicalized for IPv6; `TestPublicVolunteerSignup_RateLimited`
  proves the 429 path.
- **Audit for public signup**: nil actor (empty Subject ⇒ NULL `user_id`),
  campus-scoped, IP/UA captured, `new_values` limited to `{full_name, role_type}`;
  `audit_log` is intentionally RLS-exempt (migration 000028) so the insert on the
  non-owner pool succeeds. Integration test asserts all of this at the DB level.
- **PII-lean projections**: the public campaign response is
  `CampaignListItem` (id/name/type/status/dates/location_name — no coordinator,
  description, address, team, created_by); the integration test asserts the hidden
  fields never appear on the wire. The sync `server_data` projection omits
  document_number/birth_date/gender (person), clinical notes (triage),
  observations/recommendations (attendance); unit tests seed PII on the server row
  and assert it does **not** leak.
- **Transaction safety of the nested `Begin`.** Under `PublicCampusTx`,
  `PersonRepository.Create` calls `r.q(ctx).Begin(ctx)`, which — on the already-open
  request tx — opens a pgx **savepoint**, not a second connection. On success the
  savepoint releases and the middleware commits the outer tx; on a duplicate/error
  the savepoint rolls back, the service returns an error, the handler writes ≥400,
  and the middleware rolls back the outer tx. This is the same pattern already used
  by the triage/attendance repos on `main`. The happy-path integration test proves
  the person + role + agreement all commit together (the role INSERT sees the
  not-yet-committed person via the shared tx — impossible if it ran on a fresh
  connection).
- **The repository fix is correct and complete.** On `main`, PersonRoleRepository
  and VolunteerAgreementRepository held a raw `*pgxpool.Pool` and wrote via
  `r.pool` — bypassing the request tx entirely. Switching to `base` + `r.q(ctx)`:
  (a) makes the public signup flow work at all (role/agreement now see the pending
  person); (b) for `registerProtectedRoutes` callers (AddRole/ToggleRole/agreement
  create) their writes now correctly join the `CampusTx` request tx — an atomicity
  improvement, not a regression; (c) for `registerAgreementRoutes` callers
  (accept/reject/upload — no `CampusTx` on that group) `r.q(ctx)` finds no request
  Querier and falls back to the constructed pool, i.e. **identical behavior** to
  before; (d) for `registerAuthOnlyRoutes` (self-register/onboarding) `BypassRLS`
  installs the admin owner Querier in context, so these now route to the owner
  connection as intended. `go build ./...` confirms every caller compiles.
- **HSTS contract matches byte-for-byte.** `security_headers.go:7`
  `max-age=63072000; includeSubDomains; preload` == the assertion in
  `scripts/security-headers-check.sh` and in `security_headers_test.go`; all five
  base headers (nosniff / DENY / Referrer-Policy / CSP / Cache-Control) match the
  script. HSTS is flag-gated (`HSTS_ENABLED`, HTTPS-only).
- **Docs contract match.** `docs/11-api-design.md` endpoints, request/response
  shapes, error codes (400/404/429), and the extended `/sync/push` conflict fields
  (`server_data`, `server_updated_at`, `conflicting_fields`) match the domain JSON
  tags and handler responses exactly.
- **Ops/scripts**: `backup.sh`/`restore.sh`/`security-headers-check.sh` set
  `-euo pipefail`; `restore.sh` gates destructive `pg_restore --clean` behind
  `--confirm` and a pre-restore checksum; `backup.sh` fails closed on `pg_dump`
  error; `.env.prod.template` contains only placeholders; `.gosec.json` suppresses
  no rules (hardened perms, `nosec:false`, G101 entropy tuning).

---

## Recommendations Priority

1. (Done) The MAJOR pull-conflict defect is fixed and re-verified — no action.
2. Fix the stale route comment in `handler/public.go:57` (MINOR).
3. Consider the INFO items in a future pass (rate-limit doc/impl granularity;
   `security-scan` hard fail; `useSyncConflicts` effect refactor; a future
   `sync_service.go` split).

---

### Verdict: APPROVE
