# Hook: Post-Deployment Verification

## Purpose

Validate deployment success through smoke tests, health checks, and configuration verification. Detect deployment issues before they impact users.

## Trigger Condition

After deployment to any environment completes (staging or production).

## Status

**Non-blocking** — but mandatory for release sign-off. Findings must be addressed before the release is considered complete.

## Steps

1. **Run health check**
   - `GET /api/v1/health` — must return 200 with service status.
   - Verify database connectivity reported as healthy.
   - Verify Keycloak connectivity reported as healthy.
   - If health check fails, initiate rollback immediately.

2. **Verify database migrations applied**
   - Check migration version matches expected version.
   - Run a simple read query to verify database is accessible.
   - Verify seed data is present (service types, default campus).

3. **Verify Keycloak realm is accessible**
   - `GET /realms/chesed/.well-known/openid-configuration` — must return valid OIDC metadata.
   - Verify all expected roles exist in the realm.
   - Verify client configurations are correct.

4. **Verify API endpoints respond**
   - Test authentication flow: obtain token from Keycloak, call protected endpoint.
   - Test a list endpoint: `GET /api/v1/persons` (should return 200 with empty or seeded data).
   - Test error handling: send invalid request, verify standard error format.

5. **Verify frontend loads**
   - Load the application URL in a browser/headless check.
   - Verify Keycloak login redirect works.
   - Verify the application renders after authentication.
   - Check browser console for JavaScript errors.

6. **Check application logs**
   - Review backend logs for startup errors or warnings.
   - Verify no panic, fatal, or error-level log entries on startup.
   - Verify structured logging (slog) is producing expected output.

7. **Generate deployment verification report**
   - Record all check results (PASS/FAIL).
   - Record deployment metadata (version, environment, timestamp).
   - If any check fails, document the failure for incident reporting.

## Enforcement Mechanism

- The AI agent must execute this hook after every deployment.
- Failures in health checks (Step 1) trigger immediate rollback via `rollback-and-hotfix` playbook.
- Other failures are documented and must be resolved before the release is signed off.

## References

- `docs/14-deployment-strategy.md` — Deployment procedures and monitoring
- `docs/20-keycloak-configuration.md` — Expected Keycloak configuration

## Consequences of Skipping

- Deployment failures go undetected until users report issues.
- Database migration failures corrupt data silently.
- Keycloak misconfigurations lock all users out.
- Frontend deployment failures result in blank pages.
