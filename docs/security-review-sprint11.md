# Sprint 11 Security Review (Story 11.5)

**Date**: 2026-07-06
**Scope**: OWASP Top 10 review of the Chesed backend (`backend/`) and frontend
(`frontend/`) for the Sprint 11 integration-hardening surface, against the
project threat model (`docs/18-threat-model.md`), the Sprint 4 baseline review
(`docs/security-review-sprint4.md`), and the CLAUDE.md security rules.
**Method**: Read-only review of the production code paths plus static analysis
(`gosec`, config at `backend/.gosec.json`) and an HTTP security-header assertion
(`scripts/security-headers-check.sh`). Existing integration tests are cited as
automated evidence.

This review focuses on what Sprint 11 changes relative to the Sprint 4 baseline:
the new **unauthenticated, internet-facing public surface**
(`GET /api/v1/public/campaigns`, `POST /api/v1/public/volunteer-signup`) and the
**HSTS gap closure**. Controls already verified A-rated in Sprint 4 (JWKS auth,
campus-scoped SQL, RLS, audit logging) are re-confirmed, not re-derived. It is a
point-in-time audit; re-run when new endpoints or data flows are added.

---

## Summary

| # | Finding | OWASP | Severity | Status |
|---|---------|-------|----------|--------|
| 1 | No HSTS header on API responses (TLS downgrade / SSL-strip window) | A05 | LOW | **Resolved** — flag-gated `SecurityHeadersWith(hstsEnabled)` header + edge termination note |
| 2 | New unauthenticated public surface (campaign listing + volunteer signup) | A01 / A04 | INFO | **Controls verified present** (rate limit, campus validation, RLS fail-closed, no-PII projection, IP/UA audit) |
| 3 | Campus GUC set via `set_config($1,$2,true)` flagged by gosec G202 pattern class | A03 | INFO | **Not a vulnerability** — parameterized, injection-safe; documented instead of blanket-excluding the rule |

No CRITICAL or HIGH findings. The public surface is designed defense-in-depth and
all its controls were verified present.

---

## A01 — Broken Access Control / Multi-Tenancy

### New public surface (Finding 2 — INFO, controls verified)

Sprint 11 introduces the first **unauthenticated** operational endpoints:

- `GET /api/v1/public/campaigns` — lists active, publicly visible campaigns.
- `POST /api/v1/public/volunteer-signup` — self-service volunteer registration.

Because there is no JWT (and therefore no campus claim) on these requests, the
usual A01 control (campus from token) does not apply. The equivalent boundary is
enforced by an explicit, request-supplied `campus_id` that is **validated**, not
trusted:

- **Campus validation**: the handler resolves the incoming `campus_id` through
  `CampusRepository.FindByID` and rejects it unless the campus exists **and** is
  `is_active = true`. An attacker cannot point the public flow at an arbitrary or
  disabled campus. This closes the "trust the request body's campus" gap that
  would otherwise reproduce threat model **T3 (Campus Isolation Breach)** on an
  anonymous path.
- **RLS fail-closed (defense in depth)**: the public requests run on the
  non-owner `chesed_app` pool through `PublicCampusTx`, which sets the
  `app.current_campus` GUC to the *validated* campus before any query. If a
  future repository forgets its `WHERE campus_id` clause, RLS returns zero rows
  rather than leaking cross-campus data — the same Layer-2 guarantee described in
  threat model T3. An unset/incorrect GUC fails closed. This is the anonymous
  analogue of the authenticated `CampusTx` middleware
  (`backend/internal/middleware/campus_tx.go`).
- **No-PII public projection**: `GET /public/campaigns` returns a lean projection
  (campaign name, description, public metadata) and does **not** join or expose
  person, beneficiary, donor, or internal-team data. A01 information disclosure
  is structurally impossible because the sensitive columns never enter the query.
  This directly answers threat model **T6 (Data Exfiltration via Reports/Export)**
  for the anonymous surface: there is no bulk-PII path to exfiltrate.
- **Write path scoping**: `POST /public/volunteer-signup` writes only
  `person` + `person_role(VOLUNTEER)` + a `PENDING` `volunteer_agreement`, all
  stamped with the server-validated `campus_id`. It cannot create ADMIN/
  COORDINATOR roles or touch beneficiary data. The `volunteer_agreement` write
  runs on the pre-campus/self-register path and is RLS-excluded by design (see
  `docs/10-data-model.md` and threat model T3), reached only after the campus is
  validated.

**Re-confirmed A-rated (unchanged from Sprint 4):** authenticated routes derive
campus and role from JWT claims, never the request body; RLS (`migration 000028`)
enforces campus isolation for operational tables at the database layer.

---

## A02 — Cryptographic Failures

- Secrets remain environment-sourced (`internal/config`); no secrets are
  hardcoded. The gosec `G101` (hardcoded-credentials) rule is left **enabled** in
  `backend/.gosec.json` to keep enforcing this.
- TLS termination is at the reverse proxy / Cloudflare edge. HSTS is now emitted
  by the application when `HSTS_ENABLED=true` (see A05 / Finding 1) so the
  end-to-end TLS posture is asserted rather than merely assumed.

---

## A03 — Injection

- **All SQL uses pgx positional placeholders.** Verified again for the Sprint 11
  paths: the public campaign listing and volunteer signup queries bind every
  user-influenced value as an argument.
- **`set_config` parameterization (Finding 3 — INFO).** The campus GUC is set via
  `tx.Exec(ctx, "SELECT set_config($1, $2, true)", campusGUC, campusID.String())`
  (`backend/internal/middleware/campus_tx.go:72`, and the same shape in the new
  `PublicCampusTx`). The GUC name is a compile-time constant and the campus is a
  validated UUID bound as text — **there is no injection surface**. Static
  analyzers in the G201/G202 (SQL-string-building) rule class may pattern-match on
  any `Exec` whose first argument is a string containing SQL and raise a
  low-confidence flag. Rather than blanket-disable G201/G202 in `.gosec.json`
  (which would blind us to real string-concatenated SQL elsewhere), we keep those
  rules **enabled** and record here that this specific call site is a known-safe,
  fully-parameterized statement. If gosec surfaces it, the correct remediation is
  a narrowly-scoped `#nosec G202 -- parameterized set_config, values bound as
  args` inline directive at that single line, **not** a global exclusion.
- Frontend has no `dangerouslySetInnerHTML`; the public campaign data rendered in
  the WordPress-facing integration is JSON, not HTML injected into our DOM.

---

## A04 — Insecure Design (public surface abuse resistance)

The public endpoints are internet-facing and unauthenticated, so abuse
resistance is a design requirement, not an add-on:

- **Per-IP rate limiting** via `go-chi/httprate` on both public routes. This is
  the first in-app rate limiter in the codebase; Sprint 4 delegated rate limiting
  entirely to the edge (threat model T1/T5/T12). The edge (Cloudflare) remains the
  primary volumetric-DDoS control; the in-app per-IP limiter is a second layer
  specifically for the anonymous write path (`volunteer-signup`), bounding
  automated signup/enumeration abuse even if an attacker bypasses the edge. This
  addresses threat model **T8 (Social Engineering / abuse of the volunteer path)**
  and **T12 (service availability)** at the application tier.
- **CORS allowlist, not wildcard**: `middleware/cors.go` reflects the request
  origin only when it is in the configured allowlist and never emits `*`. The
  WordPress site's origin must be added to the production allowlist at wiring time
  (`cmd/server/main.go`) — infra item below. Credentials are not required for the
  public read path.
- **Minimal write surface**: signup creates a `PENDING` agreement requiring
  downstream human/authenticated approval before the volunteer gains any access —
  no anonymous request can self-grant a usable account.

---

## A05 — Security Misconfiguration

### HSTS gap and resolution (Finding 1 — LOW, resolved)

**Gap**: the global `SecurityHeaders` middleware
(`backend/internal/middleware/security_headers.go`) sets
`X-Content-Type-Options`, `X-Frame-Options: DENY`, `Referrer-Policy`,
`Permissions-Policy`, `Content-Security-Policy`, and `Cache-Control: no-store`,
but historically did **not** set `Strict-Transport-Security`. Sprint 4 listed HSTS
as an edge/infra item. Without HSTS a first-request SSL-strip / TLS-downgrade
window exists for clients that reach the API over `http://` before redirect.

**Resolution**: Sprint 11 adds a flag-gated variant
`SecurityHeadersWith(hstsEnabled bool)`. When `HSTS_ENABLED=true` the middleware
emits `Strict-Transport-Security: max-age=31536000; includeSubDomains`. It is
flag-gated (and defaults off) because:

1. HSTS over cleartext is meaningless and browsers ignore it — the header is only
   valid once TLS is guaranteed, which depends on the deployment topology.
2. In the standard deployment, **TLS is terminated at the nginx / Cloudflare
   edge**, and the edge is the correct, canonical place to assert HSTS for *all*
   assets (SPA + API) with a single policy and preload eligibility. The
   application header is a defense-in-depth backstop for topologies where the API
   is reachable directly over TLS. Enabling both is safe (identical directives).

The header is asserted in CI by `scripts/security-headers-check.sh`, which checks
`Strict-Transport-Security` **only on `https://` targets** (HSTS is HTTPS-only)
and every other security header on all targets.

### CSP `default-src 'none'` is correct for a JSON API (and does not break WordPress)

The API sets `Content-Security-Policy: default-src 'none'; frame-ancestors
'none'`. This is intentional and **harmless for a JSON API**:

- CSP governs how a **browser renders a document** it loads from this origin. Our
  API responses are `application/json`, not HTML documents a browser navigates to
  and executes. `default-src 'none'` means "if someone did coerce this response
  into a document context, load no subresources and run no scripts" — a strict,
  correct default that shrinks the XSS/clickjacking surface (reinforced by
  `frame-ancestors 'none'` and `X-Frame-Options: DENY`).
- **It does not break the WordPress integration.** The WordPress site consumes
  `GET /api/v1/public/campaigns` **server-side / via `fetch`**, parsing the JSON
  payload — it does not embed our API response as a rendered document or iframe.
  A response's CSP constrains *that response's own* document/subresource loads; it
  places no restriction on a third-party page that fetches the JSON and renders
  its *own* markup under its *own* CSP. So WordPress fetching and displaying
  campaign data is entirely unaffected by our `default-src 'none'`.

**Infra items**: supply the production CORS origin (including the WordPress
origin) at `cmd/server/main.go`; set `HSTS_ENABLED=true` (or assert HSTS at the
edge); keep `OIDC_SKIP_ISSUER_CHECK` false in production.

---

## A07 — Identification and Authentication Failures

Unchanged and re-confirmed from Sprint 4: JWKS signature validation via
`coreos/go-oidc`, `email_verified` enforced with 403, RBAC from token claims on
every protected route, keycloak-js PKCE S256 on the frontend. The new public
routes are **intentionally** unauthenticated and are mounted **outside** the
authenticated router group, so no auth control is bypassed — they never reach the
RBAC or `CampusTx` middleware and instead run the dedicated public pipeline
(rate limit -> campus validation -> `PublicCampusTx`).

---

## A09 — Security Logging and Monitoring Failures

- **Public-endpoint audit trail**: `POST /public/volunteer-signup` writes an
  `audit_log` entry with **`actor` nil** (anonymous), capturing the source **IP
  and User-Agent**. This gives the anonymous write path an accountable trail
  (who-from-where) without a user identity, supporting incident response for
  T8-style abuse. The nullable `campus_id` / nullable actor on `audit_log` is the
  documented, RLS-excluded shape (threat model T3, `docs/10-data-model.md`).
- **No PII in logs**: consistent with CLAUDE.md rule #7 and Sprint 4 Finding 2,
  the public paths log opaque identifiers and request metadata (IP/UA), not
  beneficiary or volunteer PII.

---

## A10 — Server-Side Request Forgery

No new outbound-request surface is introduced by the public endpoints; both
handlers only read/write the local database. The public `campus_id` is used as a
database lookup key (validated UUID), never as a URL or host — no SSRF vector.

---

## OWASP Top 10 — coverage and automated evidence

The following existing integration tests (real PostgreSQL via `testcontainers-go`,
`backend/internal/integration/`) provide automated, regression-guarding evidence
for the controls above. They are run by `make test-integration` and are the
authoritative proof that the contract holds.

| OWASP | Control | Status | Automated evidence |
|-------|---------|--------|--------------------|
| A01 Broken Access Control | Campus isolation, RBAC, RLS fail-closed | PASS | `rls_test.go`, `rbac_audit_test.go`, `sync_test.go` |
| A02 Cryptographic Failures | Env-sourced secrets, TLS at edge, HSTS flag | PASS | reviewed; `gosec` G101 enabled |
| A03 Injection | Parameterized SQL / `set_config` binding | PASS | `sync_test.go`, `document_test.go`; `gosec` G201/G202 enabled |
| A04 Insecure Design | Public-surface rate limit, minimal write | PASS | reviewed; rate-limit + campus-validation controls |
| A05 Security Misconfiguration | Security headers, CSP, HSTS, CORS allowlist | PASS | `scripts/security-headers-check.sh` |
| A06 Vulnerable Components | Dependency posture | N/A this story | dependency audit (separate track) |
| A07 Auth Failures | JWKS, `email_verified`, RBAC from claims | PASS | `auth_middleware_test.go` |
| A08 Software & Data Integrity | Audit-log append-only, no update/delete | PASS | `audit_log_test.go` |
| A09 Logging & Monitoring | Mutation + 403 + public IP/UA audit, no PII | PASS | `rbac_audit_test.go`, `audit_log_test.go` |
| A10 SSRF | No user-controlled outbound requests | PASS | reviewed; no outbound surface |

---

## Static analysis (gosec)

`backend/.gosec.json` keeps the injection rule class (G201/G202) and the
hardcoded-credential rule (G101) **enabled**; it only tunes file-permission
expectations (G301/G302/G306) and enables audit/show-ignored so suppressions are
visible. Recommended invocation enforces medium-or-higher severity **and**
confidence so the signal stays high:

```
gosec -conf .gosec.json -severity medium -confidence medium -quiet ./...
```

The one known-safe `set_config` call site (A03 / Finding 3) is documented above;
if gosec flags it, prefer a single-line `#nosec G202` with justification over any
global exclusion.

---

## Infra / configuration checklist (out of code scope)

- [ ] `HSTS_ENABLED=true` in production (or HSTS asserted at the nginx/Cloudflare edge).
- [ ] Production CORS allowlist includes the WordPress origin and the SPA origin (`cmd/server/main.go`).
- [ ] `OIDC_SKIP_ISSUER_CHECK` unset/false in production.
- [ ] Edge (Cloudflare) rate limiting configured for the public routes, in front of the in-app per-IP limiter.
- [ ] `scripts/security-headers-check.sh` run against the deployed HTTPS URL as a post-deploy smoke check.
