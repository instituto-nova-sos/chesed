# Skill: Performance Analysis

## Purpose

Analyze application performance characteristics: API response times, database query performance, frontend rendering performance, bundle size, and offline sync performance. Identify bottlenecks and recommend optimizations with measurable impact.

## When to Use / Trigger

- At sprint boundaries as part of release readiness assessment.
- When implementing performance-sensitive features (search, reports, sync).
- When a user reports slow behavior or when monitoring shows degraded response times.
- When a user says "analyze performance" or "check for bottlenecks".

## Role / Expertise

Senior performance engineer with expertise in:
- Go profiling and query optimization.
- PostgreSQL query planning (EXPLAIN ANALYZE), index effectiveness, N+1 detection.
- React rendering performance, bundle optimization, code splitting.
- Offline sync batch performance and conflict resolution efficiency.
- HTTP response time budgeting and SLA compliance.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Endpoint list | Yes | `docs/11-api-design.md` |
| Repository implementations | Yes | `backend/internal/repository/` |
| Service implementations | Yes | `backend/internal/service/` |
| Frontend bundle config | For frontend analysis | `frontend/vite.config.ts` |
| Performance budget | Yes | `performance-budget` rule |

## Process

### Step 1: Database Query Analysis

For each repository method:
1. Read the SQL query.
2. Check for N+1 patterns: Does a list operation trigger individual queries per row?
3. Check for missing indexes: Are WHERE/JOIN/ORDER BY columns indexed?
4. Check for full table scans: Are queries filtering by indexed columns?
5. Check for unnecessary columns: Are SELECT * patterns used instead of specific columns?
6. Check for missing pagination: Can list queries return unbounded result sets?
7. Verify campus_id filtering uses the campus_id index.

### Step 2: Service Layer Analysis

For each service method:
1. Count database calls per request. Multiple calls should be justified.
2. Check for unnecessary sequential operations that could be parallelized.
3. Check for repeated computations that could be cached.
4. Check for large object serialization in audit logging (JSONB).
5. Verify context propagation allows request cancellation.

### Step 3: Handler Layer Analysis

For each handler:
1. Verify pagination is enforced on list endpoints (max per_page).
2. Check response payload sizes — large responses should be paginated.
3. Verify no unnecessary data is included in responses.
4. Check for missing response compression (gzip).
5. Verify timeouts are set on request contexts.

### Step 4: Frontend Performance Analysis

1. **Bundle size**: Analyze Vite build output for oversized chunks.
   - Main bundle target: ≤ 500KB gzipped.
   - Identify large dependencies that could be lazy-loaded.
2. **Rendering**: Check for unnecessary re-renders in React components.
   - Missing `React.memo` on expensive components.
   - Missing `useMemo`/`useCallback` for expensive computations.
   - State updates that trigger full tree re-renders.
3. **Code splitting**: Verify route-level code splitting is in place.
4. **Asset optimization**: Check for unoptimized images or fonts.

### Step 5: Offline Sync Performance Analysis

1. Check sync batch sizes — are they configurable and reasonable?
2. Check conflict resolution algorithm efficiency.
3. Check IndexedDB query patterns for large datasets.
4. Verify sync queue processing doesn't block the UI thread.
5. Check for unnecessary full-sync operations (prefer delta sync).

### Step 6: Compare Against Performance Budget

For each endpoint and frontend metric, compare measured/estimated performance against thresholds from the `performance-budget` rule:

| Metric | Budget | Actual | Status |
|--------|--------|--------|--------|
| List endpoint p95 | 200ms | | |
| Single GET p95 | 100ms | | |
| Mutation p95 | 300ms | | |
| Search p95 | 500ms | | |
| Frontend initial load | 2s | | |
| Route navigation | 500ms | | |
| Sync batch (100 records) | 5s | | |
| Bundle size (gzipped) | 500KB | | |

## Outputs / Deliverables

A performance analysis report using the `performance-report` template:

1. **Endpoint performance summary** — per-endpoint analysis with budget comparison.
2. **Database query findings** — slow queries, missing indexes, N+1 patterns.
3. **Frontend performance findings** — bundle size, rendering issues, code splitting gaps.
4. **Offline sync findings** — batch timing, conflict resolution efficiency.
5. **Prioritized recommendations** — ordered by impact and effort.
6. **Verdict**: PASS (all within budget) or FAIL (specific items over budget).

## References

| Document | Path | Usage |
|----------|------|-------|
| API design | `docs/11-api-design.md` | Endpoint inventory |
| Data model | `docs/10-data-model.md` | Index information |
| Architecture | `docs/05-architecture-proposal.md` | System topology |
| Offline sync | `docs/12-offline-sync-strategy.md` | Sync design |
| Deployment | `docs/14-deployment-strategy.md` | Infrastructure constraints |

## Constraints / Quality Bar

- Must not recommend premature optimization — focus on measurable bottlenecks.
- Must reference specific code paths with file:line when reporting issues.
- Must compare findings against the performance-budget rule thresholds.
- Must prioritize recommendations by impact (user-facing latency > internal efficiency).
- Must consider the offline-first nature of the application.

## Interaction with Other Artifacts

- **Invoked by agents**: tech-lead (sprint boundary), backend-engineer (query optimization), frontend-engineer (bundle optimization).
- **Governed by rules**: performance-budget (threshold definitions).
- **Outputs use template**: performance-report.
- **Part of workflow**: performance-optimization (when issues need resolution).
- **Feeds into skills**: assess-release-readiness (performance section).
