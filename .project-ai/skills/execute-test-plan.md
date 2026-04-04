# Skill: Execute Test Plan

## Purpose

Given a test plan (from `design-test-plan` skill), validate that all specified tests exist, are correctly implemented, and pass. Identify gaps between the test plan and actual test files. Produces a test execution report with planned vs. actual coverage.

## When to Use / Trigger

- After implementation is complete and tests are written.
- During the VERIFY phase of feature delivery.
- When a user says "validate tests for feature X" or "check test coverage against plan".
- Before invoking the `pre-review` hook.

## Role / Expertise

Senior QA engineer who validates test completeness against planned specifications, identifies coverage gaps, and ensures test quality meets project standards.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Test plan document | Yes | Output from `design-test-plan` skill |
| Actual test files | Yes | `backend/**/*_test.go`, `frontend/src/**/*.test.{ts,tsx}` |
| Test execution results | Yes | `go test` / `npm test` output |
| Coverage report | Optional | `go test -coverprofile` / Vitest coverage |

## Process

### Step 1: Parse Test Plan

1. Read the test plan and enumerate all expected test cases.
2. Organize by category: unit tests, integration tests, security tests, edge cases.
3. Record the expected test count per category.

### Step 2: Map to Actual Tests

1. Search the codebase for test files related to the feature.
2. For each expected test case, find the corresponding test function:
   - Go: `func Test{Name}(t *testing.T)` with matching sub-test names.
   - React: `describe('{context}', () => { it('{expectation}', ...) })`.
3. Record mapping: planned test → actual test function + file path.

### Step 3: Identify Gaps

1. **Missing tests**: Planned test cases with no corresponding implementation.
   - Classify severity: BLOCKER (core business logic), MAJOR (edge cases), MINOR (nice-to-have).
2. **Bonus tests**: Implemented tests not in the plan (acceptable, record for documentation).
3. **Misaligned tests**: Tests that exist but don't assert what the plan specified.

### Step 4: Validate Test Execution

1. Run the test suite and capture results.
2. For each planned test, record: PASS, FAIL, or MISSING.
3. For failing tests, capture the error message and stack trace.
4. Identify flaky tests (tests that sometimes pass, sometimes fail).

### Step 5: Analyze Coverage

1. If coverage reports are available, analyze per-file and per-function coverage.
2. Compare against thresholds:
   - Service layer: ≥ 90%
   - Handler layer: ≥ 80%
   - Repository layer: integration tests for all CRUD operations
   - Frontend hooks: ≥ 80%
3. Identify uncovered code paths that correspond to planned but missing tests.

### Step 6: Generate Report

Produce a structured test execution report:

```markdown
## Test Execution Report

### Summary
- **Planned tests**: N
- **Implemented tests**: M
- **Missing tests**: N - M
- **Passing**: X
- **Failing**: Y
- **Coverage**: Z%

### Gap Analysis

| Planned Test | Status | File | Function | Severity |
|-------------|--------|------|----------|----------|
| Create person happy path | PASS | person_service_test.go | TestPersonService_Create/valid_person | — |
| Duplicate detection | MISSING | — | — | BLOCKER |
| Campus isolation | PASS | person_repo_test.go | TestPersonRepo_GetByID/wrong_campus | — |

### Failing Tests
[Details of each failing test with error messages]

### Missing Tests (Action Required)
[List of tests that need to be implemented, with suggested test structure]

### Verdict
[PASS: All planned tests implemented and passing]
[FAIL: Missing or failing tests exist — action required]
```

## Outputs / Deliverables

1. **Test execution report** with planned vs. actual comparison.
2. **Gap list** with severity classification (BLOCKER, MAJOR, MINOR).
3. **Coverage analysis** per layer compared to thresholds.
4. **Verdict**: PASS (all tests present and passing) or FAIL (gaps exist).
5. **Recommendations** for missing tests (suggested test structure and assertions).

## References

| Document | Path | Usage |
|----------|------|-------|
| Quality gates | `docs/quality/quality-gates.md` | Coverage thresholds |
| Quality profiles | `docs/quality/quality-profiles.md` | Testing requirements per stack |
| Security test strategy | `docs/17-security-test-strategy.md` | Security test patterns |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Test conventions |

## Constraints / Quality Bar

- Must not mark a test plan as satisfied if any BLOCKER-severity test is missing.
- Must report exact file paths and function names for all gaps.
- Must distinguish between missing tests and failing tests — different remediation actions.
- Must validate that tests assert the planned behavior, not just touch the code path.
- Coverage thresholds are non-negotiable per `docs/quality/quality-gates.md`.

## Interaction with Other Artifacts

- **Invoked by agents**: qa-engineer (primary), tech-lead (quality oversight).
- **Depends on skills**: design-test-plan (produces the test plan being validated).
- **Feeds into hooks**: post-implement (test verification step), pre-review (quality gate input).
- **Governed by rules**: test-coverage-enforcement (layer-specific thresholds), quality-gates (overall coverage).
