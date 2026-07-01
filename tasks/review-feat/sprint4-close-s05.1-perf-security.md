# Critical Review — working-tree changes (branch: feat/sprint4-close-s05.1-perf-security)

Reviewed scope: `git diff main...HEAD` — Sprint 4 closeout across three fronts:
S05.1 (Dexie encryption-at-rest capability + v1→v2 migration test), 4.8
(frontend performance: route lazy-loading, vendor chunk split, memoization, dead
`recharts` removal), and 4.9 (security fix for the HIGH cross-campus PII leak in
`CheckDuplicate`, email PII removed from logs, OWASP review doc).
Reviewer: autonomous critical-review gate (`reviewer` agent over the branch diff).

Each finding was verified against the current source and a real-Postgres
integration run, not taken on faith. Two MINOR items raised in the review were
**remediated** in commit `26e2dc5` (stale `DuplicateMatch.campus` type field
dropped; security-review doc phrasing clarified to state encryption is a tested
capability pending store-wiring). This verdict reflects the post-remediation diff.

## Quality Gate: PASS

### Conditions
| Condition | Status | Details |
|-----------|--------|---------|
| Bugs (reliability) | PASS | 0 new reliability bugs. Crypto round-trip, per-call IV freshness, and mixed plaintext/ciphertext read paths are correct; `TriageDetailPage` `useMemo` deps are `[serviceTypes, triage]`. |
| Vulnerabilities | PASS | 0. HIGH cross-campus PII leak in `CheckDuplicate` fixed and proven by a real-Postgres integration test (RED→GREEN). Email PII removed from the unverified-email log. No secret introduced (AES key generated client-side, held only in local IndexedDB `syncMeta`). |
| Security Hotspots | PASS | 100% reviewed: `CheckDuplicate` campus scoping (fixed+tested), `encryption.ts` AES-GCM (correct), auth-log PII (fixed). |
| Coverage (new code) | PASS | Backend fix covered by 2 integration tests; `encryption.ts` covered on both branches + backward-compat + IV freshness + key durability; migration covered by `db.migration.test.ts`. `src/offline` at 98.98% (gate 80%). |
| Integration tests | PASS | New backend behavior (campus boundary) has a real-Postgres integration test exercising the documented contract. No new HTTP endpoint or new frontend API-client function added (only an unused response field dropped), so no new frontend integration test is mandated. |
| Duplication | PASS | ≤3%. `RequestedServices` extraction and `PersonCard` memo reduce duplication. |
| Maintainability | PASS | Rating A. Small single-purpose crypto/base64 helpers. |
| Reliability | PASS | Rating A. Graceful degradation explicit and tested. |
| Security | PASS | Rating A. 0 vulnerabilities remaining in changed code. |

### Clean Code Assessment
| Category | Status | Findings |
|----------|--------|----------|
| Consistency | PASS | `encryption.ts` follows `offline/` module conventions; lazy-import pattern applied uniformly to all 19 routes; Go fix keeps the `%w` wrap + positional-placeholder SQL style. |
| Intentionality | PASS | Names reveal purpose (`isEncryptionAvailable`, `loadOrCreateKey`, `StoredPayload.enc`). Comments explain *why*. Dead `recharts` removed; stale `DuplicateMatch.campus` removed in remediation. |
| Adaptability | PASS | `StoredPayload` discriminated union is forward/backward compatible across the crypto boundary; campus fix confined to repository/domain. |
| Responsibility | PASS | `encryption.ts` owns only crypto/serialization; `RequestedServices` isolates a render concern; Go fix stays in repository+domain layers. |

## Complexity
No function exceeds any threshold. `CheckDuplicate` (~20 lines, cyclo 3, nesting 2, Go limits funlen 40 / cyclo 10). `encryption.ts` functions ≤24 lines, cyclo ≤2. `App.tsx` route wiring is flat JSX (cyclo ~1); the 19 `lazy()` calls are top-level constants.

## TDD Compliance
RED→GREEN commit order verified from `git log --name-only`:
- **S05.1**: `2da4350 test: … (RED)` (test files only) precedes `1e76300 feat: … (GREEN)` (introduces `encryption.ts`). ✅
- **4.9**: `2d14098 test: … (RED)` (`person_test.go` only) precedes `50c3d1d fix: … (GREEN)`. The RED test genuinely fails against the pre-fix query (returned the cross-campus match). ✅
- Perf/chore/docs commits are behavior-preserving refactors / config / docs — exempt from RED-first per the TDD rule's Allowed Exceptions.

## Issues
- **BLOCKER**: None.
- **MAJOR**: None.
- **MINOR**: Both raised items **remediated** in `26e2dc5` — (1) stale `frontend/src/types/person.ts` `DuplicateMatch.campus` field removed; (2) `docs/security-review-sprint4.md` A02 phrasing clarified (encryption is a tested capability pending store-wiring).
- **SUGGESTION**: Track the encryption store-wiring follow-up (wrap the `data` field through `encryptPayload`/`decryptPayload` in the async read/write helpers) so the capability is put on the live path. Captured in `docs/security-review-sprint4.md` and HANDOFF.

Reviewer scrutiny confirmed independently: encryption not being on the live write path is a **legitimate scoping decision** (Dexie sync hooks cannot await AES-GCM; forcing it into the reviewed drainer would regress it), the campus fix is **complete** (no other query leaks a campus; no consumer read `.Campus`), lazy-loading preserves guard/Suspense behavior, and no PII/secret was introduced.

### Verdict: APPROVE
