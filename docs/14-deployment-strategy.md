# 14 - Deployment Strategy

## Environments

| Environment | Purpose | Infrastructure | Database |
|-------------|---------|----------------|----------|
| **Local** | Development | Docker Compose (Go + PostgreSQL + React dev server) | Local PostgreSQL in container |
| **CI** | Automated testing | GitHub Actions + PostgreSQL service container | Ephemeral test database |
| **Staging** | Pre-production testing | Cloud hosting (same as production, smaller instance) | Separate database with test data |
| **Production** | Live system | Cloud hosting | Managed PostgreSQL with backups |

---

## Hosting Options (Budget-Friendly)

### Recommended: Railway or Render

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
React PWA → Cloudflare Pages (free)
Go API    → Railway ($5/month)
PostgreSQL → Supabase free tier or Railway add-on
Storage   → Cloudflare R2 (free tier)
Domain    → Cloudflare DNS (free)
SSL       → Automatic (Cloudflare + Railway)

Estimated monthly cost: $5-10
```

### Production Setup (Phase 2+)

```
React PWA → Cloudflare Pages (free)
Go API    → Railway or Render ($10-20/month)
PostgreSQL → Supabase Pro or Railway ($25/month)
Storage   → Cloudflare R2 ($5/month)
Redis     → Railway or Upstash (free tier)
Domain    → Cloudflare DNS (free)
Monitoring → Grafana Cloud (free tier)

Estimated monthly cost: $40-50
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

  # Deploy to staging (on push to develop)
  deploy-staging:
    needs: [backend-test, frontend-test]
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      # Deploy steps depend on hosting provider
      # Railway: uses railway CLI
      # Render: auto-deploys from branch

  # Deploy to production (on push to main)
  deploy-production:
    needs: [backend-test, frontend-test]
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

# JWT
JWT_SECRET=<random_64_char_string>
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h        # 7 days

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
