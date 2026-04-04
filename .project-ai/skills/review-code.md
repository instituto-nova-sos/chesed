# Skill: Review Code

## Purpose

Perform a code review with Chesed-specific quality standards covering backend Go patterns, frontend React/TypeScript patterns, architecture layering rules, naming conventions, and project-specific constraints defined in CLAUDE.md.

## When to Use / Trigger

- Before merging any PR.
- When a user says "review this code" or "code review for feature X".
- After implementing a feature, as a self-review step.
- Invoked by tech-lead agent as a quality gate.

## Role / Expertise

Senior developer familiar with:
- Go idioms: error handling, context propagation, interfaces at consumption site.
- React/TypeScript: strict mode, functional components, hook patterns.
- Clean architecture: layer separation, dependency direction.
- Chesed-specific rules from CLAUDE.md.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Files to review | Yes | File paths or git diff |
| Architecture rules | Yes | CLAUDE.md, `docs/15-implementation-guidelines.md` |
| Quality profile | Yes | `docs/quality/quality-profiles.md` |
| Complexity thresholds | Yes | `docs/quality/complexity-guidelines.md` |
| Clean code guidelines | Yes | `docs/quality/clean-code-guidelines.md` |
| Quality gates | Yes | `docs/quality/quality-gates.md` |

## Process

### Backend Go Checks

#### Error Handling
- [ ] No `_` used for error returns. Every error is handled.
- [ ] Errors wrapped with context: `fmt.Errorf("personService.Create: %w", err)`.
- [ ] Custom domain errors defined where appropriate (e.g., `ErrNotFound`, `ErrDuplicate`).
- [ ] Errors propagated up the stack, not swallowed silently.

#### Context Propagation
- [ ] All I/O functions accept `context.Context` as first parameter.
- [ ] Context passed from handler through service to repository.
- [ ] No `context.Background()` in production code (only in tests and main).

#### Interface Design
- [ ] Interfaces defined at the consumption site (in the `service` package, not `repository` package).
- [ ] Interfaces are minimal (only methods the consumer needs).
- [ ] No interface pollution (avoid interfaces with single implementations unless for testing).

#### Dependency Direction
- [ ] `handler` depends on `service` only (never imports `repository` or `pgx`).
- [ ] `service` depends on repository interfaces (defined in `service` package) and `domain`.
- [ ] `repository` depends on `domain` and `pgx` only.
- [ ] `domain` has zero imports from other internal packages.
- [ ] `middleware` depends on `config` and optionally `service`.
- [ ] No circular dependencies.

#### Naming Conventions
- [ ] Package names: lowercase, single word (e.g., `handler`, `service`, `domain`).
- [ ] Exported types: PascalCase.
- [ ] Unexported: camelCase.
- [ ] File names: snake_case (e.g., `person_service.go`, `person_handler.go`).
- [ ] Test files: `*_test.go` in same package.
- [ ] Constructor functions: `NewXxx(deps) *Xxx`.

#### Logging
- [ ] Uses `slog` (not `log`, `fmt.Println`, or third-party loggers).
- [ ] Structured fields: `slog.String("key", value)`.
- [ ] No PII in log messages (no document_number, phone, email, health data).
- [ ] Log at appropriate levels: Error for failures, Warn for degraded paths, Info for operations, Debug for development.

#### Database Access
- [ ] Uses pgx (not `database/sql`).
- [ ] Parameterized queries (no string concatenation for SQL).
- [ ] All queries include `campus_id` filter.
- [ ] `pgx.CollectRows` for list queries.
- [ ] Transactions used for multi-statement mutations.

### Frontend React/TypeScript Checks

#### TypeScript Strictness
- [ ] No `any` type usage.
- [ ] No `@ts-ignore` or `@ts-nocheck` comments.
- [ ] Explicit return types on exported functions.
- [ ] Union types for known string values (e.g., `'CPF' | 'SSN' | 'EU_ID'`).

#### Component Quality
- [ ] Functional components only (no class components).
- [ ] Components under 150 lines. If larger, sub-components must be extracted.
- [ ] No direct API calls in components (use hooks from `hooks/`).
- [ ] Props interface defined and exported.
- [ ] Destructured props in function signature.

#### Hooks
- [ ] Custom hooks prefixed with `use` (e.g., `usePersons`, `useAuth`).
- [ ] No side effects outside `useEffect`.
- [ ] Dependencies array complete and correct in `useEffect`.
- [ ] Loading and error states handled.

#### Forms
- [ ] React Hook Form with Zod resolver.
- [ ] Validation schema co-located with the type definition.
- [ ] Error messages in Portuguese (pt-BR).
- [ ] Accessible form fields (labels, aria attributes).

#### Styling
- [ ] Tailwind CSS classes only (no CSS modules, styled-components, or inline styles via `style=`).
- [ ] Mobile-first breakpoints (`sm:`, `md:`, `lg:` for larger screens).
- [ ] Responsive at 320px width.

#### Authentication
- [ ] keycloak-js adapter used for auth (no custom login forms).
- [ ] Auth context provides user info (email, role, campus_id).
- [ ] API client auto-attaches Bearer token.
- [ ] Protected routes check authentication.

### Architecture Layer Checks

- [ ] No handler importing from repository package.
- [ ] No component importing from API directly (must go through hooks).
- [ ] No domain struct importing from any other internal package.
- [ ] No frontend code in backend directory or vice versa.
- [ ] No Phase 2 features implemented in Phase 1 code.

### Code Hygiene

- [ ] No hardcoded secrets, API keys, or credentials.
- [ ] No global mutable state or singletons.
- [ ] No `TODO` or `FIXME` without a linked issue.
- [ ] No commented-out code blocks.
- [ ] All code comments in English.
- [ ] All UI text in Portuguese (pt-BR).
- [ ] Variable names in English.

### Quality Profile Compliance

Evaluate code against the applicable quality profile from `docs/quality/quality-profiles.md`:

#### Complexity Check
- [ ] Cognitive complexity within threshold (Go: 25, TS: 15). See `docs/quality/complexity-guidelines.md`.
- [ ] Cyclomatic complexity ≤ 10 per function.
- [ ] Function length within threshold (Go: 40 lines, TS: 50 lines).
- [ ] File length within threshold (Go: 400 lines, TS: 300 lines).
- [ ] Nesting depth ≤ 3 levels.
- [ ] Parameter count ≤ 5.
- [ ] Return values ≤ 3 (Go only).
- [ ] Component JSX ≤ 80 lines (React only).

#### Duplication Check
- [ ] No copy-pasted logic within the changed files.
- [ ] No duplication of logic already present elsewhere in the codebase.
- [ ] Duplicated lines on new code ≤ 3%.

#### Clean Code Assessment

Evaluate against all four categories from `docs/quality/clean-code-guidelines.md`:

- [ ] **Consistency**: Patterns match sibling files in the same package/directory.
- [ ] **Intentionality**: Names reveal purpose, no dead code, comments explain why not what.
- [ ] **Adaptability**: Changes in one area have minimal ripple effects, dependencies point inward.
- [ ] **Responsibility**: Each function/component has a single, well-defined responsibility.

#### Quality Gate Validation

After evaluating all checks above, render a quality gate verdict per `docs/quality/quality-gates.md`:

- [ ] 0 new bugs (reliability issues).
- [ ] 0 new vulnerabilities (security issues).
- [ ] Coverage on new code ≥ 80%.
- [ ] Maintainability rating = A.
- [ ] Reliability rating = A.
- [ ] Security rating = A.

## Outputs / Deliverables

A review report with:

1. **Summary**: Files reviewed, issues found by severity.
2. **Quality Gate**: PASS / FAIL with condition-by-condition results.
3. **Issues** (per file):
   - Severity: BLOCKER / MAJOR / MINOR / SUGGESTION.
   - Category: error-handling, architecture, naming, security, style, complexity, duplication, clean-code.
   - Location: file path, line range.
   - Description: what is wrong and why.
   - Fix: specific correction.
4. **Clean Code Assessment**: Pass/fail per category (Consistency, Intentionality, Adaptability, Responsibility).
5. **Complexity Report**: Functions exceeding thresholds with current values.
6. **Positive observations**: Well-written code worth noting.
7. **Verdict**: APPROVE / REQUEST_CHANGES / NEEDS_DISCUSSION.

## References

| Document | Path | Usage |
|----------|------|-------|
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Coding patterns |
| Project rules | `CLAUDE.md` | Non-negotiable constraints |
| Quality profiles | `docs/quality/quality-profiles.md` | Stack-specific quality standards |
| Complexity guidelines | `docs/quality/complexity-guidelines.md` | Measurable thresholds |
| Clean code guidelines | `docs/quality/clean-code-guidelines.md` | Clean code categories |
| Quality gates | `docs/quality/quality-gates.md` | Pass/fail criteria |

## Constraints / Quality Bar

- BLOCKER issues must be fixed before merge.
- Any `_` for errors in Go code is an automatic BLOCKER.
- Any `any` type in TypeScript is an automatic BLOCKER.
- Any handler importing repository is an automatic BLOCKER.
- Any missing campus_id filter is an automatic BLOCKER.
- Components over 150 lines without extraction is a MAJOR issue.

## Interaction with Other Artifacts

- **Invoked by agents**: tech-lead (quality gate), security-engineer (code patterns).
- **Used alongside skills**: review-api-contract, review-security, review-migration.
- **Blocks**: assess-release-readiness (BLOCKER issues block release).
