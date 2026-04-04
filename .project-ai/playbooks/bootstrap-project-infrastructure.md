# Playbook: Bootstrap Project Infrastructure

Step-by-step guide for setting up the Chesed project infrastructure from scratch. This playbook is executed once during Sprint 1 to establish the foundational development environment.

---

## When to Use

- Sprint 1 initialization — first-time project setup.
- After a major infrastructure redesign (rare).
- When onboarding a new development environment from scratch.

---

## Prerequisites

- Docker and Docker Compose installed.
- Go 1.22+ installed.
- Node.js 20+ and npm installed.
- `golangci-lint` installed.
- Access to the repository.

---

## Flow Overview

```
GO MODULE → REACT APP → DOCKER COMPOSE → MAKEFILE → ENV CONFIG → KEYCLOAK → DATABASE → CI/CD → VERIFY
```

---

## Step 1: Initialize Go Module

Location: `backend/`

1. Create the directory structure:
   ```
   backend/
   ├── cmd/server/main.go
   ├── internal/
   │   ├── config/
   │   ├── domain/
   │   ├── handler/
   │   ├── middleware/
   │   ├── repository/
   │   ├── service/
   │   └── sync/
   ├── migrations/
   ├── scripts/
   ├── Dockerfile
   ├── go.mod
   └── go.sum
   ```

2. Initialize Go module: `go mod init github.com/instituto-nova-sos/chesed/backend`
3. Add core dependencies:
   - `github.com/go-chi/chi/v5`
   - `github.com/jackc/pgx/v5`
   - `github.com/golang-migrate/migrate/v4`
   - `github.com/coreos/go-oidc/v3`
   - `github.com/go-playground/validator/v10`
   - `github.com/stretchr/testify`
4. Create minimal `cmd/server/main.go` with health check endpoint.
5. Create `Dockerfile` (multi-stage build):
   - Build stage: `golang:1.22-alpine`, compile binary.
   - Runtime stage: `alpine:3.19`, copy binary, expose port.

---

## Step 2: Initialize React Application

Location: `frontend/`

1. Create React app with Vite: `npm create vite@latest frontend -- --template react-ts`
2. Install core dependencies:
   - `keycloak-js`
   - `react-router-dom`
   - `react-hook-form`
   - `@hookform/resolvers` + `zod`
   - `dexie` + `dexie-react-hooks`
   - `tailwindcss` + `postcss` + `autoprefixer`
3. Configure TypeScript strict mode in `tsconfig.json`.
4. Configure Tailwind CSS.
5. Create `Dockerfile` (multi-stage build):
   - Build stage: `node:20-alpine`, npm install, npm build.
   - Runtime stage: `nginx:alpine`, copy build output, configure nginx.

---

## Step 3: Create Docker Compose

Location: `docker-compose.yml` (project root)

Services to define:

1. **postgres**:
   - Image: `postgres:16-alpine`
   - Health check: `pg_isready`
   - Volume: `chesed-postgres-data`
   - Port: 5432

2. **keycloak**:
   - Image: `quay.io/keycloak/keycloak:latest`
   - Command: `start-dev --import-realm`
   - Health check: HTTP on `/health/ready`
   - Volume mount: `./keycloak/realm-export.json:/opt/keycloak/data/import/realm-export.json`
   - Port: 8180
   - Depends on: postgres (if using external DB) or standalone

3. **backend**:
   - Build: `./backend`
   - Health check: HTTP on `/api/v1/health`
   - Port: 8080
   - Depends on: postgres (healthy), keycloak (healthy)
   - Env file: `.env`

4. **frontend**:
   - Build: `./frontend`
   - Port: 5173 (dev) or 3000 (prod)
   - Depends on: backend (healthy)

---

## Step 4: Create Makefile

Location: `Makefile` (project root)

Define all targets per the `infrastructure-setup` skill specification:

- Development: `run`, `dev`, `docker-up`, `docker-down`
- Build: `build`, `build-frontend`
- Test: `test`, `test-frontend`, `test-all`, `coverage`
- Lint: `lint`, `lint-frontend`, `lint-all`
- Database: `migrate-up`, `migrate-down`, `migrate-create`, `seed`
- Utility: `clean`, `help`

Include `.PHONY` declarations for all targets.
Include self-documentation via `help` target that parses `##` comments.

---

## Step 5: Create Environment Configuration

Location: `.env.example` (project root)

1. Document all variables with comments explaining purpose and valid values.
2. Use sensible defaults for local development.
3. Group variables by service (Application, PostgreSQL, Keycloak, Frontend, CORS, Rate Limiting).
4. Add `.env` to `.gitignore`.

---

## Step 6: Configure Keycloak Realm

Location: `keycloak/`

1. Create `keycloak/realm-export.json` following `docs/20-keycloak-configuration.md`:
   - Realm: `chesed`
   - Clients: `chesed-api` (confidential), `chesed-frontend` (public)
   - Roles: ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER
   - Custom claims mapper for `campus_id`
2. Verify realm imports correctly on container startup.

---

## Step 7: Create Database Initialization

1. Create initial migration: `000001_create_extensions.up.sql`
   - Enable `uuid-ossp` or use `gen_random_uuid()` (built-in PG 13+).
   - Enable `pg_trgm` for text search.
2. Create seed data script: `scripts/seed.sql`
   - Default campus record.
   - Service type records (8 types).
3. Verify migrations run forward and backward cleanly.

---

## Step 8: Create CI/CD Pipeline

Location: `.github/workflows/`

1. Create `ci.yml` per the `infrastructure-setup` skill specification.
2. Configure branch protection rules (document, don't auto-apply).
3. Verify pipeline runs successfully on a test commit.

---

## Step 9: Verify End-to-End

Run the full verification sequence:

```bash
# 1. Start infrastructure
make docker-up

# 2. Wait for health checks
# (Docker Compose health checks should handle this)

# 3. Apply migrations
make migrate-up

# 4. Apply seed data
make seed

# 5. Run tests
make test-all

# 6. Run linters
make lint-all

# 7. Verify health endpoint
curl http://localhost:8080/api/v1/health

# 8. Verify Keycloak
curl http://localhost:8180/realms/chesed/.well-known/openid-configuration

# 9. Clean up
make docker-down
```

All 9 steps must succeed before infrastructure is considered complete.

---

## Agent Assignments

| Step | Agent |
|------|-------|
| 1. Go Module | backend-engineer + devops-engineer |
| 2. React App | frontend-engineer + devops-engineer |
| 3. Docker Compose | devops-engineer |
| 4. Makefile | devops-engineer |
| 5. Environment | devops-engineer |
| 6. Keycloak | devops-engineer + security-engineer |
| 7. Database | backend-engineer + devops-engineer |
| 8. CI/CD | devops-engineer |
| 9. Verify | devops-engineer |

---

## Linked Artifacts

| Artifact | Type | Usage |
|----------|------|-------|
| `infrastructure-setup` | Skill | Produces the design specifications implemented by this playbook |
| `devops-engineer` | Agent | Primary owner of this playbook |
| `backend-engineer` | Agent | Go module and migration setup |
| `frontend-engineer` | Agent | React application setup |
| `security-engineer` | Agent | Keycloak configuration review |
| `docs/05-architecture-proposal.md` | Doc | System architecture reference |
| `docs/14-deployment-strategy.md` | Doc | Environment and CI/CD reference |
| `docs/20-keycloak-configuration.md` | Doc | Keycloak realm setup |

---

## Success Criteria

- [ ] `make docker-up` starts all services without errors.
- [ ] All health checks pass within 60 seconds.
- [ ] `make migrate-up` applies migrations successfully.
- [ ] `make seed` populates reference data.
- [ ] `make test-all` runs (may have minimal tests at this point).
- [ ] `make lint-all` passes.
- [ ] Health endpoint returns 200.
- [ ] Keycloak OIDC discovery endpoint is accessible.
- [ ] `.env.example` documents all required variables.
- [ ] CI/CD pipeline runs on push.
