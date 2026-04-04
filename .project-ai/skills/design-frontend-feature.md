# Skill: Design Frontend Feature

## Purpose

Design a complete React frontend feature following the pages -> components -> hooks architecture pattern. Produces TypeScript interfaces, API client functions, custom hooks, UI components, page composition, offline strategy, and mobile-first layout using React Hook Form + Zod for validation.

## When to Use / Trigger

- After a story has been analyzed (via analyze-requirements skill).
- When a user says "design frontend for story X" or "build the React side of feature Y".
- Before writing any frontend code for a new feature.

## Role / Expertise

React/TypeScript frontend engineer with deep knowledge of:
- React 18+ with functional components and hooks.
- TypeScript strict mode (no `any` types).
- React Hook Form + Zod for type-safe form validation.
- Tailwind CSS for mobile-first responsive design.
- Dexie.js for IndexedDB offline storage.
- keycloak-js for OIDC authentication.
- React Router for client-side routing.
- Zustand for global state management.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Story analysis (from analyze-requirements) | Yes | Prior skill output |
| API endpoint contract | Yes | `docs/11-api-design.md` |
| Offline requirements | Yes | `docs/12-offline-sync-strategy.md` |
| RBAC role for UI visibility | Yes | `docs/11-api-design.md` |

## Process

### Step 1: Define TypeScript Interfaces

Location: `frontend/src/types/`

1. Read the API response shapes from `docs/11-api-design.md`.
2. Create TypeScript interfaces matching the API contract exactly.
3. Create separate interfaces for: API response, create input, update input, list filter, list response.
4. Use strict types -- no `any`, no `unknown` without narrowing.

```typescript
// Example: frontend/src/types/person.ts
export interface Person {
  id: string;
  full_name: string;
  birth_date?: string;
  document_type: 'CPF' | 'SSN' | 'EU_ID' | 'PASSPORT' | 'OTHER';
  document_number?: string;
  gender?: 'M' | 'F' | 'OTHER' | 'PREFER_NOT_TO_SAY';
  email?: string;
  phone?: string;
  campus_id: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePersonInput {
  full_name: string;
  birth_date?: string;
  document_type: string;
  document_number?: string;
  // ... all fields from POST request body
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}
```

### Step 2: Define Zod Validation Schemas

Location: alongside the type file or in `frontend/src/types/`

1. Create Zod schemas that mirror the TypeScript interfaces.
2. Add field-level validation rules matching backend constraints (max lengths, patterns, required fields).
3. These schemas are used by React Hook Form for client-side validation.

```typescript
import { z } from 'zod';

export const createPersonSchema = z.object({
  full_name: z.string().min(1, 'Nome completo obrigatorio').max(200),
  document_type: z.enum(['CPF', 'SSN', 'EU_ID', 'PASSPORT', 'OTHER']),
  document_number: z.string().max(30).optional(),
  birth_date: z.string().optional(),
  phone: z.string().max(30).optional(),
  // ... validation rules per field
});

export type CreatePersonFormData = z.infer<typeof createPersonSchema>;
```

Note: UI-facing validation messages are in Portuguese (pt-BR) per project i18n rules.

### Step 3: Create API Client Functions

Location: `frontend/src/api/` (one file per domain entity)

1. Create typed HTTP functions for each endpoint.
2. Use the shared API client that auto-attaches the Keycloak access token via interceptor.
3. Return typed responses matching the TypeScript interfaces.
4. Handle error responses consistently.

```typescript
// Example: frontend/src/api/persons.ts
import { apiClient } from './client';
import { Person, CreatePersonInput, PaginatedResponse } from '../types/person';

export async function createPerson(input: CreatePersonInput): Promise<Person> {
  const response = await apiClient.post<Person>('/api/v1/persons', input);
  return response.data;
}

export async function listPersons(params: { q?: string; page?: number; per_page?: number }): Promise<PaginatedResponse<Person>> {
  const response = await apiClient.get<PaginatedResponse<Person>>('/api/v1/persons', { params });
  return response.data;
}
```

### Step 4: Create Custom Hooks

Location: `frontend/src/hooks/`

1. Create data-fetching hooks that call the API client.
2. Handle loading, error, and success states.
3. For offline-capable features, integrate with Dexie.js (read from IndexedDB first, then API).
4. Use the auth context to get current user's campus_id and role.

```typescript
// Example: frontend/src/hooks/usePersons.ts
export function usePersons(filter: PersonFilter) {
  // State: loading, error, data
  // Effect: fetch from API (or IndexedDB if offline)
  // Return: { persons, loading, error, refetch }
}
```

### Step 5: Design UI Components

Location: `frontend/src/components/`

Rules:
- Functional components only (no class components).
- Components under 150 lines; extract sub-components if larger.
- Use Tailwind CSS classes directly (no CSS modules, no styled-components).
- Mobile-first: design for 320px width first, then add responsive breakpoints.
- Form components use React Hook Form's `useForm` with Zod resolver.
- No direct API calls in components; use hooks.

Component categories:
- `components/forms/` -- PersonForm, TriageForm, AttendanceForm.
- `components/common/` -- Button, Input, Select, Modal, Badge, Alert.
- `components/layout/` -- Navbar, Sidebar, PageContainer.
- `components/tables/` -- DataTable, Pagination.

For forms:
```typescript
// Example pattern for form components
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { createPersonSchema, CreatePersonFormData } from '../../types/person';

export function PersonForm({ onSubmit }: PersonFormProps) {
  const { register, handleSubmit, formState: { errors } } = useForm<CreatePersonFormData>({
    resolver: zodResolver(createPersonSchema),
  });
  // ... render form fields with Tailwind, showing errors
}
```

### Step 6: Compose Page Component

Location: `frontend/src/pages/`

1. Pages are route-level components that compose hooks + components.
2. Pages handle: data fetching (via hooks), routing, permission checks.
3. Wrap with error boundary.
4. Check user role for conditional UI rendering (e.g., hide "New Person" button for VOLUNTEER if not allowed).

```typescript
// Example: frontend/src/pages/PersonListPage.tsx
export function PersonListPage() {
  const { persons, loading, error } = usePersons(filter);
  const { hasRole } = useAuth();
  // Compose: SearchBar + PersonTable + Pagination + "New Person" button (if role allows)
}
```

### Step 7: Define Offline Strategy (if applicable)

1. Determine if the feature requires offline support per `docs/12-offline-sync-strategy.md`.
2. If yes, define:
   - Dexie.js table schema for local storage.
   - Sync queue entry format for pending changes.
   - Client-side UUID generation (crypto.randomUUID()).
   - Sync status indicator in UI (pending/synced/conflict badge).
   - Pre-caching strategy for reference data.
3. Person creation, triage creation, and attendance creation MUST work offline.
4. Service types are pre-cached reference data (read-only offline).

### Step 8: Define Test Cases

1. **Component tests** (Vitest + React Testing Library):
   - Form renders all fields.
   - Validation errors display correctly.
   - Submit calls onSubmit with correct data.
2. **Hook tests** (Vitest):
   - Returns loading state initially.
   - Returns data after successful fetch.
   - Returns error on API failure.
3. **Page tests** (Vitest + RTL):
   - Renders with mock data.
   - Role-based UI visibility.
4. **Offline tests**:
   - Data saved to IndexedDB when offline.
   - Sync queue populated on create.
   - Sync status badge reflects state.

## Outputs / Deliverables

1. **TypeScript interfaces** for API types and form data.
2. **Zod validation schemas** with pt-BR error messages.
3. **API client functions** (typed, using shared client with Keycloak token).
4. **Custom hooks** for data fetching and offline integration.
5. **Component specifications** with Tailwind layout and mobile-first design.
6. **Page composition** with routing and permission checks.
7. **Offline strategy** (Dexie schema, sync queue, pre-caching).
8. **Test case specifications** per layer.
9. **File manifest**: exact file paths for all files to create/modify.

## References

| Document | Path | Usage |
|----------|------|-------|
| API design | `docs/11-api-design.md` | Endpoint contracts, response shapes |
| Offline sync | `docs/12-offline-sync-strategy.md` | Dexie schema, sync protocol |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | React patterns, approved dependencies |
| IAM and access control | `docs/16-iam-and-access-control.md` | RBAC roles for UI visibility |
| MVP scope | `docs/07-mvp-scope.md` | User flows for page design |

## Constraints / Quality Bar

- TypeScript strict mode: no `any` types.
- Functional components only; no class components.
- Components under 150 lines.
- No direct API calls in components (use hooks).
- Mobile-first: test at 320px width.
- Tailwind CSS only (no CSS modules, no styled-components).
- Forms use React Hook Form + Zod resolver.
- Authentication via keycloak-js adapter (no custom login forms).
- UI text in Portuguese (pt-BR); code comments and variable names in English.
- Accessibility: keyboard-navigable interactive elements, semantic HTML.

## Interaction with Other Artifacts

- **Invoked by agents**: frontend-engineer, tech-lead.
- **Depends on skills**: analyze-requirements (provides story analysis).
- **Feeds into skills**: design-offline-support (offline details), design-test-plan (test specifications).
- **Triggers skills**: maintain-docs (after implementation).
