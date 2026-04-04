# Complexity Guidelines

This document defines measurable and configurable complexity parameters for the Chesed project. These parameters are enforced through quality gates, skills, and code review.

All thresholds are explicit, adjustable, and aligned with real-world tooling (`golangci-lint`, ESLint).

---

## Complexity Thresholds

### Backend (Go)

| Metric | Threshold | Tooling |
|--------|-----------|---------|
| Cognitive complexity per function | 25 | `golangci-lint` → `gocognit` |
| Cyclomatic complexity per function | 10 | `golangci-lint` → `gocyclo` |
| Function length | 40 lines | `golangci-lint` → `funlen` |
| File length | 400 lines | Manual review |
| Nesting depth | 3 levels | `golangci-lint` → `nestif` |
| Parameter count | 5 | Manual review |
| Return values | 3 | Manual review |

### Frontend (React/TypeScript)

| Metric | Threshold | Tooling |
|--------|-----------|---------|
| Cognitive complexity per function | 15 | ESLint → `sonarjs/cognitive-complexity` |
| Cyclomatic complexity per function | 10 | ESLint → `complexity` |
| Function length | 50 lines | ESLint → `max-lines-per-function` |
| File length | 300 lines | ESLint → `max-lines` |
| Nesting depth | 3 levels | ESLint → `max-depth` |
| Parameter count | 5 | ESLint → `max-params` |
| Component JSX lines | 80 lines | Manual review |

---

## Metric Definitions

### Cognitive Complexity

Measures how difficult a function is to understand. Unlike cyclomatic complexity, cognitive complexity penalizes nested control flow more heavily because nesting increases mental effort.

**Increments:**
- +1 for each `if`, `else if`, `else`, `switch`, `for`, `while`
- +1 for each `&&`, `||` in conditions (Go: `&&`, `||`)
- +1 nesting penalty for each level of nesting (nested `if` inside `for` adds more than a flat `if`)
- +1 for `break`, `continue`, `goto` to a label
- +1 for recursion

**Go example — compliant (cognitive complexity: 6):**
```go
func (s *PersonService) Create(ctx context.Context, p domain.Person) error {
    if err := s.validator.Struct(p); err != nil {        // +1
        return fmt.Errorf("PersonService.Create: %w", err)
    }
    existing, err := s.repo.FindByDocument(ctx, p.DocumentNumber)
    if err != nil {                                       // +1
        return fmt.Errorf("PersonService.Create: %w", err)
    }
    if existing != nil {                                  // +1
        return ErrDuplicateDocument
    }
    if err := s.repo.Create(ctx, p); err != nil {        // +1
        return fmt.Errorf("PersonService.Create: %w", err)
    }
    if err := s.audit.Log(ctx, "CREATE", "person", p.ID); err != nil {  // +1
        slog.Error("audit log failed", "error", err)
    }
    return nil
}
```

**Go example — violation (cognitive complexity: 28):**
```go
func processRecords(records []Record) []Result {
    var results []Result
    for _, r := range records {                          // +1
        if r.Type == "A" {                               // +1 (nesting: +1)
            if r.Status == "active" {                    // +1 (nesting: +2)
                if r.Value > 0 {                         // +1 (nesting: +3)
                    // deeply nested logic...
                } else {                                 // +1
                    for _, sub := range r.SubRecords {   // +1 (nesting: +4)
                        if sub.Valid {                    // +1 (nesting: +5)
                            // more nesting...
                        }
                    }
                }
            }
        } else if r.Type == "B" {                        // +1
            // another branch with similar nesting...
        } else {                                         // +1
            // ...
        }
    }
    return results
}
```

**How to fix:** Extract inner logic into well-named helper functions. Use early returns. Replace nested conditions with guard clauses.

### Cyclomatic Complexity

Counts the number of independent execution paths through a function. Each decision point adds one path.

**Decision points:**
- `if`, `else if` (Go: `if`, `else if`)
- `case` in `switch` (each `case` adds 1)
- `for`, `while`, `do-while` (Go: `for`)
- `&&`, `||` in conditions
- `catch` / error handling branch

**Threshold: 10** means a function should have at most 10 independent paths.

**How to fix:** Extract branches into separate functions. Use strategy patterns or lookup tables instead of long switch statements.

### Function Length

Lines of code in a function body, excluding blank lines and comments.

- **Go: 40 lines** — Go functions should be short and focused. If a function exceeds this, it likely has multiple responsibilities.
- **TypeScript: 50 lines** — Slightly higher to account for JSX rendering patterns, but still enforces focus.

**How to fix:** Extract logical sections into well-named helper functions. Each extracted function should have a single responsibility.

### File Length

Total lines in a file including imports, type definitions, and all functions.

- **Go: 400 lines** — If a file exceeds this, the package likely needs to be split or responsibilities redistributed.
- **TypeScript: 300 lines** — Enforces component and module focus.

**How to fix:** Split the file by responsibility. In Go, split into multiple files within the same package. In React, extract components or hooks into separate files.

### Nesting Depth

Maximum levels of nested control structures within a function.

**Threshold: 3 levels** for both Go and TypeScript.

**Level 0 (function body) → Level 1 (first if/for) → Level 2 (nested if/for) → Level 3 (maximum)**

**How to fix:**
- Use early returns (guard clauses) to reduce nesting
- Extract nested logic into helper functions
- Use `continue` in loops to skip invalid items early

**Go example — refactored from 4 levels to 2:**
```go
// Before: 4 levels of nesting
func process(items []Item) {
    for _, item := range items {
        if item.IsValid {
            if item.NeedsProcessing {
                if item.HasDependencies {
                    // process...
                }
            }
        }
    }
}

// After: 2 levels of nesting
func process(items []Item) {
    for _, item := range items {
        if !item.IsValid || !item.NeedsProcessing || !item.HasDependencies {
            continue
        }
        // process...
    }
}
```

### Parameter Count

Maximum number of parameters a function accepts.

**Threshold: 5** for both Go and TypeScript.

**How to fix:** Group related parameters into a struct (Go) or interface/object (TypeScript).

```go
// Before: 6 parameters
func CreatePerson(ctx context.Context, name, doc, email, phone string, campusID uuid.UUID) error

// After: 2 parameters (context + domain struct)
func CreatePerson(ctx context.Context, person domain.Person) error
```

### Return Values (Go only)

Maximum number of return values from a Go function.

**Threshold: 3**

Standard patterns: `(error)`, `(result, error)`, `(result, total, error)` for paginated queries.

**How to fix:** If returning more than 3 values, define a result struct.

### Component JSX Lines (React only)

Maximum lines of JSX in a component's return statement.

**Threshold: 80 lines**

**How to fix:** Extract sections of JSX into sub-components with descriptive names.

---

## Adjusting Thresholds

Thresholds are configurable. To adjust a threshold:

1. Document the change reason in an ADR (`.project-ai/templates/adr.md`)
2. Update the threshold in this file
3. Update the corresponding linter configuration (`golangci-lint` config or ESLint config)
4. Get Tech Lead approval

Thresholds should only be relaxed when there is a demonstrated technical reason. They should be tightened as the codebase matures.

---

## Linter Configuration Alignment

### Go (`golangci-lint`)

The following linters enforce complexity thresholds:

```yaml
linters:
  enable:
    - gocognit      # cognitive complexity
    - gocyclo        # cyclomatic complexity
    - funlen         # function length
    - nestif         # nesting depth

linters-settings:
  gocognit:
    min-complexity: 25
  gocyclo:
    min-complexity: 10
  funlen:
    lines: 40
    statements: 30
  nestif:
    min-complexity: 3
```

### TypeScript (ESLint)

```json
{
  "rules": {
    "complexity": ["error", 10],
    "max-depth": ["error", 3],
    "max-lines-per-function": ["error", { "max": 50, "skipBlankLines": true, "skipComments": true }],
    "max-lines": ["error", { "max": 300, "skipBlankLines": true, "skipComments": true }],
    "max-params": ["error", 5],
    "sonarjs/cognitive-complexity": ["error", 15]
  }
}
```

---

## References

| Document | Path |
|----------|------|
| Quality profiles | [`docs/quality/quality-profiles.md`](quality-profiles.md) |
| Quality gates | [`docs/quality/quality-gates.md`](quality-gates.md) |
| Clean code guidelines | [`docs/quality/clean-code-guidelines.md`](clean-code-guidelines.md) |
| Implementation guidelines | `docs/15-implementation-guidelines.md` |
