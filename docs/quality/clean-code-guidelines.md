# Clean Code Guidelines

This document defines four clean code categories that all Chesed code must satisfy: **Consistency**, **Intentionality**, **Adaptability**, and **Responsibility**. Each category includes definitions, good practices, violations, evaluation criteria, and maintainability impact.

These categories apply to both Backend (Go) and Frontend (React/TypeScript) code.

---

## 1. Consistency

### Definition

Code follows established patterns, conventions, and styles uniformly across the codebase. A developer reading any file can predict the structure, naming, and patterns used in other files.

### Good Practices

#### Go
- All handlers follow the same signature pattern: `func (h *Handler) Method(w http.ResponseWriter, r *http.Request)`
- All services accept `context.Context` as the first parameter
- All errors wrapped with `fmt.Errorf("package.Method: %w", err)`
- All repository methods return `(result, error)` — never panic
- All constructors follow `NewXxx(deps) *Xxx` pattern
- File naming: `person_handler.go`, `person_service.go`, `person_repository.go`

#### React/TypeScript
- All components are functional with explicit props interfaces
- All custom hooks prefixed with `use` and return consistent shapes (data, loading, error)
- All API client functions follow the same pattern: `async function getEntity(id: string): Promise<Entity>`
- All forms use React Hook Form with Zod resolver
- All pages follow the same layout structure with loading and error states
- File naming: `PersonForm.tsx`, `usePersons.ts`, `personApi.ts`

### Violation Examples

- A handler that returns errors differently from other handlers
- A service method that does not accept `context.Context`
- A component using class-based React when all others are functional
- Inconsistent error response shapes across endpoints
- Mixed naming styles (camelCase and snake_case in the same layer)
- Some hooks returning `{ data, error }` while others return `[data, error]`

### Evaluation Criteria

- [ ] Naming conventions match the profile (see quality profiles)
- [ ] Error handling follows the established pattern throughout the file
- [ ] File structure matches the pattern used by sibling files in the same package/directory
- [ ] Import ordering is consistent
- [ ] Function signatures follow established patterns for that layer

### Impact on Maintainability

Inconsistency forces developers to context-switch between patterns, increasing cognitive load. It creates uncertainty about which pattern is "correct," leading to further inconsistency. Consistent code enables faster onboarding and safer modifications.

---

## 2. Intentionality

### Definition

Code clearly communicates its purpose. Names, structures, and organization reveal what the code does and why, without requiring comments to explain. Every element exists for a reason.

### Good Practices

#### Go
- Function names describe the action: `CreatePerson`, `FindByDocumentNumber`, `ValidateTriageTransition`
- Variable names reveal meaning: `campusID` not `id`, `activePersons` not `list`
- Package names describe the domain: `handler`, `service`, `repository`, `domain`
- Error variables describe the failure: `ErrPersonNotFound`, `ErrDuplicateDocument`
- Boolean variables read as assertions: `isActive`, `hasPermission`, `canTransition`

#### React/TypeScript
- Component names describe what they render: `PersonForm`, `TriageStatusBadge`, `AttendanceTimeline`
- Hook names describe what they provide: `usePersons`, `useTriageTransition`, `useOfflineSync`
- Props describe the component's interface: `onSubmit`, `initialValues`, `isLoading`
- Type names describe the data shape: `PersonCreateRequest`, `TriageListResponse`
- Event handlers describe the action: `handlePersonSubmit`, `handleStatusChange`

### Violation Examples

- Generic names: `data`, `result`, `temp`, `item`, `list`, `obj`, `val`
- Abbreviated names: `ps` instead of `personService`, `ctx` is acceptable only for `context.Context`
- Functions that do more than their name suggests (side effects not reflected in the name)
- Dead code left in the file "just in case"
- Comments explaining what the code does instead of the code explaining itself
- Boolean flags that require reading the implementation to understand: `flag`, `status`, `mode`

### Evaluation Criteria

- [ ] Function/method names are verb-noun pairs that describe the action
- [ ] Variable names reveal domain meaning, not implementation detail
- [ ] No dead code (commented-out blocks, unused functions, unreachable branches)
- [ ] No comments that restate what the code does — comments explain _why_ only
- [ ] Types and interfaces named after the domain concept they represent

### Impact on Maintainability

Unclear naming forces developers to read entire implementations to understand purpose. Dead code creates false leads during debugging. When code communicates intent, modifications are safer because the developer understands what should change and what should not.

---

## 3. Adaptability

### Definition

Code is structured to accommodate change without requiring widespread modifications. Changes in one area have minimal ripple effects. The architecture supports extension without modification of existing code.

### Good Practices

#### Go
- Interfaces defined at the consumption site — changing the implementation does not change the consumer
- Dependency injection via constructors — swapping implementations requires changing only the wiring
- Repository pattern isolates database access — changing query logic does not affect business logic
- Domain structs have no external dependencies — domain rules are portable
- Configuration via environment variables — no hardcoded values that differ per environment

#### React/TypeScript
- Components receive behavior through props, not hardcoded logic
- API layer isolated in `api/` — changing endpoints requires modifying one file per entity
- Hooks encapsulate state management — components do not know about stores or API calls
- Offline storage isolated in `offline/` — changing sync strategy does not affect UI components
- Zod schemas co-located with types — validation changes require one file change

### Violation Examples

- Business logic in a handler (changing business rules requires modifying HTTP layer)
- Direct `pgx` calls in a service (changing database requires modifying business logic)
- Component with hardcoded API calls (changing the API requires modifying UI code)
- Deeply coupled modules that require shotgun surgery for any change
- Magic strings scattered across multiple files instead of constants
- Configuration values hardcoded instead of externalized

### Evaluation Criteria

- [ ] Changes to database access do not require changes to service logic
- [ ] Changes to business rules do not require changes to HTTP handler logic
- [ ] Changes to API contracts do not require changes to UI components (hooks absorb the change)
- [ ] Adding a new entity follows the same structural pattern without modifying existing code
- [ ] External dependencies (Keycloak, PostgreSQL) are abstracted behind interfaces

### Impact on Maintainability

Rigid code makes every change expensive and risky. When a simple requirement change requires modifications across 10 files, the system is fragile. Adaptable code confines changes to the appropriate layer, reducing both effort and risk of regression.

---

## 4. Responsibility

### Definition

Each code unit (function, struct, component, module) has a single, well-defined responsibility. It does one thing well and delegates other concerns to appropriate units.

### Good Practices

#### Go
- Handlers: parse request, validate input, call service, write response — nothing else
- Services: business logic and validation rules — no HTTP concerns, no SQL
- Repositories: SQL execution and row scanning — no business logic
- Domain: data structures — no behavior, no dependencies
- Middleware: cross-cutting concerns (auth, audit, CORS) — no business logic

#### React/TypeScript
- Pages: composition of components + hooks — no business logic or API calls
- Components: rendering UI based on props — no API calls, no store access
- Hooks: state management and API orchestration — no rendering
- API client: HTTP requests — no state management or UI logic
- Offline module: IndexedDB operations — no UI concerns

### Violation Examples

- Handler that contains SQL queries (handler + repository responsibility)
- Service that writes HTTP responses (service + handler responsibility)
- Component that fetches data directly from API (component + hook responsibility)
- A function that validates input AND saves to database AND sends notifications
- Middleware that contains business rules specific to one endpoint
- A 200-line function that handles multiple unrelated operations

### Evaluation Criteria

- [ ] Each function/method has a single reason to change
- [ ] Functions under complexity threshold (see quality profiles)
- [ ] No layer violations (handler importing repository, component calling API directly)
- [ ] Middleware contains only cross-cutting concerns
- [ ] Test files can test the unit in isolation without mocking unrelated concerns

### Impact on Maintainability

Multi-responsibility code is the primary source of complexity. When a function handles HTTP parsing, business validation, database access, and audit logging, any change to one concern risks breaking the others. Single-responsibility code is easier to test, easier to understand, and safer to modify.

---

## Evaluation Matrix

Use this matrix when reviewing code. Each file should be assessed against all four categories.

| Category | Key Question | Passing Criteria |
|----------|-------------|-----------------|
| Consistency | Does this file follow the same patterns as sibling files? | Yes — naming, structure, error handling, imports all match established patterns |
| Intentionality | Can a new developer understand this code without asking the author? | Yes — names reveal purpose, no dead code, comments explain why not what |
| Adaptability | Can the most likely change be made by modifying only this layer? | Yes — dependencies point inward, external concerns are abstracted |
| Responsibility | Does each function/component do exactly one thing? | Yes — no layer violations, no multi-concern functions, complexity within limits |

### Severity Classification

| Severity | Criteria |
|----------|---------|
| BLOCKER | Layer violation, multi-responsibility function > 2x complexity threshold, dead code hiding bugs |
| MAJOR | Inconsistent patterns across a package, unclear naming requiring investigation, tight coupling requiring shotgun surgery |
| MINOR | Naming could be more descriptive, slight inconsistency with sibling files, minor duplication |
| SUGGESTION | Alternative structure that would improve clarity, opportunity for better abstraction |

---

## References

| Document | Path |
|----------|------|
| Quality profiles | [`docs/quality/quality-profiles.md`](quality-profiles.md) |
| Complexity guidelines | [`docs/quality/complexity-guidelines.md`](complexity-guidelines.md) |
| Implementation guidelines | `docs/15-implementation-guidelines.md` |
