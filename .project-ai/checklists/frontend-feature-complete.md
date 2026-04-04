# Frontend Feature Complete Checklist

Use this checklist before marking any frontend feature as done. Every item must pass.

---

## Type Definitions

- [ ] TypeScript interfaces defined in `src/types/` (one file per domain entity)
- [ ] Strict mode — no `any` type anywhere
- [ ] Interfaces match API response schemas from `docs/11-api-design.md`
- [ ] Zod schemas defined for form validation (colocated with form or in `src/types/`)

## API Client

- [ ] API client function in `src/api/` (one file per domain, e.g., `src/api/person.ts`)
- [ ] Request/response types match `docs/11-api-design.md` exactly
- [ ] Proper error handling (network errors, 4xx, 5xx)
- [ ] Authentication token attached via keycloak-js interceptor
- [ ] `Content-Type: application/json` header set

## Hooks

- [ ] Custom hooks for data fetching and state in `src/hooks/`
- [ ] Hooks handle loading, error, and success states
- [ ] No direct API calls in components — always go through hooks
- [ ] Vitest tests for hooks and utility functions

## Components

- [ ] Components in `src/components/` — each under 150 lines
- [ ] No direct API calls in components (use hooks)
- [ ] Reusable components have no business logic baked in
- [ ] React Testing Library tests for form interactions and key behaviors

## Pages

- [ ] Page component in `src/pages/` composes components + hooks
- [ ] Protected route with keycloak-js authentication guard
- [ ] Route registered in the app router

## Forms

- [ ] React Hook Form used for all forms
- [ ] Zod schema for validation (integrated with React Hook Form via `@hookform/resolvers`)
- [ ] Validation error messages displayed inline (in Portuguese)
- [ ] Form handles loading/submitting states (disable button, show spinner)

## Responsive Design

- [ ] Mobile-responsive at 320px minimum width
- [ ] Tailwind CSS only (no CSS modules, no styled-components, no inline styles)
- [ ] Touch-friendly tap targets (minimum 44x44px)
- [ ] Tested on mobile viewport in browser dev tools

## Internationalization

- [ ] All user-visible text in Portuguese (Brazilian)
- [ ] All code comments and variable names in English
- [ ] i18n-ready structure (text strings extractable for future localization)

## Offline Behavior

- [ ] Offline behavior documented: what happens when the user is offline?
- [ ] Graceful degradation — show cached data or clear offline message
- [ ] If offline-capable: Dexie.js IndexedDB storage implemented in `src/offline/`
- [ ] Sync queue for mutations made offline (where applicable)

## Authentication & Authorization

- [ ] Route protected by keycloak-js auth guard
- [ ] User role checked for feature visibility (RBAC roles: ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER)
- [ ] Campus context loaded from Keycloak token claims

## Code Quality

- [ ] `npm test` passes with zero failures
- [ ] ESLint passes with zero warnings
- [ ] No `TODO`, `FIXME`, or `HACK` comments left unresolved
- [ ] No `console.log` statements left in production code

## Documentation Sync

- [ ] If new page/route: documented in relevant feature doc
- [ ] If new API integration: verify `docs/11-api-design.md` matches
- [ ] If offline feature: documented in `docs/12-offline-sync-strategy.md`

---

## How to Use

Run through this checklist item by item. If any item fails, fix it before proceeding.

```
Skill:   review-code (for implementation quality)
Skill:   design-frontend-feature (for design phase)
Skill:   design-offline-support (if offline-capable)
Hook:    pre-review (automated checks before marking complete)
Agent:   frontend-engineer (for implementation), tech-lead (for design review)
```
