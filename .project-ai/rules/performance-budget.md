# Rule: Performance Budget

## Purpose

Define measurable response time and resource budgets for API endpoints, frontend metrics, and offline sync operations. Prevent performance regression by establishing clear thresholds that are validated at sprint boundaries.

## Rule Statement

All API endpoints, frontend interactions, and offline sync operations must meet the defined performance budgets. Budget violations at sprint release are blocking for user-facing endpoints and advisory for internal operations.

## Budgets

### API Response Time (p95)

| Endpoint Type | Budget | Examples |
|--------------|--------|----------|
| List (paginated) | 200ms | GET /api/v1/persons, GET /api/v1/triages |
| Single record GET | 100ms | GET /api/v1/persons/{id} |
| Create/Update mutations | 300ms | POST /api/v1/persons, PUT /api/v1/triages/{id} |
| Search (text/trigram) | 500ms | GET /api/v1/persons/search?q=term |
| State transitions | 300ms | POST /api/v1/attendances/{id}/transitions |
| Report generation | 2000ms | GET /api/v1/reports/attendance-summary |
| Health check | 50ms | GET /api/v1/health |

### Frontend Performance

| Metric | Budget | Measurement |
|--------|--------|-------------|
| Initial load (cached) | 2s | Time to interactive (TTI) |
| Route navigation | 500ms | Time to first meaningful paint |
| Form submission feedback | 200ms | Time from click to UI response |
| Bundle size (main, gzipped) | 500KB | Vite build output |
| Bundle size (total, gzipped) | 1MB | All chunks combined |
| Largest Contentful Paint | 2.5s | Web Vitals |
| First Input Delay | 100ms | Web Vitals |
| Cumulative Layout Shift | 0.1 | Web Vitals |

### Offline Sync

| Operation | Budget | Measurement |
|-----------|--------|-------------|
| Sync batch (100 records) | 5s | Time to sync 100 queued mutations |
| Conflict resolution (per record) | 50ms | Time per conflict detection and resolution |
| IndexedDB read (single record) | 10ms | Dexie.js get by primary key |
| IndexedDB list (100 records) | 100ms | Dexie.js collection query |
| Service worker activation | 3s | Time from registration to active state |

### Database Queries

| Query Type | Budget | Measurement |
|------------|--------|-------------|
| Single record by PK + campus_id | 5ms | EXPLAIN ANALYZE |
| Paginated list (20 records) | 50ms | EXPLAIN ANALYZE |
| Text search (trigram) | 200ms | EXPLAIN ANALYZE |
| Aggregate report query | 500ms | EXPLAIN ANALYZE |
| Migration (up or down) | 30s | Wall clock time |

## Trigger Condition

- **Sprint release gate**: All user-facing endpoints and frontend metrics evaluated.
- **New endpoint implementation**: Budget assigned during API contract design.
- **Performance-sensitive changes**: Any change to queries, indexes, or sync logic.

## Enforcement Mechanism

- **Performance-analysis skill**: Evaluates code against budgets during sprint review.
- **Assess-release-readiness skill**: Includes performance budget as a release gate dimension.
- **Pre-release hook**: Performance budget violations flagged in release readiness report.
- **CI/CD pipeline**: Frontend bundle size check on every PR (automated).

## Budget Violation Handling

| Severity | Condition | Action |
|----------|-----------|--------|
| **BLOCKER** | User-facing endpoint > 2x budget | Block release. Must fix. |
| **MAJOR** | User-facing endpoint > budget but < 2x | Fix before next sprint. Track as tech debt. |
| **MINOR** | Internal operation > budget | Advisory. Fix if effort is low. |
| **INFO** | Within 80% of budget | No action. Monitor trend. |

## Consequences of Skipping

- User-facing latency degrades gradually without detection.
- Frontend bundle size grows unchecked, hurting mobile users on slow connections.
- Database queries slow down as data volume increases (no early warning).
- Offline sync becomes unusable on low-end devices.

## References

- `docs/05-architecture-proposal.md` — Performance requirements
- `docs/12-offline-sync-strategy.md` — Sync performance considerations
- `docs/14-deployment-strategy.md` — Infrastructure constraints
- `docs/quality/quality-gates.md` — Quality gate framework
