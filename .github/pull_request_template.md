## Summary

<!-- 1-3 sentences describing what changed and why. Link the relevant doc/issue. -->

## Type

- [ ] feat
- [ ] fix
- [ ] refactor
- [ ] test
- [ ] docs
- [ ] chore
- [ ] ci

## Quality Gate Checklist

Per `.project-ai/hooks/pre-merge.md` and `docs/quality/quality-gates.md` — every box must be checked or have an explicit reason recorded below.

**Automated (CI enforces)**
- [ ] `Backend` workflow green (build, vet, golangci-lint, migrations, tests, coverage floor)
- [ ] `Frontend` workflow green (typecheck, lint, tests, coverage report, build)
- [ ] `Security` workflow green (gitleaks, govulncheck, npm audit)
- [ ] No new TODO/FIXME without an issue link

**Manual (reviewer enforces)**
- [ ] **Bugs**: 0 new reliability issues introduced
- [ ] **Vulnerabilities**: 0 new security issues; PII not logged
- [ ] **Coverage on new code**: ≥ 80% (Sprint 3+ enforced via diff-coverage)
- [ ] **Duplication on new code**: ≤ 3%
- [ ] **Complexity**: no function exceeds CLAUDE.md thresholds (Go cognitive 25 / TS cognitive 15 / cyclomatic 10)
- [ ] **Architecture**: handler → service → repository direction preserved; no DB drivers in handlers
- [ ] **Campus scoping**: all queries/mutations filtered by `campus_id` from auth context
- [ ] **Audit logging**: all mutations write to `audit_log`
- [ ] **Keycloak**: no custom credential handling added
- [ ] **Docs updated**: `04-domain-model`, `10-data-model`, `11-api-design` reflect changes
- [ ] **HANDOFF.md updated** with session entry

## Migrations

- [ ] No migrations OR each migration has both `.up.sql` and `.down.sql` files
- [ ] Tested up and down locally

## Test Plan

<!-- Bulleted checklist of what was tested. Include reproduction steps for any bug fix. -->

## Notes for Reviewer

<!-- Anything reviewer should look at first, known limitations, follow-ups, deferred items. -->
