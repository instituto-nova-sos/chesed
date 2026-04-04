# Playbook: Refactor for Quality

Unified playbook for resolving quality gate failures, reducing complexity, improving test coverage, and eliminating duplication. Use this when quality checks identify issues that must be fixed.

---

## When to Use

- Quality gate fails on a PR (any condition).
- `maintainability-analysis` skill identifies complexity or duplication issues.
- `reliability-validation` skill identifies error handling or state consistency gaps.
- Tech debt assessment reveals quality degradation.
- Sprint release blocked by overall code quality gate.

---

## Flow Overview

```
IDENTIFY → PRIORITIZE → REFACTOR → VERIFY → DOCUMENT
```

---

## Step 1: Identify Quality Gate Failures

Review the quality gate report and categorize failures:

| Category | Common Findings |
|----------|----------------|
| **Complexity** | Functions exceeding cognitive/cyclomatic thresholds, deep nesting, long functions |
| **Duplication** | Copy-pasted logic, similar code blocks across files |
| **Coverage** | Business logic without tests, untested error paths |
| **Reliability** | Unhandled errors, missing transactions, inconsistent state transitions |
| **Security** | Missing RBAC, exposed PII, insecure patterns |

---

## Step 2: Prioritize by Severity

Fix in this order:

1. **BLOCKER** — Quality gate fails. Must fix before merge.
2. **CRITICAL** — Security vulnerabilities, data corruption risks.
3. **MAJOR** — Significant quality problems that should be fixed before release.
4. **MINOR** — Improvements that can be batched.

---

## Step 3: Refactor by Category

### Reducing Complexity

**Extract Function/Method**: Break large functions into focused helpers.

```go
// Before: cognitive complexity 30+
func (s *TriageService) Create(ctx context.Context, input CreateTriageInput) (*Triage, error) {
    // 40+ lines of validation, creation, audit logging
}

// After: cognitive complexity < 25 each
func (s *TriageService) Create(ctx context.Context, input CreateTriageInput) (*Triage, error) {
    if err := s.validateTriageInput(ctx, input); err != nil {
        return nil, fmt.Errorf("triageService.Create: %w", err)
    }
    triage := s.buildTriage(input)
    if err := s.repo.Create(ctx, triage); err != nil {
        return nil, fmt.Errorf("triageService.Create: %w", err)
    }
    s.logAudit(ctx, "CREATE", "triage", triage.ID, nil, triage)
    return triage, nil
}
```

**Replace Nested Conditions with Guard Clauses**:

```go
// Before: nesting depth 4
func process(item Item) error {
    if item.IsValid {
        if item.HasPermission {
            if item.InScope {
                // do work
            }
        }
    }
    return nil
}

// After: nesting depth 1
func process(item Item) error {
    if !item.IsValid {
        return ErrInvalidItem
    }
    if !item.HasPermission {
        return ErrForbidden
    }
    if !item.InScope {
        return ErrOutOfScope
    }
    // do work
    return nil
}
```

**Replace Long Switch with Lookup Table**:

```go
// Before: cyclomatic complexity 8+
func getStatusLabel(status string) string {
    switch status {
    case "PENDING": return "Pendente"
    case "SCHEDULED": return "Agendado"
    // ... many cases
    }
}

// After: cyclomatic complexity 1
var statusLabels = map[string]string{
    "PENDING":   "Pendente",
    "SCHEDULED": "Agendado",
    // ...
}
func getStatusLabel(status string) string {
    return statusLabels[status]
}
```

**Extract React Component**: Split large components into sub-components.

```typescript
// Before: 200+ lines, JSX 120+ lines
function PersonDetailPage() {
    return (
        <div>
            {/* 120 lines of JSX */}
        </div>
    );
}

// After: focused components
function PersonDetailPage() {
    return (
        <div>
            <PersonHeader person={person} />
            <PersonContactInfo person={person} />
            <PersonRoles person={person} onAssign={handleAssign} />
            <PersonAttendanceHistory personId={person.id} />
        </div>
    );
}
```

### Improving Test Coverage

1. **Identify uncovered code paths**: Focus on service layer and handler error paths.
2. **Add table-driven tests** for missing scenarios:
   - Each validation rule should have a test case.
   - Each error branch should have a test case.
   - Each business rule should have at least one positive and one negative test.
3. **Add integration tests** for repository methods.
4. **Add form tests** for validation messages and submission behavior.

### Eliminating Duplication

1. **Identify duplicated logic**: Similar code blocks across files.
2. **Extract to appropriate layer**:
   - Shared domain logic → domain package utilities.
   - Shared handler logic → handler helpers (e.g., `writeJSON`, `writeError`).
   - Shared component logic → reusable components or hooks.
3. **Verify callers** still work after extraction.

### Fixing Reliability Issues

1. **Add error handling**: Replace `_` with proper error checks.
2. **Add error context**: Wrap errors with `fmt.Errorf("scope.Method: %w", err)`.
3. **Add transactions**: Wrap multi-statement mutations in database transactions.
4. **Fix state transitions**: Ensure invalid transitions are rejected.
5. **Add timeouts**: Configure timeouts on external calls.

---

## Step 4: Verify Quality Gates Pass

After refactoring:

1. **Run all tests**: `go test ./...` and `npm test`. Zero failures.
2. **Run linter**: `golangci-lint run` and ESLint. Zero warnings.
3. **Re-evaluate quality gate**: Run `review-code` skill with quality gate validation.
4. **Verify behavior preserved**: Existing tests still pass. No regressions.
5. **Run `refactoring` checklist**: `.project-ai/checklists/refactoring.md`.

---

## Step 5: Document Changes

1. Commit with clear message: `refactor: <description of what was improved>`.
2. If complexity thresholds were adjusted, update `docs/quality/complexity-guidelines.md` with ADR.
3. Update any affected documentation (API docs, data model).

---

## Safety Rules

- **Never change behavior during refactoring.** Tests must pass before AND after.
- **Refactor in small steps.** Each step should be independently verifiable.
- **Do not add features during refactoring.** Separate concerns: refactor first, then add features.
- **Keep the test suite green at every step.** If tests break, revert and try a smaller step.

---

## References

| Artifact | Path | Usage |
|----------|------|-------|
| Quality gates | `docs/quality/quality-gates.md` | Pass/fail criteria |
| Complexity guidelines | `docs/quality/complexity-guidelines.md` | Thresholds and refactoring examples |
| Clean code guidelines | `docs/quality/clean-code-guidelines.md` | Categories |
| Refactoring checklist | `.project-ai/checklists/refactoring.md` | Post-refactoring verification |
| Quality gates rule | `.project-ai/rules/quality-gates.md` | Enforcement rule |
