// Chesed — Sprint 11 / S12.5 load test (RNF-08: 100 concurrent users)
//
// This k6 scenario drives 100 virtual users against the public,
// unauthenticated API surface (the highest-fan-out, cache-cold read path)
// plus an optional authenticated sync push. It is the committable artifact
// for RNF-08: the script and its thresholds encode the acceptance target
// (p95 < 500ms, < 1% failed requests at 100 concurrent VUs).
//
// RNF-08 target
//   - 100 concurrent users, p95 request latency < 500ms, error rate < 1%.
//   - Primary throughput driver: GET /public/campaigns (documented ~60/min/IP).
//
// How to run (public-only mode — no token required):
//   k6 run -e CAMPUS_ID=<uuid> scripts/load-test/k6-public-and-sync.js
//
// With a custom base URL and an authenticated sync push leg:
//   k6 run \
//     -e BASE_URL=https://staging.chesed.example/api/v1 \
//     -e CAMPUS_ID=<uuid> \
//     -e AUTH_TOKEN=<keycloak-access-token> \
//     scripts/load-test/k6-public-and-sync.js
//
// OPERATIONS NOTE (documented-manual step):
//   Actually running 100 concurrent VUs against real infrastructure, and then
//   interpreting the p95 / error-rate summary against RNF-08, is a manual ops
//   step performed against a staging-like environment — it is NOT wired into
//   the automated `make deliver` gate (which would need a live stack and would
//   be non-deterministic in CI). What is committed and version-controlled is
//   THIS script plus its thresholds; `make load-test` only asserts the script
//   is present and parses (SKIPPED-NEEDS-K6 when k6 is not installed).

import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';
const CAMPUS_ID = __ENV.CAMPUS_ID || '';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

export const options = {
  // Ramp to 100 VUs, hold, then ramp down — a standard sustained-load shape
  // for a concurrency target rather than a spike test.
  scenarios: {
    public_and_sync: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 100 }, // ramp 0 -> 100 concurrent users
        { duration: '1m', target: 100 },  // hold at 100 (the RNF-08 target)
        { duration: '15s', target: 0 },   // ramp down
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    // RNF-08 acceptance criteria.
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export function setup() {
  if (!CAMPUS_ID) {
    throw new Error(
      'CAMPUS_ID is required. Run: k6 run -e CAMPUS_ID=<uuid> scripts/load-test/k6-public-and-sync.js',
    );
  }
  return { campusID: CAMPUS_ID };
}

// listActiveCampaigns exercises the primary read path. This is the main
// throughput driver: at ~60/min/IP it sustains steady load, and it returns a
// lean, no-PII projection so it is safe to hammer.
function listActiveCampaigns(campusID) {
  const url = `${BASE_URL}/public/campaigns?campus_id=${campusID}&page=1&per_page=20`;
  const res = http.get(url, { tags: { name: 'GET /public/campaigns' } });

  check(res, {
    'campaigns: status is 200': (r) => r.status === 200,
    'campaigns: body has data array': (r) => {
      if (r.status !== 200) {
        return false;
      }
      try {
        return Array.isArray(r.json('data'));
      } catch (_e) {
        return false;
      }
    },
  });
}

// volunteerSignup exercises the anonymous write path. The unique
// ${__VU}-${__ITER} suffix prevents dedupe so each attempt is a genuine insert.
//
// Rate-limit awareness (INTENTIONAL): the endpoint is throttled to ~10/min/IP.
// A k6 run originates from a single IP, so once the per-IP budget is spent the
// server correctly returns 429 for the rest of the window. That 429 is the
// control being VALIDATED, not a failure — so we accept 201 OR 429 as the
// expected outcome. Treating 429 as a failure here would falsely fail the run
// and, worse, would reward removing the very rate limit we want to prove works.
function volunteerSignup(campusID) {
  const suffix = `${__VU}-${__ITER}`;
  const payload = JSON.stringify({
    full_name: `Load Test Volunteer ${suffix}`,
    email: `loadtest+${suffix}@example.com`,
    campus_id: campusID,
  });
  const res = http.post(`${BASE_URL}/public/volunteer-signup`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'POST /public/volunteer-signup' },
  });

  check(res, {
    'signup: created (201) or rate-limited (429)': (r) =>
      r.status === 201 || r.status === 429,
  });
}

// pushSync optionally exercises the authenticated sync push. It is guarded by
// AUTH_TOKEN so the whole script runs in public-only mode without a token.
function pushSync() {
  if (!AUTH_TOKEN) {
    return;
  }
  const suffix = `${__VU}-${__ITER}`;
  const payload = JSON.stringify({
    records: [
      {
        entity_type: 'person',
        sync_id: `00000000-0000-4000-8000-${String(__VU).padStart(6, '0')}${String(
          __ITER % 1000000,
        ).padStart(6, '0')}`,
        data: {
          full_name: `Sync Person ${suffix}`,
          document_type: 'CPF',
          nationality: 'BRA',
        },
      },
    ],
  });
  const res = http.post(`${BASE_URL}/sync/push`, payload, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${AUTH_TOKEN}`,
    },
    tags: { name: 'POST /sync/push' },
  });

  check(res, {
    'sync push: status is 200': (r) => r.status === 200,
  });
}

export default function (data) {
  listActiveCampaigns(data.campusID);
  volunteerSignup(data.campusID);
  pushSync();

  // Pace each VU so 100 VUs produce a realistic request rate rather than a
  // tight busy-loop; keeps the per-IP rate-limit interaction representative.
  sleep(1);
}
