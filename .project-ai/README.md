# Chesed AI-Assisted Delivery Operating System

This directory contains the **project-level AI delivery operating model** for the Chesed platform. It provides structured tools that enable AI agents (Claude Code, Codex, or similar) to operate like a senior multidisciplinary development team.

## Relationship to Other Files

| File | Contains | Scope |
|------|----------|-------|
| `CLAUDE.md` | Architecture guardrails, code rules, tech stack constraints | **What** the code must look like |
| `CODEX.md` | Same rules, agent-agnostic format | **What** the code must look like |
| `.project-ai/` | Process orchestration, workflows, quality gates | **How** work flows through delivery |
| `docs/` | Product requirements, domain model, API design | **What** to build |

**Rule**: `.project-ai/` complements but never duplicates `CLAUDE.md` or `CODEX.md`.

---

## Quick Reference

| I want to... | Use this artifact |
|---------------|-------------------|
| Start a new feature | `workflows/feature-delivery.md` |
| Refine a business request into a story | `skills/refine-requirements.md` |
| Validate acceptance criteria | `skills/validate-acceptance-criteria.md` |
| Break a story into tasks | `skills/analyze-requirements.md` |
| Design an API contract | `skills/design-api-contract.md` |
| Design a database schema | `skills/design-database-schema.md` |
| Design a backend endpoint | `skills/design-backend-feature.md` |
| Design a frontend page | `skills/design-frontend-feature.md` |
| Bootstrap project infrastructure | `playbooks/bootstrap-project-infrastructure.md` |
| Implement a backend endpoint | `playbooks/implement-backend-endpoint.md` |
| Implement a frontend page | `playbooks/implement-frontend-page.md` |
| Add a database table | `playbooks/add-database-table.md` |
| Add offline support | `playbooks/add-offline-support.md` |
| Review API contract | `skills/review-api-contract.md` |
| Review security | `skills/review-security.md` + `playbooks/conduct-security-review.md` |
| Review code quality | `skills/review-code.md` |
| Review a migration | `skills/review-migration.md` |
| Design tests | `skills/design-test-plan.md` |
| Validate test coverage | `skills/execute-test-plan.md` |
| Analyze performance | `skills/performance-analysis.md` |
| Check if backend work is done | `checklists/backend-feature-complete.md` |
| Check if frontend work is done | `checklists/frontend-feature-complete.md` |
| Prepare a sprint release | `playbooks/prepare-sprint-delivery.md` |
| Make an architecture decision | `templates/adr.md` + `workflows/architecture-change.md` |
| Handle security-sensitive change | `workflows/security-sensitive-change.md` |
| Fix a bug | `workflows/bug-fix-workflow.md` |
| Handle a production emergency | `workflows/hotfix-workflow.md` + `playbooks/rollback-and-hotfix.md` |
| Onboard a new domain area | `playbooks/onboard-new-feature-domain.md` |
| Run a sprint retrospective | `templates/sprint-retrospective.md` |
| Document an incident | `templates/incident-report.md` |
| End a session | `skills/prepare-handoff.md` |

---

## Artifact Types

### Skills (`skills/`)
Specialized expertise invocations. Each skill represents a senior-level capability (requirements analysis, backend design, security review, etc.) with clear inputs, process, and outputs.

### Agents (`agents/`)
Role-based AI personas with defined identity, scope, and responsibilities. Each agent knows which skills to invoke and what quality standards to enforce.

### Hooks (`hooks/`)
Procedural triggers that fire at key delivery moments (before implementing, before/after API changes, before review, before release). Blocking hooks must pass before work proceeds.

### Rules (`rules/`)
Codified decision rules for process discipline (documentation-first, backlog traceability, phase boundaries, security review triggers, offline-first assessment).

### Prompts (`prompts/`)
Standalone, reusable prompts that drive each stage of the development lifecycle. Each prompt defines a role, objective, inputs, outputs, constraints, quality enforcement, and AI tooling integration. Prompts are the executable layer that orchestrates skills, agents, hooks, and rules into a complete pipeline.

### Playbooks (`playbooks/`)
Step-by-step guides for recurring multi-step workflows (implement endpoint, implement page, add table, add offline support, security review, sprint delivery).

### Templates (`templates/`)
Reusable document structures for artifacts created repeatedly (feature specs, ADRs, API change proposals, test plans, security review reports).

### Checklists (`checklists/`)
Quality gate enforcement lists that must be satisfied before marking work complete (backend feature, frontend feature, API review, security review, sprint release).

### Workflows (`workflows/`)
Process diagrams showing how skills, agents, hooks, and playbooks connect across the delivery lifecycle.

---

## Artifact Inventory

| Category | Count | Files |
|----------|-------|-------|
| Skills | 20 | analyze-requirements, assess-release-readiness, design-api-contract, design-backend-feature, design-database-schema, design-frontend-feature, design-offline-support, design-test-plan, execute-test-plan, infrastructure-setup, maintain-docs, maintainability-analysis, performance-analysis, prepare-handoff, refine-requirements, reliability-validation, review-api-contract, review-code, review-migration, review-security, validate-acceptance-criteria |
| Agents | 8 | tech-lead, backend-engineer, frontend-engineer, security-engineer, reviewer, devops-engineer, qa-engineer, product-analyst |
| Hooks | 11 | pre-implement, pre-api-change, post-api-change, pre-migration, post-migration, post-implement, pre-review, pre-merge, pre-release, pre-deploy, post-deploy |
| Rules | 10 | documentation-first, backlog-traceability, phase-boundary, security-review-triggers, offline-first-assessment, quality-gates, dependency-management, test-coverage-enforcement, performance-budget, api-versioning-strategy |
| Playbooks | 11 | implement-backend-endpoint, implement-frontend-page, add-database-table, add-offline-support, conduct-security-review, prepare-sprint-delivery, implement-with-quality, refactor-for-quality, bootstrap-project-infrastructure, rollback-and-hotfix, onboard-new-feature-domain |
| Templates | 8 | feature-spec, adr, api-change-proposal, test-plan, security-review-report, performance-report, incident-report, sprint-retrospective |
| Checklists | 7 | backend-feature-complete, frontend-feature-complete, api-review, security-review, sprint-release, pr-quality, refactoring |
| Workflows | 6 | feature-delivery, architecture-change, security-sensitive-change, bug-fix-workflow, hotfix-workflow, performance-optimization |
| Prompts | 10 | requirement-analysis, architecture-design, task-breakdown, backend-implementation, frontend-implementation, code-review, test-generation, security-review, performance-review, release-readiness |
| **Total** | **91 + 2 meta** | |

---

## Continuous Improvement

These tools are living project artifacts. During delivery, AI agents must:

1. **Evaluate** whether tools are sufficient, need refinement, or need consolidation
2. **Propose** improvements when recurring friction, ambiguity, or quality gaps are found
3. **Create** new artifacts when justified by repeated patterns
4. **Update** existing artifacts when outdated
5. **Remove** artifacts that prove unnecessary

See `CLAUDE.md` and `CODEX.md` for the continuous improvement mandate.
