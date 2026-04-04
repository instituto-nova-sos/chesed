# Rule: API Versioning Strategy

## Purpose

Define when and how to version API endpoints. Prevent breaking changes to published endpoints and ensure backward compatibility for mobile/PWA clients that may not update immediately.

## Rule Statement

All API endpoints are versioned under `/api/v1/`. Breaking changes to existing endpoints require a new version path (`/api/v2/`), an API change proposal (template), and tech-lead approval. Non-breaking additions do not require versioning.

## Definitions

### Breaking Changes (Require New Version)

- Removing an existing endpoint.
- Removing a field from a response body.
- Renaming a field in a request or response body.
- Changing a field's type (e.g., string to number).
- Making an optional request field required.
- Changing the semantics of an existing field.
- Changing status codes for existing scenarios.
- Changing the authentication/authorization requirements.

### Non-Breaking Changes (Do NOT Require New Version)

- Adding a new endpoint.
- Adding a new optional field to a request body.
- Adding a new field to a response body.
- Adding a new query parameter (optional).
- Adding a new status code for a new scenario.
- Relaxing validation (e.g., increasing max length).
- Adding new enum values to an existing enum field.

## Process for Breaking Changes

1. **Identify**: Determine that the change is breaking per definitions above.
2. **Propose**: Fill the `api-change-proposal` template with:
   - Current behavior.
   - Proposed change.
   - Justification (why breaking change is necessary).
   - Migration plan for existing clients.
   - Deprecation timeline for the old version.
3. **Review**: Tech-lead approves or rejects the proposal.
4. **Implement**: Create new version endpoints alongside old ones.
5. **Deprecate**: Mark old version as deprecated (response header: `Deprecation: true`).
6. **Remove**: After deprecation period (minimum 1 sprint), remove old version.

## Trigger Condition

- Any modification to an existing endpoint's request or response schema.
- `pre-api-change` hook evaluates whether a change is breaking.
- `review-api-contract` skill validates versioning compliance.

## Enforcement Mechanism

- **Pre-api-change hook**: Checks if changes to existing endpoints are breaking.
- **Review-api-contract skill**: Validates that breaking changes use new version path.
- **Tech-lead agent**: Approves breaking change proposals.
- **Reviewer agent**: Verifies versioning compliance during PR review.

## MVP Phase Consideration

During Phase 1 (pre-production), breaking changes to v1 endpoints are acceptable without creating v2, since there are no external consumers. However:
- All changes must still be documented in `docs/11-api-design.md`.
- The `api-change-proposal` template should still be used for significant changes.
- This relaxation ends when the first production deployment occurs.

## Consequences of Skipping

- Mobile/PWA clients break when API changes without versioning.
- Offline-first clients with cached requests fail silently.
- No migration path for existing data consumers.
- Trust erosion with API stability commitments.

## References

- `docs/11-api-design.md` — API endpoint specifications
- `docs/12-offline-sync-strategy.md` — Offline client compatibility concerns
- `docs/05-architecture-proposal.md` — API design principles
