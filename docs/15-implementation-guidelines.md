# 15 - Implementation Guidelines

## Coding Principles

### Backend (Go)

1. **Standard library first**: Use `net/http`, `slog`, `testing`, `encoding/json` before reaching for third-party libraries.
2. **Explicit over magic**: No code generation, no reflection-heavy frameworks, no "convention over configuration." Every behavior should be traceable by reading the code.
3. **Error handling**: Always handle errors explicitly. Never use `_` for errors. Use `fmt.Errorf("context: %w", err)` for wrapping.
4. **Dependency injection**: Pass dependencies (database, logger, config) as function parameters or struct fields. No global singletons.
5. **Interface at consumption site**: Define interfaces where they're used, not where they're implemented.
6. **Repository pattern**: All database access goes through repository interfaces. Services never import `pgx` directly.
7. **Context propagation**: Pass `context.Context` as the first parameter to all functions that do I/O.
8. **Table-driven tests**: Use table-driven test patterns for comprehensive coverage.

### Frontend (React + TypeScript)

1. **TypeScript strict mode**: `strict: true` in tsconfig. No `any` types except for truly dynamic data.
2. **Functional components only**: No class components. Use hooks for state and effects.
3. **Co-location**: Keep component, styles, and tests in the same directory.
4. **Small components**: If a component exceeds 150 lines, extract sub-components.
5. **Custom hooks for logic**: Extract reusable logic into custom hooks (`useAuth`, `useOffline`, `useSync`).
6. **Form validation**: Use React Hook Form with Zod schemas for type-safe validation.
7. **Error boundaries**: Wrap route-level components with error boundaries.
8. **Accessibility**: All interactive elements must be keyboard-navigable. Use semantic HTML.
9. **Authentication**: Authentication is handled by the Keycloak JS adapter. No custom login forms. No custom token management code. The adapter handles login redirect, token refresh, and logout.

### Approved Go Dependencies

| Dependency | Purpose |
|------------|---------|
| `go-chi/chi` | HTTP routing |
| `jackc/pgx` | PostgreSQL driver |
| `golang-migrate` | Database migrations |
| `coreos/go-oidc` | OIDC token validation (Keycloak) |
| `golang-jwt` | Indirect dependency via go-oidc; not used directly for token issuance |
| `go-playground/validator` | Struct tag validation |
| `slog` (stdlib) | Structured logging |

### Approved React Dependencies

| Dependency | Purpose |
|------------|---------|
| `keycloak-js` | Official Keycloak JavaScript adapter for OIDC authentication |
| `react-hook-form` | Form state management |
| `zod` | Schema validation |
| `dexie` | IndexedDB wrapper (offline support) |
| `recharts` | Charts (Phase 2) |
| `react-router` | Client-side routing |
| `zustand` | Global state management |

---

## Modularization Rules

### Backend

```
backend/
├── cmd/server/main.go          # Wire everything together; start server
├── internal/
│   ├── config/                  # Environment config struct + loader
│   ├── domain/                  # Pure data structs (no behavior, no dependencies)
│   ├── handler/                 # HTTP handlers: parse request → call service → write response
│   ├── service/                 # Business logic: validate → orchestrate → return result
│   ├── repository/              # Database operations: SQL → domain structs
│   ├── middleware/              # HTTP middleware: auth, audit, CORS, rate limit
│   └── sync/                   # Offline sync engine
├── migrations/                  # SQL migration files
└── scripts/                    # Setup and utility scripts
```

**Rules:**
- `handler` depends on `service` (never on `repository` directly)
- `service` depends on `repository` interfaces (defined in `service` package)
- `repository` depends on `domain` structs and database driver
- `domain` has zero dependencies (pure structs)
- `middleware/auth.go`: Validates Keycloak OIDC tokens by verifying the JWT signature against Keycloak's JWKS endpoint (cached with TTL). Extracts `sub`, `realm_access.roles`, and `campus_id` custom claim from the token. No token issuance logic exists in the application.
- `middleware` depends on `config` and may call `service` for auth validation
- No circular dependencies

### Frontend

```
frontend/src/
├── api/            # API client functions (one file per domain)
├── components/     # Reusable UI components
│   ├── common/     # Buttons, inputs, modals, badges
│   ├── forms/      # Form components (PersonForm, TriageForm)
│   ├── layout/     # Navbar, Sidebar, PageContainer
│   └── tables/     # DataTable, Pagination
├── hooks/          # Custom hooks
├── offline/        # IndexedDB, sync queue, conflict resolution
├── pages/          # Route-level page components
├── store/          # Global state (auth, offline status)
├── types/          # TypeScript interfaces (shared)
└── utils/          # Pure utility functions
```

**Rules:**
- `pages` import from `components`, `hooks`, `api`, `store`
- `components` import from `hooks`, `types`, `utils` (never from `pages`)
- `api` imports from `types` only
- `hooks` may import from `api`, `offline`, `store`
- `offline` has no UI dependencies

---

## Naming Conventions

### Backend (Go)

| Element | Convention | Example |
|---------|-----------|---------|
| Package names | Lowercase, single word | `handler`, `service`, `repository` |
| Exported types | PascalCase | `PersonService`, `AttendanceHandler` |
| Unexported functions | camelCase | `validateTransition`, `buildQuery` |
| Constants | PascalCase | `StatusCompleted`, `RoleAdmin` |
| Database columns | snake_case | `full_name`, `campus_id`, `created_at` |
| API endpoints | kebab-case | `/forgot-password`, `/check-duplicate` |
| Environment variables | UPPER_SNAKE_CASE | `DB_HOST`, `KEYCLOAK_URL` |
| File names | snake_case.go | `person_handler.go`, `attendance_service.go` |
| Test files | snake_case_test.go | `person_handler_test.go` |
| Migration files | `NNNNNN_description.up.sql` | `000001_create_campus.up.sql` |

### Frontend (TypeScript/React)

| Element | Convention | Example |
|---------|-----------|---------|
| Components | PascalCase.tsx | `PersonForm.tsx`, `AttendanceList.tsx` |
| Hooks | camelCase with `use` prefix | `useAuth.ts`, `useOffline.ts` |
| Utilities | camelCase.ts | `formatDate.ts`, `validateCPF.ts` |
| Types/Interfaces | PascalCase with suffix | `PersonData`, `AttendanceResponse` |
| Constants | UPPER_SNAKE_CASE | `API_BASE_URL`, `SYNC_INTERVAL` |
| CSS classes | Tailwind utilities | `className="flex items-center p-4"` |
| API files | camelCase.ts | `personApi.ts`, `attendanceApi.ts` |

---

## Test Strategy

### Backend Testing Pyramid

```
┌───────────────────┐
│   E2E Tests (few)  │  Full API → DB round-trips
├───────────────────┤
│  Integration Tests │  Service + Repository with real DB
├───────────────────┤
│   Unit Tests (many)│  Service logic, validators, helpers
└───────────────────┘
```

**Unit tests**: Test service logic with mocked repository interfaces. Cover business rules (workflow transitions, duplicate detection, RBAC checks).

**Integration tests**: Test repository implementations against a real PostgreSQL instance (Docker in CI). Cover SQL queries, migrations, and data integrity.

**E2E tests**: Test critical API flows end-to-end (login → create person → create triage → create attendance → generate report).

**Coverage target**: 80%+ for service layer; 60%+ for handlers; focus on business-critical paths.

### Frontend Testing

**Unit tests**: Vitest for utility functions, hooks, and component rendering.

**Component tests**: React Testing Library for form validation, user interactions.

**Integration tests**: MSW (Mock Service Worker) to mock API responses and test full page flows.

**E2E tests (Phase 2+)**: Playwright for critical user flows on mobile and desktop viewports.

### Test Data

- Use factory functions to generate test data (not fixtures)
- Each test creates its own data (no shared mutable state)
- Database tests use transactions that are rolled back after each test

---

## Legacy Data Migration (from SOS-Gestao-Final)

If data exists in the legacy Django system that needs to be carried forward:

### Migration Mapping

| Legacy Model | Chesed Table(s) | Notes |
|-------------|-------------|-------|
| `Pessoa` | `person`, `address` | Split address fields; add UUID |
| `Atendimento` | `attendance` | Map `data_atendimento` → `attendance_date` |
| `TipoServico` | `service_type` | Add `category` field |
| `User` + `PerfilUsuario` | `person` + `app_user` + `person_role` | Merge user profile into person |
| `PermissaoCustomizada` | RBAC config | Map to v2 role-based permissions |
| `AuditoriaAcesso` | `audit_log` | Transform field names |
| `DespesaONG` | Not migrated | Out of scope for Chesed |
| `RegistroPonto` | Not migrated | Out of scope for Chesed |

### Migration Process

1. **Export**: Script reads legacy SQLite/PostgreSQL and writes JSON
2. **Transform**: Map legacy models to Chesed schema
3. **Load**: Insert into Chesed PostgreSQL via API or direct SQL
4. **Validate**: Compare record counts; spot-check data integrity
5. **Cutover**: Freeze legacy writes → final migration → switch DNS → decommission legacy

---

## Code Review Checklist

Before merging any PR:

- [ ] All tests pass (CI green)
- [ ] No lint warnings
- [ ] New code has tests for business logic
- [ ] SQL migrations have both `up` and `down` files
- [ ] API changes are reflected in `docs/11-api-design.md`
- [ ] No hardcoded secrets or credentials
- [ ] Error messages are user-friendly (no stack traces in API responses)
- [ ] Audit logging added for new data mutations
- [ ] Campus scoping applied to new queries
- [ ] Mobile-first responsive design verified
- [ ] Offline behavior considered (does this feature work without internet?)
- [ ] No custom authentication or credential handling code (all auth delegated to Keycloak)
- [ ] Keycloak realm changes exported and committed to `keycloak/realm-export.json`

---

## Git Conventions

### Commit Messages

```
<type>: <short description>

<optional body with details>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`

Examples:
```
feat: add person duplicate detection endpoint
fix: correct campus filter on attendance list query
refactor: extract sync conflict resolver to separate package
test: add integration tests for triage creation
docs: update API design with consent endpoints
```

### Branch Naming

```
feature/person-registration
feature/offline-sync-engine
fix/duplicate-detection-case-sensitivity
chore/update-go-dependencies
```
