# Skill: Infrastructure Setup

## Purpose

Design and configure project infrastructure: Docker Compose services, Makefile targets, CI/CD pipeline configuration, environment variable management, and local development setup. Produces complete, ready-to-implement infrastructure specifications.

## When to Use / Trigger

- Sprint 1 bootstrap — setting up the initial project infrastructure.
- When adding new services to Docker Compose (e.g., Redis, MinIO).
- When modifying CI/CD pipeline stages or adding new quality gates.
- When restructuring Makefile targets or adding new build steps.
- When a user says "set up infrastructure" or "configure Docker" or "add CI/CD".

## Role / Expertise

Senior DevOps engineer with expertise in:
- Docker Compose v3 for multi-service local development and CI environments.
- Makefile design for Go and React project build automation.
- GitHub Actions for CI/CD with quality gate enforcement.
- PostgreSQL container configuration with health checks and initialization.
- Keycloak container configuration with realm import.
- Environment variable management and secrets hygiene.

## Inputs

| Input | Required | Source |
|-------|----------|--------|
| System architecture | Yes | `docs/05-architecture-proposal.md` |
| Deployment strategy | Yes | `docs/14-deployment-strategy.md` |
| Keycloak configuration | Yes | `docs/20-keycloak-configuration.md` |
| Technology stack | Yes | `CLAUDE.md` (Implementation Constraints section) |
| Quality gates | For CI/CD | `docs/quality/quality-gates.md` |
| Security requirements | For CI/CD | `docs/13-security-and-compliance.md` |

## Process

### Step 1: Design Docker Compose Services

Read `docs/05-architecture-proposal.md` and `docs/14-deployment-strategy.md` to understand the service topology.

Design services:

1. **PostgreSQL 16**:
   - Image: `postgres:16-alpine`
   - Health check: `pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}`
   - Persistent volume for data
   - Initialization script for database creation
   - Environment variables: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`

2. **Keycloak**:
   - Image: `quay.io/keycloak/keycloak:latest`
   - Health check: HTTP GET on `/health/ready`
   - Realm import from `keycloak/realm-export.json`
   - Environment variables: `KEYCLOAK_ADMIN`, `KEYCLOAK_ADMIN_PASSWORD`, `KC_DB`, `KC_DB_URL`
   - Depends on PostgreSQL (or uses embedded H2 for development)

3. **Backend (Go API)**:
   - Build from `backend/Dockerfile` (multi-stage: build + runtime)
   - Health check: HTTP GET on `/api/v1/health`
   - Depends on PostgreSQL and Keycloak (with health conditions)
   - Environment variables: database URL, Keycloak OIDC settings, server port

4. **Frontend (React)**:
   - Build from `frontend/Dockerfile` (multi-stage: build + nginx)
   - Development mode: Vite dev server with hot reload
   - Production mode: Nginx serving built assets
   - Environment variables: API URL, Keycloak client settings

### Step 2: Design Makefile Targets

Organize Makefile with clear sections:

**Development targets:**
- `make run` — Start Go server locally (requires local PostgreSQL)
- `make dev` — Start frontend dev server with Vite
- `make docker-up` — Start all services via Docker Compose
- `make docker-down` — Stop all services

**Build targets:**
- `make build` — Build Go binary (`go build -o bin/server ./cmd/server`)
- `make build-frontend` — Build React app (`npm run build`)

**Test targets:**
- `make test` — Run all Go tests (`go test -race -count=1 ./...`)
- `make test-frontend` — Run all React tests (`npm test`)
- `make test-all` — Run both backend and frontend tests
- `make coverage` — Generate coverage reports

**Lint targets:**
- `make lint` — Run golangci-lint (`golangci-lint run ./...`)
- `make lint-frontend` — Run ESLint (`npx eslint .`)
- `make lint-all` — Run both linters

**Database targets:**
- `make migrate-up` — Apply all pending migrations
- `make migrate-down` — Rollback last migration
- `make migrate-create NAME=xxx` — Create new migration file pair
- `make seed` — Apply seed data (service types, initial campus)

**Utility targets:**
- `make clean` — Remove build artifacts
- `make generate` — Run code generators (if any)
- `make help` — Show all available targets with descriptions

### Step 3: Design CI/CD Pipeline

Read `docs/quality/quality-gates.md` and `docs/13-security-and-compliance.md` for quality and security requirements.

Design GitHub Actions workflows:

**`.github/workflows/ci.yml`** (runs on every PR):
1. **Lint job**: golangci-lint (Go) + ESLint (TypeScript)
2. **Test job**: go test with coverage + Vitest with coverage
3. **Build job**: Docker multi-stage build (verifies images build successfully)
4. **Security scan job**: govulncheck (Go) + npm audit (frontend) + Trivy (Docker images)
5. **Quality gate job**: Coverage threshold check (≥80% new code), duplication check

**`.github/workflows/release.yml`** (runs on tag push):
1. Full CI pipeline
2. Build production Docker images
3. Tag images with version
4. Deploy to staging environment
5. Run smoke tests against staging

**Pipeline rules:**
- All jobs must pass before merge is allowed.
- Security scan blocks on CRITICAL or HIGH findings.
- Coverage below threshold blocks merge.
- Failed lint blocks merge.

### Step 4: Design Environment Configuration

Create `.env.example` with all variables:

```bash
# Application
APP_ENV=development
APP_PORT=8080
APP_LOG_LEVEL=debug

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=chesed
POSTGRES_PASSWORD=changeme
POSTGRES_DB=chesed

# Keycloak OIDC
KEYCLOAK_URL=http://localhost:8180
KEYCLOAK_REALM=chesed
KEYCLOAK_CLIENT_ID=chesed-api
KEYCLOAK_CLIENT_SECRET=changeme
KEYCLOAK_JWKS_URL=http://localhost:8180/realms/chesed/protocol/openid-connect/certs

# Frontend
VITE_API_URL=http://localhost:8080/api/v1
VITE_KEYCLOAK_URL=http://localhost:8180
VITE_KEYCLOAK_REALM=chesed
VITE_KEYCLOAK_CLIENT_ID=chesed-frontend

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:5173

# Rate Limiting
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200
```

### Step 5: Verify Design Completeness

Validate the infrastructure design against:
- [ ] All services from `docs/05-architecture-proposal.md` are represented in Docker Compose.
- [ ] All Makefile targets cover the build-test-lint-deploy lifecycle.
- [ ] CI/CD pipeline enforces quality gates from `docs/quality/quality-gates.md`.
- [ ] Environment variables cover all configuration from `docs/14-deployment-strategy.md`.
- [ ] No secrets hardcoded (all use environment variable placeholders).
- [ ] Health checks defined for all services.
- [ ] Docker builds use multi-stage for minimal production images.

## Outputs / Deliverables

1. **Docker Compose specification** — Complete `docker-compose.yml` with all services, health checks, volumes, networks.
2. **Makefile specification** — All targets with dependencies and commands.
3. **CI/CD pipeline specification** — GitHub Actions workflow YAML with all stages.
4. **Environment template** — `.env.example` with all variables and documentation.
5. **Dockerfile specifications** — Multi-stage builds for backend and frontend.
6. **Setup documentation** — README section for local development setup.

## References

| Document | Path | Usage |
|----------|------|-------|
| Architecture | `docs/05-architecture-proposal.md` | Service topology |
| Deployment strategy | `docs/14-deployment-strategy.md` | Environments, CI/CD, Docker, backups |
| Keycloak configuration | `docs/20-keycloak-configuration.md` | Realm setup |
| Quality gates | `docs/quality/quality-gates.md` | CI/CD quality enforcement |
| Security | `docs/13-security-and-compliance.md` | Security scanning requirements |
| Implementation guidelines | `docs/15-implementation-guidelines.md` | Build conventions |

## Constraints / Quality Bar

- Docker Compose must use v3 format.
- No secrets hardcoded in any configuration file.
- All services must have health checks with appropriate timeouts.
- Docker builds must be multi-stage (build + runtime) for minimal image size.
- CI/CD must enforce all conditions from the New Code Quality Gate.
- Makefile must be self-documenting (`make help` shows all targets).
- All environment variables must be documented in `.env.example`.
- Infrastructure must support running without external network access (for offline development).

## Interaction with Other Artifacts

- **Invoked by agents**: devops-engineer (primary), tech-lead (infrastructure decisions).
- **Feeds into playbooks**: bootstrap-project-infrastructure (step-by-step implementation guide).
- **Enables hooks**: pre-merge (CI/CD enforces quality gates), pre-release (deployment pipeline).
- **References rules**: quality-gates (CI enforcement), dependency-management (security scanning).
