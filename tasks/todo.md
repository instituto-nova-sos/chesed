# Sprint 8 — E10: Advanced Reports and Dashboards (Phase 2, final epic)

Branch: `feat/sprint8-reports-dashboards`
Delivery: parallelized (2 write-capable subagents on disjoint file sets) → `make deliver` → READY-FOR-PR. NEVER push/PR/merge.

## Scope (docs/09-backlog.md E10; detailed AC deferred to this kickoff per phase-boundary rule)

- **S10.1** Reports by service type, team, campaign
- **S10.2** Statistical charts (Recharts)
- **S10.3** Dashboard with key metrics (`GET /reports/dashboard` — documented, unimplemented)
- **S10.4** Report filter UI

## Key facts established from codebase (verified)

- Report stack already exists: attendance summary, CSV export, campaign metrics (backend + FE `ReportsPage`).
- **No new tables/migrations needed.** `attendance` has `professional_id` (indexed), `campaign_id`, `service_type_id`, `status`, `attendance_date` — read-only aggregation over existing data.
- `fetchByServiceType` groups by `service_type.category` (mirror that; `by_professional` joins `person prof`).
- `campaign_team` exists (person↔campaign + `role_in_campaign`). **No generic team entity** → "by team" = group by `professional_id`.
- **recharts is NOT installed** — S10.2 must add it.
- **No report integration tests** exist (backend or FE) — mandate applies to `/reports/attendances` + `/reports/dashboard`.
- RBAC: all report routes are **COORDINATOR+** (`main.go registerReportRoutes`). Dashboard must match.
- `DashboardPage` today computes 4 KPI cards **client-side** via `listAttendances`/`listTriages` `per_page:1`. S10.3 replaces this with the server endpoint.
- Review verdict line the gate greps: `### Verdict: APPROVE`. File: `tasks/review-feat/sprint8-reports-dashboards.md`.

## Kickoff decisions (pragmatic defaults, phase-boundary rule)

1. **S10.3 Dashboard** — `GET /reports/dashboard` per docs/11 (campus-scoped KPIs: total persons, attendances this month, by-status snapshot, upcoming scheduled, active campaigns, recent-months trend). Coordinator+. Replaces client-side KPI computation in `DashboardPage`.
2. **S10.1 By team/professional** — enrich `/reports/attendances`: add `by_professional` breakdown + optional `service_type_id`, `campaign_id`, `professional_id` filters. No new endpoint. Campaign metrics endpoint already covers "by campaign".
3. **S10.2 Charts** — add `recharts`; render `by_month` (bar/line), `by_status` (donut), `by_service_type` (bar) in `ReportsPage` + dashboard trend. Charts degrade to existing lists when offline/empty.
4. **S10.4 Filter UI** — extend `ReportFilters` with service-type select (`/service-types`), campaign select (`/campaigns`), professional filter; wire into `useAttendanceReport` params + query string.
5. **Docs first** — update docs/11-api-design.md (dashboard body + new attendance filters + `by_professional`) before implementing.

---

## Parallelization plan (disjoint file sets)

### Orchestrator (me) — shared/coupled files + RED→GREEN sequencing
- `docs/11-api-design.md`, `docs/09-backlog.md` (E10 → done), `docs/08-roadmap.md`, `HANDOFF.md`
- `backend/cmd/server/main.go` (dashboard route wiring)
- `backend/internal/integration/harness_test.go` (only if shared wiring needed)
- `frontend/src/types/index.ts`, `frontend/src/types/report.ts` (shared types)
- `frontend/src/__integration__/server.ts` (shared MSW handlers)
- RED (test-only) commit before each GREEN feat commit, per story
- Final `make deliver` + independent reviewer agent → `tasks/review-feat/sprint8-reports-dashboards.md`
- E2E `@smoke` slice + rebuild e2e stack before smoke

### Subagent A — Backend (new/owned files)
- `backend/internal/domain/report.go` — add `DashboardMetrics`, `ProfessionalCount`; extend `AttendanceReportFilter` (ServiceTypeID, CampaignID, ProfessionalID *uuid.UUID)
- `backend/internal/repository/report_repository.go` — `BuildDashboard`, `fetchByProfessional`, optional-filter WHERE extensions
- `backend/internal/service/report_service.go` — `GetDashboard`, extended filter validation
- `backend/internal/handler/report.go` — `Dashboard` handler + parse new query params
- `backend/internal/service/report_service_test.go`, `backend/internal/handler/report_test.go` (extend, table-driven)
- `backend/internal/integration/report_test.go` (NEW): dashboard happy path + DB assertions, campus boundary, RBAC 403 (VOLUNTEER), attendance filter contract, by_professional aggregation

### Subagent B — Frontend (new/owned files)
- `frontend/package.json` — add `recharts`
- `frontend/src/api/reports.ts` — `getDashboard`; extend `getAttendanceReport` params (service_type_id, campaign_id, professional_id)
- `frontend/src/hooks/useDashboard.ts` (NEW); extend `useAttendanceReport`
- `frontend/src/pages/ReportsPage.tsx` — charts + extended filters
- `frontend/src/pages/DashboardPage.tsx` — server KPIs + trend chart
- `frontend/src/components/charts/` (NEW): Recharts wrappers (Bar, Donut, Trend) with light/dark + empty/offline fallback
- `frontend/src/pages/__tests__/*`, `frontend/src/api/reports.test.ts` (extend)
- `frontend/src/__integration__/reports.integration.test.tsx` (NEW): dashboard + report happy path, error mapping, filter query-string contract

### E2E (orchestrator, after both merge locally)
- `frontend/e2e/` `@smoke`: Coordinator loads dashboard (KPI + chart visible); generates a filtered report.
- `docker compose -f docker-compose.e2e.yml up -d --build` before smoke.

## Verification gate
`make deliver` → validate-backlog → backend build/lint/test/integration → FE typecheck/lint/test/integration/coverage/build → e2e smoke → critical-review APPROVE → DoD → READY-FOR-PR.

## Review
(to be filled at closeout)
