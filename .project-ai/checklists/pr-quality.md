# PR Quality Checklist

Use this checklist for every pull request before merge. It validates all quality gate conditions and clean code compliance. Every item must pass.

---

## Quality Gate Conditions

Per `docs/quality/quality-gates.md` — New Code Quality Gate:

### Reliability
- [ ] 0 new bugs — no unhandled errors, null dereferences, race conditions, or incorrect state transitions
- [ ] All errors handled and wrapped with context
- [ ] State transitions follow documented lifecycle rules
- [ ] Transactions used for multi-statement mutations

### Security
- [ ] 0 new vulnerabilities — no injection risks, missing auth, exposed PII, insecure patterns
- [ ] All security-sensitive code reviewed (auth, PII, crypto, external input)
- [ ] RBAC middleware on all new endpoints
- [ ] Campus isolation (`campus_id` filter) on all new queries
- [ ] No PII in logs or error responses
- [ ] No hardcoded secrets

### Coverage
- [ ] Test coverage on new code ≥ 80%
- [ ] Service layer methods have table-driven tests
- [ ] Error paths tested (not just happy path)
- [ ] Form validation and submission tested (frontend)

### Duplication
- [ ] Duplicated lines on new code ≤ 3%
- [ ] No copy-pasted logic — shared logic extracted

### Maintainability
- [ ] No function exceeds cognitive complexity threshold (Go: 25, TS: 15)
- [ ] No function exceeds cyclomatic complexity 10
- [ ] No function exceeds length threshold (Go: 40 lines, TS: 50 lines)
- [ ] No file exceeds length threshold (Go: 400 lines, TS: 300 lines)
- [ ] Nesting depth ≤ 3 levels
- [ ] Parameter count ≤ 5

---

## Clean Code Assessment

Per `docs/quality/clean-code-guidelines.md`:

- [ ] **Consistency**: Patterns match sibling files (naming, structure, error handling, imports)
- [ ] **Intentionality**: Names reveal purpose, no dead code, comments explain why not what
- [ ] **Adaptability**: Dependencies point inward, changes confined to appropriate layer
- [ ] **Responsibility**: Each function has a single responsibility, no layer violations

---

## Architecture Compliance

- [ ] Handler depends only on service (no repository imports)
- [ ] Service depends on repository interfaces (no pgx imports)
- [ ] Domain structs have zero external dependencies
- [ ] Components do not call API directly (use hooks)
- [ ] No circular dependencies

---

## Audit & Documentation

- [ ] Audit log entries created for all data mutations
- [ ] API changes reflected in `docs/11-api-design.md`
- [ ] Schema changes reflected in `docs/10-data-model.md`
- [ ] Domain changes reflected in `docs/04-domain-model.md`
- [ ] Commit messages follow convention: `<type>: <description>`

---

## Automated Checks

- [ ] All tests pass (Go + TypeScript)
- [ ] All linters pass (golangci-lint + ESLint)
- [ ] TypeScript strict mode — no `any` types
- [ ] No `_` for errors in Go code

---

## Verdict

```
ALL items pass → APPROVE (merge allowed)
ANY item fails → REQUEST_CHANGES (fix required)
```

---

## How to Use

Run this checklist for every PR. The reviewer agent uses this as the basis for quality gate evaluation.

```
Hook:    pre-merge (automated quality gate enforcement)
Skill:   review-code (comprehensive code review with quality profiles)
Agent:   reviewer (quality gate enforcer)
```
