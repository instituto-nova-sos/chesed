# Rule: Offline-First Assessment

## Purpose

Ensure every new React page has a documented and implemented offline behavior strategy. Prevent blank pages, data loss, or confusing UX when the user loses network connectivity.

## Rule Statement

Every new React page must document its offline behavior category and implement the corresponding strategy. No page may be created without an explicit offline classification.

## Trigger Condition

Every time the AI agent creates a new page component in `frontend/src/pages/` or adds a new route to the application router.

## Enforcement

### Offline Behavior Categories

Every page must be classified into exactly one of three categories:

#### Category A: Fully Offline-Capable
- The page works completely without network connectivity.
- Data is stored in Dexie.js (IndexedDB) and synced when online.
- Mutations are queued in the sync queue for later processing.
- **Requirements:**
  - Dexie.js table defined for the page's data in `frontend/src/offline/`.
  - Sync queue integration for create/update/delete operations.
  - Conflict resolution strategy documented (last-write-wins, merge, or manual).
  - Optimistic UI updates with rollback on sync failure.
  - Visual indicator showing offline status and pending sync count.

#### Category B: Read-Only Offline (Cached Data)
- The page displays previously fetched data when offline.
- No mutations are possible offline.
- **Requirements:**
  - Pre-caching strategy defined (cache on first load, background refresh).
  - Dexie.js table for cached data.
  - Stale data indicator visible to the user ("Last updated: X minutes ago").
  - Mutation controls disabled with clear messaging when offline.
  - Auto-refresh when connectivity is restored.

#### Category C: Online-Only
- The page requires network connectivity to function.
- **Requirements:**
  - Graceful degradation — NEVER show a blank page or unhandled error.
  - Clear offline message: "This feature requires an internet connection."
  - Navigation to offline-capable pages remains functional.
  - No data loss if connectivity drops mid-interaction (form data preserved locally).

### Classification Guide

Reference `docs/12-offline-sync-strategy.md` section "What Gets Stored Offline" for authoritative guidance. Expected Phase 1 classifications:

| Page | Category | Rationale |
|------|----------|-----------|
| Person list/detail | A | Core field operation, must work offline |
| Triage form | A | May be created during field visits |
| Attendance recording | A | Core workflow, must work offline |
| Dashboard/reports | B | Read-only summary, cached data acceptable |
| User management | C | Admin function, requires live Keycloak connection |
| Settings/config | C | Infrequent use, online acceptable |

### Documentation Requirement

When creating a new page, add a comment block at the top of the page component:

```typescript
/**
 * Offline Behavior: Category A — Fully Offline-Capable
 * - Dexie table: persons
 * - Sync: Create/Update queued, Delete queued
 * - Conflict resolution: Last-write-wins by updated_at
 * - Reference: docs/12-offline-sync-strategy.md
 */
```

### Implementation Checklist

For each new page, before marking work complete:

1. [ ] Offline category documented in component header comment.
2. [ ] Category-appropriate infrastructure implemented (Dexie table, cache, or degradation).
3. [ ] Tested with network disabled in browser DevTools.
4. [ ] No blank pages or unhandled errors when offline.
5. [ ] Sync behavior verified when coming back online (Category A).
6. [ ] Stale data indicators present (Category B).
7. [ ] Offline message displayed (Category C).

## Enforcement Mechanism

- The AI agent must classify offline behavior before implementing any new page.
- The `pre-review` hook verifies offline documentation exists for new pages.
- The agent must test offline behavior (describe the test scenario) before marking work complete.
- Pages without offline classification must not be merged.

## References

- `docs/12-offline-sync-strategy.md` — Offline sync architecture and data storage decisions
- `docs/05-architecture-proposal.md` — PWA and offline-first architecture
- `frontend/src/offline/` — Dexie.js table definitions and sync queue
- `frontend/src/hooks/` — Custom hooks for online/offline status detection

## Consequences of Skipping

- Users in the field (social workers visiting communities) lose access to critical data without connectivity.
- Blank pages or crashes when offline destroy user trust and make the app unusable in real-world conditions.
- Missing sync queues cause data loss — hours of fieldwork disappear when the user goes offline.
- Inconsistent offline behavior confuses users about what they can and cannot do without connectivity.
- The offline-first capability is a core differentiator of this platform — treating it as optional undermines the product's value proposition.
