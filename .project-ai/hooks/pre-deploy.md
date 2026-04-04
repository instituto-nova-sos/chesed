# Hook: Pre-Deployment Gate Check

## Purpose

Validate all deployment prerequisites before pushing to any environment. Ensures the release is tested, documented, secure, and properly tagged before deployment.

## Trigger Condition

Before executing any deployment command — Docker image push, Kubernetes apply, or any environment deployment.

## Status

**Blocking** — Do not proceed with deployment if any gate fails.

## Steps

1. **Verify all tests pass**
   - Run full test suite (`make test-all`), not just changed files.
   - Zero test failures required.
   - If any test fails, STOP. Fix before deploying.

2. **Verify sprint-release checklist is complete**
   - Confirm all items in `checklists/sprint-release.md` are checked.
   - If any item is incomplete, STOP. Complete the checklist first.

3. **Verify database migrations are reversible**
   - For every `.up.sql` in the release, confirm a `.down.sql` exists.
   - Run migration down/up cycle in staging to verify reversibility.
   - If any migration is irreversible, document the risk and get tech-lead approval.

4. **Verify environment variables are documented**
   - Check `.env.example` contains all variables used in the code.
   - No new environment variables without documentation.
   - No secrets in source control.

5. **Verify Docker images build successfully**
   - Run `docker build` for backend and frontend.
   - Verify images start and pass health checks.
   - Run Trivy security scan — block on CRITICAL or HIGH findings.

6. **Verify no secrets in code or configuration**
   - Scan for hardcoded credentials, API keys, or tokens.
   - Verify `.env` is in `.gitignore`.
   - Verify no secrets in Docker build layers.

7. **Verify git tag exists**
   - Release must be tagged: `sprint-N-complete` or `vX.Y.Z`.
   - Tag must be on the commit being deployed.
   - If no tag exists, STOP. Tag the release first.

8. **Verify documentation is current**
   - API documentation matches implemented endpoints.
   - Data model documentation matches current schema.
   - Deployment documentation reflects current configuration.

## Enforcement Mechanism

- The AI agent must execute this hook before any deployment action.
- All steps must pass before deployment proceeds.
- If any step fails, the agent must report which gate failed and what action is needed.

## References

- `docs/14-deployment-strategy.md` — Deployment environments and procedures
- `docs/quality/quality-gates.md` — Overall Code Quality Gate
- `checklists/sprint-release.md` — Sprint release checklist

## Consequences of Skipping

- Untested code reaches production, causing user-facing failures.
- Irreversible migrations prevent rollback on failure.
- Missing environment variables cause runtime crashes in production.
- Untagged releases make rollback identification impossible.
- Secrets in code create security vulnerabilities.
