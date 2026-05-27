# Chesed - Instituto Nova SOS Management Platform

**Chesed** (חסד — "loving-kindness" in Hebrew) is the operational management platform for **Instituto Nova SOS**, a social organization focused on community support through educational, social, assistance, and human development activities.

## What This System Does

- Centralized registration of beneficiaries, volunteers, and professionals
- Triage and service attendance workflow management
- Campaign and social event coordination
- Donation tracking and accountability
- Management reports and social impact metrics
- LGPD-compliant consent and data handling
- Offline-first field operations via mobile PWA

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (Golang) with chi router, pgx, golang-migrate |
| Frontend | React + TypeScript + Vite + Tailwind CSS |
| Database | PostgreSQL 16 |
| Architecture | REST API, offline-first PWA, mobile-first |
| Auth | Keycloak (OIDC) + RBAC with campus-scoped data isolation |
| Storage | S3-compatible (MinIO dev, Cloudflare R2 prod) |

## Project Structure

```
chesed/
├── backend/                  # Go API server
│   ├── cmd/server/           # Application entry point
│   ├── internal/             # Domain, handlers, services, repositories
│   ├── migrations/           # SQL migration files
│   └── Makefile
├── frontend/                 # React PWA
│   ├── src/
│   │   ├── api/              # API client layer
│   │   ├── components/       # Reusable UI components
│   │   ├── hooks/            # Custom React hooks
│   │   ├── offline/          # IndexedDB + sync engine
│   │   ├── pages/            # Route-level pages
│   │   └── types/            # TypeScript interfaces
│   └── vite.config.ts
├── .project-ai/              # AI delivery operating model (skills, agents, hooks, rules, playbooks, templates, checklists, workflows)
├── docs/                     # Architecture and product documentation
├── CLAUDE.md                 # AI agent rules (Claude Code)
├── CODEX.md                  # AI agent rules (any coding agent)
└── HANDOFF.md                # Session history and next steps
```

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Go 1.22+ (for running backend outside Docker)
- Node.js 20+ (for running frontend outside Docker)
- [golang-migrate CLI](https://github.com/golang-migrate/migrate) (`brew install golang-migrate`)

### Local Development Setup

```bash
# 1. Start all services (PostgreSQL, Keycloak, Go API, React dev server)
docker compose up -d

# 2. Wait for Keycloak to be healthy (~30-60 seconds)
docker inspect --format='{{.State.Health.Status}}' chesed-keycloak-1

# 3. Run database migrations
cd backend && make migrate-up

# 4. Initialize Keycloak realm (User Profile + test users)
./keycloak/init-realm.sh

# 5. Open the frontend
open http://localhost:5173
```

### Test Users

All test users share the password **`Test1234!`** and belong to the default campus (Instituto Nova SOS).

| Username | Email | Password | Role | Access Level |
|----------|-----------|----------|------|-------------|
| `volunteer` | `volunteer@chesed.test` | `Test1234!` | VOLUNTEER | Basic data entry, triage creation |
| `secretary` | `secretary@chesed.test` | `Test1234!` | SECRETARY | Person registration, scheduling |
| `professional` | `professional@chesed.test` | `Test1234!` | PROFESSIONAL | Service attendance recording |
| `coordinator` | `coordinator@chesed.test` | `Test1234!` | COORDINATOR | Full operational access within campus |
| `admin` | `admin@chesed.test` | `Test1234!` | ADMIN | Full system access, cross-campus queries |

### Service URLs

| Service | URL |
|---------|-----|
| Frontend | http://localhost:5173 |
| API | http://localhost:8080/api/v1/health |
| Keycloak Admin | http://localhost:8180/admin (admin/admin) |
| Mailpit (email) | http://localhost:8025 |
| PostgreSQL | localhost:5432 (chesed/chesed) |

### Running Tests
```bash
cd backend && make test
cd frontend && npm test
```

### Fresh Restart
```bash
# Wipe everything and start clean
docker compose down -v
docker compose up -d
# Wait for Keycloak, then re-run steps 3-4 above
```

## Documentation

All product, architecture, and implementation documentation is in [`docs/`](docs/). Start here:

| Document | Purpose |
|----------|---------|
| [Product Vision](docs/01-product-vision.md) | What we're building and why |
| [Requirements](docs/03-requirements-catalog.md) | Complete functional and non-functional requirements |
| [Architecture](docs/05-architecture-proposal.md) | System architecture and technology decisions |
| [MVP Scope](docs/07-mvp-scope.md) | What's in the first release |
| [Roadmap](docs/08-roadmap.md) | Phased implementation plan |
| [API Design](docs/11-api-design.md) | REST API contracts |
| [Data Model](docs/10-data-model.md) | Database schema |

## AI-Assisted Development

This repository is designed for AI-assisted development. See:
- [`CLAUDE.md`](CLAUDE.md) — Rules for Claude Code
- [`CODEX.md`](CODEX.md) — Rules for any AI coding agent
- [`HANDOFF.md`](HANDOFF.md) — Session continuity and next steps
- [`.project-ai/`](.project-ai/) — Delivery operating model with 49 artifacts (skills, agents, hooks, rules, playbooks, templates, checklists, workflows)

## Origin

This project is a rebuild of the [SOS-Gestao-Final](https://github.com/Amadeus-22/SOS-Gestao-Final) Django prototype. The migration rationale is documented in the Session 1 notes in [`HANDOFF.md`](HANDOFF.md).

## License

Private — Instituto Nova SOS
