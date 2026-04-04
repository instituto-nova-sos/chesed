# Playbook: Onboard New Feature Domain

Step-by-step guide for introducing a new domain area into the project (e.g., when transitioning from Phase 1 to Phase 2, adding campaign management, donation tracking, or document management).

---

## When to Use

- Transitioning to a new project phase (Phase 1 → Phase 2).
- Adding a new business domain that requires new tables, endpoints, and UI.
- Introducing a new bounded context that doesn't fit existing domain areas.

---

## Flow Overview

```
DOCUMENT DOMAIN ──> DEFINE REQUIREMENTS ──> DESIGN SCHEMA ──> DESIGN API ──>
    ──> UPDATE SCOPE ──> CREATE STORIES ──> ASSESS CROSS-CUTTING ──> UPDATE RULES
```

---

## Step 1: Document the Domain

Update `docs/04-domain-model.md`:

1. Define new domain entities with attributes and relationships.
2. Define entity lifecycle (states, transitions) if applicable.
3. Define business rules specific to this domain.
4. Identify relationships to existing entities (person, campus, etc.).
5. Define the domain vocabulary (ubiquitous language).

---

## Step 2: Define Requirements

Update `docs/03-requirements-catalog.md`:

1. Add new functional requirements with RF-XX codes.
2. Add any new non-functional requirements with RNF-XX codes.
3. Define acceptance criteria for each requirement.
4. Map requirements to the new domain entities.
5. Validate requirements with stakeholders before proceeding.

---

## Step 3: Design Database Schema

Use the `design-database-schema` skill:

1. Design tables for the new domain entities.
2. Define foreign key relationships to existing tables.
3. Apply standard columns (UUID PK, campus_id, is_active, timestamps).
4. Design indexes based on expected query patterns.
5. Plan migration sequence with dependency ordering.
6. Update `docs/10-data-model.md` with the new schema.

---

## Step 4: Design API

Use the `design-api-contract` skill:

1. Design REST endpoints for the new domain resources.
2. Follow existing API conventions from `docs/11-api-design.md`.
3. Define request/response schemas matching the data model.
4. Assign RBAC roles per endpoint.
5. Define pagination, filtering, and search capabilities.
6. Update `docs/11-api-design.md` with the new endpoints.

---

## Step 5: Update Scope Documentation

Update `docs/07-mvp-scope.md`:

1. Move features from "Won't Have" to the appropriate phase.
2. Define the new phase scope clearly.
3. Update the phase-boundary table lists.

Update `docs/08-roadmap.md`:

1. Add new sprints for the domain implementation.
2. Define sprint scope and dependencies.

---

## Step 6: Create Backlog Stories

Update `docs/09-backlog.md`:

1. Break the new domain into implementable stories.
2. Each story should follow the standard format (ID, title, description, acceptance criteria).
3. Order stories by dependency (schema first, then API, then UI).
4. Assign stories to sprints based on the updated roadmap.

---

## Step 7: Assess Cross-Cutting Concerns

For the new domain, evaluate:

| Concern | Questions | Reference |
|---------|-----------|-----------|
| **Security** | Does this domain handle PII? New RBAC roles needed? | `docs/13-security-and-compliance.md` |
| **Offline** | Which features must work offline? Conflict resolution strategy? | `docs/12-offline-sync-strategy.md` |
| **Audit** | Which mutations need audit logging? New audit categories? | `docs/10-data-model.md` (audit_log table) |
| **LGPD** | New personal data categories? Retention policies? Erasure scope? | `docs/13-security-and-compliance.md` |
| **Keycloak** | New roles? New client scopes? Realm config changes? | `docs/20-keycloak-configuration.md` |
| **Reports** | New report types needed? New CSV export endpoints? | `docs/11-api-design.md` |

Update relevant documentation with findings.

---

## Step 8: Update Phase Boundary Rules

If the project's phase-boundary rule references specific table lists or feature lists:

1. Update `.project-ai/rules/phase-boundary.md` with the new phase scope.
2. Update `CLAUDE.md` Product Scope Guardrails section if table lists change.
3. Update the tech-lead agent's Phase Boundary Enforcement section.
4. Verify the `pre-implement` hook will correctly validate the new phase.

---

## Agent Assignments

| Step | Agent |
|------|-------|
| 1. Document domain | tech-lead + product-analyst (if available) |
| 2. Define requirements | tech-lead + product-analyst |
| 3. Design schema | backend-engineer (using design-database-schema skill) |
| 4. Design API | backend-engineer (using design-api-contract skill) |
| 5. Update scope | tech-lead |
| 6. Create stories | tech-lead (using analyze-requirements skill) |
| 7. Assess cross-cutting | tech-lead + security-engineer |
| 8. Update rules | tech-lead |

---

## Linked Artifacts

| Artifact | Type | Usage |
|----------|------|-------|
| `design-database-schema` | Skill | Schema design for new tables |
| `design-api-contract` | Skill | API contract design for new endpoints |
| `analyze-requirements` | Skill | Story breakdown for new domain |
| `phase-boundary` | Rule | Updated with new phase scope |
| `architecture-change` | Workflow | If structural changes needed |
| `feature-delivery` | Workflow | For implementing each story |

---

## Success Criteria

- [ ] Domain model documented with entities, relationships, and lifecycle.
- [ ] Requirements defined with RF-XX codes and acceptance criteria.
- [ ] Database schema designed and documented in `docs/10-data-model.md`.
- [ ] API contracts designed and documented in `docs/11-api-design.md`.
- [ ] Scope documentation updated (MVP scope, roadmap).
- [ ] Backlog stories created and ordered by dependency.
- [ ] Cross-cutting concerns assessed and documented.
- [ ] Phase boundary rules updated to include new domain.
