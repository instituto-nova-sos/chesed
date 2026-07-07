# 14 - Deployment Strategy

## Environments

| Environment | Purpose | Infrastructure | Database |
|-------------|---------|----------------|----------|
| **Local** | Development | Docker Compose (Go + PostgreSQL + Keycloak + React dev server) | Local PostgreSQL in container |
| **CI** | Automated testing | GitHub Actions + PostgreSQL service container | Ephemeral test database |
| **Staging** | Pre-production testing | Cloud hosting (same as production, smaller instance) | Separate database with test data |
| **Production** | Live system | Cloud hosting | Managed PostgreSQL with backups |

---

## Hosting Options (Budget-Friendly)

### Recommended (2026 cost-minimized): single VPS + Cloudflare free tiers

> The decisive constraint is **Keycloak cannot scale to zero** — it holds sessions
> in JVM heap and validates a token on every request, so it needs ~512MB–1GB RAM
> running 24/7. This eliminates every "free tier that sleeps" (Render free web
> services, Fly.io auto-stop, Railway usage-only trial) and every serverless model.
> You are effectively paying for **one always-on box** (~2.5 vCPU / 2–3GB across the
> whole compose stack). The PWA (static) and object storage (a few GB) are free on
> Cloudflare regardless, so the only real cost is that one box.

**Primary recommendation — ~€4/month, indefinitely, production-grade:**

```
PWA        → Cloudflare Pages          (free, unlimited bandwidth)
Documents  → Cloudflare R2             (free tier: 10GB + zero egress fees)
Postgres + Keycloak + Go API + TLS
           → one ARM VPS running docker-compose.prod.yml
             • Contabo Cloud VPS 10  (4 vCPU / 8GB)  ≈ €4/mo  ← safest RAM headroom
             • Hetzner CAX11 (ARM)   (2 vCPU / 4GB)  ≈ €4/mo  ← cap Keycloak heap -Xmx512m
TLS        → Caddy / nginx + Let's Encrypt on the box (free, auto-renew)

Estimated monthly cost: ~€4 (≈ US$4–5). Confirm live VPS price at order time —
Hetzner adjusted prices in Apr and Jun 2026.
```

**Runner-up — $0/month + Brazilian data residency:** Oracle Cloud **Always Free**
(Ampere A1 ARM, **São Paulo / Vinhedo** region). Runs the whole stack for free
forever and is the only free option physically in Brazil (LGPD win). Docked to
runner-up because of A1 capacity scarcity, the June-2026 cut to 2 OCPU / 12GB,
idle-reclamation policy, and no support — great if a technical volunteer owns the
ops, risky if nobody can.

**Non-profit credit bridge (year one free, then fall back to the €4 VPS):** the
Instituto is Brazil-eligible for **Google for Nonprofits Cloud credits** and the
**AWS Nonprofit Credit Program (up to $5,000/yr)** — both via TechSoup Global /
Goodstack validation, both with a **São Paulo** region. Use these as runway, not as
the permanent architecture (credits expire ~1 yr).

**LGPD / data residency (practical):** LGPD does **not** require Brazilian data to
stay in Brazil — international transfer is allowed with an adequate legal basis, so
an EU VPS (Hetzner Germany/Finland) is compliant. Prefer a **São Paulo** region only
when it is free/near-free via Oracle or a nonprofit credit (cleaner paperwork, lower
latency); do not pay a premium for it.

| Option | $/mo (all-in) | Pros | Cons |
|---|---|---|---|
| **Contabo 8GB VPS + CF Pages/R2** ⭐ | ~€4 | Cheapest with RAM headroom; full control | Self-managed; EU/US only (no BR region) |
| **Hetzner CAX11 ARM + CF Pages/R2** | ~€4 | Reliable, great tooling, ARM efficiency | 4GB needs Keycloak heap cap; no BR region |
| **Oracle Always Free A1 (São Paulo)** 🥈 | $0 | Free forever; **BR residency**; 12GB RAM | Capacity scarcity; idle-reclaim; no support |
| **Railway / Render / Fly.io** | ~$25–45 | Zero ops, managed DB | 6–10× VPS cost (always-on Keycloak RAM) |
| **AWS/GCP on nonprofit credit** | $0 yr 1, then market | São Paulo region; free first year | Credit expires (~1 yr); more setup |

The PaaS options below remain valid if the Instituto prefers zero-ops over cost;
they are 6–10× the VPS price purely because Keycloak must stay always-on.

### Alternative (managed, higher cost): Railway or Render

| Service | Free Tier | Paid Tier | Notes |
|---------|-----------|-----------|-------|
| **Railway** | $5/month credit | $5-20/month | Easy Docker deployment; built-in PostgreSQL |
| **Render** | Free web services (sleep after 15min inactivity) | $7/month | Static site hosting for React PWA |
| **Fly.io** | 3 shared VMs free | $5+/month | Good for multi-region (future) |
| **Supabase** | Free PostgreSQL (500MB) | $25/month | Managed PostgreSQL with backups |
| **Neon** | Free PostgreSQL (512MB) | $19/month | Serverless PostgreSQL |
| **Cloudflare R2** | 10GB free | $0.015/GB/month | Object storage for documents |
| **Cloudflare Pages** | Free | Free | Static hosting for React PWA |

### Recommended Initial Setup (Phase 1)

```
React PWA     → Cloudflare Pages (free)
Go API        → Railway ($5/month)
Keycloak      → Railway ($5-7/month, ~512MB RAM)
PostgreSQL    → Supabase free tier or Railway add-on
Storage       → Cloudflare R2 (free tier)
Domain        → Cloudflare DNS (free)
SSL           → Automatic (Cloudflare + Railway)
WAF           → Cloudflare (free tier)

Estimated monthly cost: $10-17/month
```

### Production Setup (Phase 2+)

```
React PWA  → Cloudflare Pages (free)
Go API     → Railway or Render ($10-20/month)
Keycloak   → Railway ($10-15/month, ~1GB RAM)
PostgreSQL → Supabase Pro or Railway ($25/month)
Storage    → Cloudflare R2 ($5/month)
Redis      → Railway or Upstash (free tier)
Domain     → Cloudflare DNS (free)
Monitoring → Grafana Cloud (free tier)

Estimated monthly cost: $55-70/month
```

---

## CI/CD Pipeline

### GitHub Actions Workflow

```yaml
name: CI/CD

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  # Backend tests
  backend-test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: test_db
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_password
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run migrations
        run: make migrate-up-test
      - name: Run tests
        run: make test
      - name: Run linter
        run: make lint

  # Frontend tests and build
  frontend-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json
      - run: cd frontend && npm ci
      - run: cd frontend && npm run lint
      - run: cd frontend && npm run test
      - run: cd frontend && npm run build

  # Keycloak image scan
  keycloak-image-scan:
    runs-on: ubuntu-latest
    steps:
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: 'quay.io/keycloak/keycloak:26-alpine'
          exit-code: '1'
          severity: 'CRITICAL'
          format: 'table'

  # Deploy to staging (on push to develop)
  deploy-staging:
    needs: [backend-test, frontend-test, keycloak-image-scan]
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      # Deploy steps depend on hosting provider
      # Railway: uses railway CLI
      # Render: auto-deploys from branch

  # Deploy to production (on push to main)
  deploy-production:
    needs: [backend-test, frontend-test, keycloak-image-scan]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      # Production deploy with manual approval (optional)
```

### Branch Strategy

```
main       → Production deployments
develop    → Staging deployments
feature/*  → Feature branches (PR to develop)
hotfix/*   → Hotfix branches (PR to main)
```

---

## Docker Configuration

### Backend Dockerfile (Production)

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /server .
COPY migrations/ ./migrations/

EXPOSE 8080

CMD ["./server"]
```

### Frontend Dockerfile (Production)

```dockerfile
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# Serve with nginx or deploy to CDN
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
```

### Docker Compose (Development)

```yaml
services:
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile.dev
    ports:
      - "8080:8080"
    volumes:
      - ./backend:/app
    depends_on:
      db:
        condition: service_healthy
    env_file: .env

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.dev
    ports:
      - "5173:5173"
    volumes:
      - ./frontend/src:/app/src
    env_file: .env

  keycloak:
    image: quay.io/keycloak/keycloak:26-alpine
    command: start-dev --import-realm
    ports:
      - "8180:8080"
    volumes:
      - ./keycloak/realm-export.json:/opt/keycloak/data/import/realm.json
    environment:
      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://db:5432/chesed_keycloak
      KC_DB_USERNAME: ${DB_USER}
      KC_DB_PASSWORD: ${DB_PASSWORD}
      KC_HOSTNAME: ${KEYCLOAK_HOSTNAME:-localhost}
      KC_HTTP_RELATIVE_PATH: /auth
      KEYCLOAK_ADMIN: admin
      KEYCLOAK_ADMIN_PASSWORD: ${KEYCLOAK_ADMIN_PASSWORD}
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

> **Note:** Keycloak can share the same PostgreSQL instance (separate database/schema) to reduce cost. In production with high load, consider a dedicated database.

---

## Backup and Disaster Recovery

### Database Backups

| Type | Frequency | Retention | Method |
|------|-----------|-----------|--------|
| Automated snapshot | Daily | 7 days | Managed PostgreSQL feature (Supabase/Railway) |
| Logical backup | Weekly | 30 days | `pg_dump` via cron job or GitHub Action |
| Point-in-time recovery | Continuous | 7 days | WAL archiving (managed PostgreSQL) |

### Backup Verification

- Monthly: Restore backup to test environment and verify data integrity
- Quarterly: Full disaster recovery drill

### Keycloak Backup

Keycloak realm configuration is exported as JSON and version-controlled in `keycloak/realm-export.json`. Database backup covers Keycloak data (users, sessions). Recovery: import realm JSON into a fresh Keycloak instance, restore database.

### Recovery Procedure

1. **Database loss**: Restore from most recent snapshot or PITR
2. **Application failure**: Redeploy from last known good commit
3. **Complete infrastructure loss**: Rebuild from Docker images + database backup
4. **Data corruption**: PITR to point before corruption; verify with audit logs

### Recovery Time Objectives

| Scenario | RTO | RPO |
|----------|-----|-----|
| Application server failure | 5 minutes (auto-restart) | 0 (stateless) |
| Database failure | 15 minutes (restore snapshot) | 1 hour (snapshot interval) |
| Complete infrastructure loss | 2 hours | 24 hours (daily backup) |

---

## Monitoring and Alerting

### Application Metrics (Grafana Cloud free tier)

- API response times (p50, p95, p99)
- Error rates (4xx, 5xx)
- Active users
- Sync operations (push/pull count, failures)
- Database connection pool usage

### Health Checks

```
GET /api/v1/health → 200 { "status": "ok", "db": "ok", "version": "1.0.0" }
```

### Alerting (Phase 2+)

- API error rate > 5% for 5 minutes → Alert
- Database connection failures → Alert
- Sync failure rate > 10% → Alert
- Disk usage > 80% → Alert

---

## Environment Variables

```bash
# Application
APP_ENV=production          # development, staging, production
APP_PORT=8080
APP_LOG_LEVEL=info

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=sos_gestao
DB_USER=sos_user
DB_PASSWORD=<secure_password>
DB_SSL_MODE=require         # disable for local dev

# Keycloak (OIDC)
KEYCLOAK_URL=http://keycloak:8080       # Keycloak base URL
KEYCLOAK_REALM=chesed                    # Realm name
KEYCLOAK_CLIENT_ID=chesed-api            # OIDC client ID for the Go API
KEYCLOAK_CLIENT_SECRET=<secret>          # Client secret for Keycloak Admin API access
KEYCLOAK_ADMIN_PASSWORD=<secure_password> # Keycloak admin console password

# Storage
STORAGE_TYPE=s3             # local, s3
S3_ENDPOINT=https://...
S3_BUCKET=sos-gestao-docs
S3_ACCESS_KEY=<key>
S3_SECRET_KEY=<secret>

# CORS
CORS_ORIGINS=https://sos-gestao.pages.dev

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60        # seconds
```
