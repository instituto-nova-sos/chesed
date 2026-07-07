# Load Testing — RNF-08 (100 concurrent users)

This directory holds the [k6](https://k6.io) load-test scenario that encodes the
RNF-08 performance acceptance target for the Chesed platform:

> **RNF-08** — the API sustains **100 concurrent users** with **p95 request
> latency < 500ms** and a **request error rate < 1%**.

The committed, version-controlled artifact is the **script plus its thresholds**
(`k6-public-and-sync.js`). Executing 100 concurrent VUs against real
infrastructure and interpreting the p95 / error-rate summary is a
**documented-manual operations step** run against a staging-like environment —
it is intentionally *not* part of the automated `make deliver` gate, because it
needs a live stack and would be non-deterministic in CI.

## What the scenario exercises

| Call | Auth | Role in the test |
|------|------|------------------|
| `GET /public/campaigns?campus_id=…` | none | Primary throughput driver (lean, no-PII read path, ~60/min/IP). |
| `POST /public/volunteer-signup` | none | Anonymous write path. Rate-limited to ~10/min/IP — see note below. |
| `POST /sync/push` | Bearer | Optional authenticated leg, only when `AUTH_TOKEN` is set. |

### Rate-limit awareness (why 429 is expected, not a failure)

`POST /public/volunteer-signup` is throttled to **~10/min per IP**. A k6 run
originates from a single IP, so once the per-IP budget for the window is spent,
the server correctly returns **HTTP 429** for the remaining signup calls. The
scenario therefore treats **201 OR 429** as the expected outcome for that call:
the 429 is the security control being *validated*, not a defect. The
`GET /public/campaigns` path (~60/min) is what actually drives sustained
throughput for the p95 measurement.

## Install k6

- macOS: `brew install k6`
- Debian/Ubuntu: see <https://k6.io/docs/get-started/installation/>
- Docker: `docker run --rm -i grafana/k6 run - < k6-public-and-sync.js`

Full instructions: <https://k6.io/docs/get-started/installation/>

## Environment variables

| Var | Required | Default | Purpose |
|-----|----------|---------|---------|
| `BASE_URL` | no | `http://localhost:8080/api/v1` | API root the scenario targets. |
| `CAMPUS_ID` | **yes** | — | UUID of an existing active campus (query param + signup body). |
| `AUTH_TOKEN` | no | — | Keycloak access token; enables the authenticated `POST /sync/push` leg. Without it the script runs in public-only mode. |

## Run it

Public-only mode (no token required):

```bash
k6 run -e CAMPUS_ID=<uuid> scripts/load-test/k6-public-and-sync.js
```

Against a staging URL with the authenticated sync leg:

```bash
k6 run \
  -e BASE_URL=https://staging.chesed.example/api/v1 \
  -e CAMPUS_ID=<uuid> \
  -e AUTH_TOKEN=<keycloak-access-token> \
  scripts/load-test/k6-public-and-sync.js
```

k6 exits non-zero if either RNF-08 threshold
(`http_req_duration: p(95)<500`, `http_req_failed: rate<0.01`) is breached, so
the run is self-asserting once pointed at a live environment.

## `make load-test` (repo-root target)

The root `Makefile` wires this scenario as a **non-blocking** convenience target:

```bash
make load-test CAMPUS_ID=<uuid>
```

- When **k6 is installed**, it runs `k6 run scripts/load-test/k6-public-and-sync.js`.
- When **k6 is absent**, it prints
  `load-test: SKIPPED-NEEDS-K6 (…)` and returns success — the same
  `SKIPPED-NEEDS-<dep>` convention used elsewhere in the repo for optional,
  environment-gated tooling. This keeps `make load-test` runnable on any
  developer machine without forcing a k6 install, while the target stays ready
  for use in a provisioned load-testing environment.

## Related Go benchmarks

RNF-08 hot paths also have micro-benchmarks under
`backend/internal/service/`:

- `BenchmarkPushBatch` (`sync_service_bench_test.go`) — the sync push batch path.
- `BenchmarkListActiveCampaigns` (`public_service_bench_test.go`) — the public
  campaign list read path.

Run them without a live database (they use in-memory mocks):

```bash
cd backend && GOTOOLCHAIN=go1.25.5 go test -run '^$' -bench=. -benchmem ./internal/service/...
# or, from the repo root:
make bench
```
