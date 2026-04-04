# Skill: Maintainability Analysis

## Purpose

Analyze code for maintainability issues including coupling, cohesion, complexity, duplication, and naming quality. Produces a maintainability assessment with a rating and actionable refactoring recommendations.

## When to Use / Trigger

- After implementing a feature, before requesting review.
- When a quality gate flags maintainability issues.
- When tech debt needs assessment.
- When refactoring is being considered — to identify where to focus effort.
- Invoked by reviewer agent during PR review.

## Role / Expertise

Senior developer with expertise in:
- Code complexity analysis (cognitive and cyclomatic complexity).
- Software design principles (SOLID, clean architecture).
- Refactoring patterns (extract method, replace conditional, introduce parameter object).
- Go and React/TypeScript idioms.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| Files to analyze | Yes | File paths or git diff |
| Quality profile | Yes | `docs/quality/quality-profiles.md` |
| Complexity thresholds | Yes | `docs/quality/complexity-guidelines.md` |
| Clean code guidelines | Yes | `docs/quality/clean-code-guidelines.md` |

## Process

### 1. Complexity Assessment

For each function in the changed files:

- [ ] Calculate cognitive complexity. Flag if > threshold (Go: 25, TS: 15).
- [ ] Calculate cyclomatic complexity. Flag if > 10.
- [ ] Check function length. Flag if > threshold (Go: 40 lines, TS: 50 lines).
- [ ] Check nesting depth. Flag if > 3 levels.
- [ ] Check parameter count. Flag if > 5.
- [ ] Check return values (Go only). Flag if > 3.
- [ ] Check component JSX lines (React only). Flag if > 80 lines.

### 2. Duplication Detection

- [ ] Identify code blocks that are structurally similar across the changed files.
- [ ] Identify code blocks in changed files that duplicate logic already present elsewhere in the codebase.
- [ ] Calculate duplication percentage for the new/changed code.
- [ ] Flag if duplication > 3% (new code threshold).

### 3. Coupling Analysis

- [ ] Verify dependency direction follows architecture layers (handler → service → repository → domain).
- [ ] Check for circular dependencies between packages/modules.
- [ ] Identify functions with too many external dependencies (high afferent coupling).
- [ ] Flag any layer violations (e.g., handler importing repository).

### 4. Cohesion Assessment

- [ ] Verify each file/module has a single, focused responsibility.
- [ ] Check that all functions in a file serve the same purpose.
- [ ] Flag files with unrelated functions grouped together.
- [ ] Flag functions with multiple responsibilities (doing more than their name suggests).

### 5. Naming Quality

- [ ] Verify function names are verb-noun pairs describing the action.
- [ ] Verify variable names reveal domain meaning.
- [ ] Verify types/interfaces named after domain concepts.
- [ ] Flag generic names: `data`, `result`, `temp`, `item`, `list`, `obj`, `val`.
- [ ] Flag abbreviated names (except `ctx` for `context.Context`, `err` for `error`).

### 6. Clean Code Categories

Evaluate against all four categories from `docs/quality/clean-code-guidelines.md`:

- [ ] **Consistency**: Patterns match sibling files in the same package/directory.
- [ ] **Intentionality**: Code communicates purpose without requiring comments to explain.
- [ ] **Adaptability**: Changes in one area have minimal ripple effects.
- [ ] **Responsibility**: Each function/component has a single, well-defined responsibility.

### 7. Refactoring Recommendations

For each issue found, provide:

- **Issue**: What was found and where.
- **Impact**: Why it matters (complexity, duplication, coupling).
- **Recommendation**: Specific refactoring pattern to apply.
- **Priority**: HIGH (blocks quality gate), MEDIUM (should fix before release), LOW (improve when convenient).

Common refactoring patterns:
- **Extract Method/Function**: Break large functions into focused helpers.
- **Extract Component**: Split large React components into sub-components.
- **Introduce Parameter Object**: Replace long parameter lists with structs/interfaces.
- **Replace Conditional with Guard Clause**: Reduce nesting with early returns.
- **Extract Interface**: Decouple consumers from implementations.
- **Move Method**: Relocate logic to the appropriate layer.
- **Consolidate Duplicate**: Extract shared logic into a common function.

## Outputs / Deliverables

A maintainability report with:

1. **Maintainability Rating**: A through E (per rating definitions in `docs/quality/quality-gates.md`).
2. **Complexity Summary**: Functions exceeding thresholds, with current values and thresholds.
3. **Duplication Summary**: Duplicated blocks, percentage, and extraction opportunities.
4. **Coupling/Cohesion**: Layer violations, circular dependencies, unfocused modules.
5. **Naming Issues**: Generic or unclear names with suggested improvements.
6. **Clean Code Assessment**: Pass/fail per category with specific findings.
7. **Refactoring Recommendations**: Prioritized list of actions with specific patterns.
8. **Quality Gate Impact**: Whether the issues would cause a quality gate failure.

## References

| Document | Path | Usage |
|----------|------|-------|
| Quality profiles | `docs/quality/quality-profiles.md` | Thresholds per stack |
| Complexity guidelines | `docs/quality/complexity-guidelines.md` | Metric definitions |
| Clean code guidelines | `docs/quality/clean-code-guidelines.md` | Category evaluation |
| Quality gates | `docs/quality/quality-gates.md` | Pass/fail criteria |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Coding patterns |

## Constraints / Quality Bar

- Any function exceeding 2x the complexity threshold is a BLOCKER.
- Any layer violation is a BLOCKER.
- Duplication above 3% on new code is a MAJOR issue.
- At least one refactoring recommendation must be provided for every MAJOR or BLOCKER finding.

## Interaction with Other Artifacts

- **Invoked by agents**: reviewer (PR review), tech-lead (architecture review).
- **Used alongside skills**: review-code (broader review), reliability-validation, review-security.
- **Triggers playbook**: refactor-for-quality (when significant issues found).
- **Blocks**: pre-merge hook (BLOCKER issues block merge).
