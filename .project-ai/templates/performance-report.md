# Template: Performance Report

Use this template to document performance analysis findings at sprint boundaries or after performance-sensitive changes.

---

## Performance Report: [Sprint N / Feature Name]

**Date**: YYYY-MM-DD
**Scope**: [What was analyzed — full system, specific feature, specific endpoint]
**Analyst**: [Agent that performed the analysis]

---

### Executive Summary

**Verdict**: PASS / FAIL
**Critical findings**: [Count]
**Endpoints over budget**: [Count] / [Total]
**Frontend metrics over budget**: [Count] / [Total]

---

### API Endpoint Performance

| Endpoint | Method | Budget | Estimated/Measured | Status | Finding |
|----------|--------|--------|-------------------|--------|---------|
| /api/v1/persons | GET | 200ms | | | |
| /api/v1/persons/{id} | GET | 100ms | | | |
| /api/v1/persons | POST | 300ms | | | |

---

### Database Query Analysis

#### Slow Queries
| Query | File:Line | Estimated Time | Issue | Recommendation |
|-------|-----------|---------------|-------|----------------|
| | | | | |

#### Missing Indexes
| Table | Column(s) | Query Pattern | Impact |
|-------|-----------|--------------|--------|
| | | | |

#### N+1 Patterns
| Location | File:Line | Description | Fix |
|----------|-----------|-------------|-----|
| | | | |

---

### Frontend Performance

| Metric | Budget | Measured | Status |
|--------|--------|---------|--------|
| Bundle size (main, gzipped) | 500KB | | |
| Bundle size (total, gzipped) | 1MB | | |
| Initial load (TTI) | 2s | | |
| Route navigation | 500ms | | |
| LCP | 2.5s | | |
| FID | 100ms | | |
| CLS | 0.1 | | |

#### Bundle Analysis
| Chunk | Size (gzipped) | Contents | Optimization |
|-------|----------------|----------|-------------|
| | | | |

---

### Offline Sync Performance

| Operation | Budget | Measured | Status |
|-----------|--------|---------|--------|
| Sync batch (100 records) | 5s | | |
| Conflict resolution (per record) | 50ms | | |
| IndexedDB single read | 10ms | | |
| IndexedDB list (100) | 100ms | | |

---

### Recommendations (Prioritized)

| Priority | Finding | Impact | Effort | Recommendation |
|----------|---------|--------|--------|---------------|
| 1 | | | | |
| 2 | | | | |
| 3 | | | | |

---

### Trend (Sprint-over-Sprint)

| Metric | Sprint N-1 | Sprint N | Trend |
|--------|-----------|----------|-------|
| Avg endpoint response | | | |
| Bundle size | | | |
| Test suite execution | | | |
