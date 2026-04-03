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
| Auth | JWT + RBAC with campus-scoped data isolation |
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
├── docs/                     # Architecture and product documentation
├── CLAUDE.md                 # AI agent rules (Claude Code)
├── CODEX.md                  # AI agent rules (any coding agent)
└── HANDOFF.md                # Session history and next steps
```

## Getting Started

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker and Docker Compose
- PostgreSQL 16 (or use Docker)

### Development
```bash
# Start all services
docker compose up

# Backend only
cd backend && make run

# Frontend only
cd frontend && npm run dev

# Run tests
cd backend && make test
cd frontend && npm test
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

## Origin

This project is a rebuild of the [SOS-Gestao-Final](https://github.com/Amadeus-22/SOS-Gestao-Final) Django prototype. The migration rationale is documented in the Session 1 notes in [`HANDOFF.md`](HANDOFF.md).

## License

Private — Instituto Nova SOS
