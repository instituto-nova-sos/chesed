# 05 - Architecture Proposal

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    CLIENTS                                │
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  PWA (React)  │  │  Desktop Web │  │  WordPress    │  │
│  │  Mobile-first │  │  (same PWA)  │  │  Portal (API) │  │
│  │  Offline-first│  │              │  │               │  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬────────┘  │
│         │                  │                  │           │
└─────────┼──────────────────┼──────────────────┼───────────┘
          │                  │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────┐
│                   API GATEWAY / REVERSE PROXY             │
│                   (Nginx / Caddy)                         │
│                   TLS termination, rate limiting           │
└─────────────────────────┬───────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                   GO API SERVER                           │
│                                                           │
│  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐  │
│  │    Auth      │ │   REST API   │ │  Sync Engine     │  │
│  │  (JWT/RBAC)  │ │  (Handlers)  │ │  (Offline sync)  │  │
│  └──────┬──────┘ └──────┬───────┘ └──────┬───────────┘  │
│         │               │                 │              │
│  ┌──────┴───────────────┴─────────────────┴──────────┐  │
│  │              Domain / Service Layer                 │  │
│  │  Person │ Attendance │ Campaign │ Donation │ Audit  │  │
│  └──────────────────────┬────────────────────────────┘  │
│                         │                                │
│  ┌──────────────────────┴────────────────────────────┐  │
│  │              Repository Layer (Data Access)         │  │
│  └──────────────────────┬────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│  PostgreSQL   │ │  Object      │ │  Redis       │
│  (Primary DB) │ │  Storage     │ │  (Cache +    │
│               │ │  (S3/MinIO)  │ │   Sessions)  │
└──────────────┘ └──────────────┘ └──────────────┘
```

---

## Backend: Go API Server

### Why Go

| Factor | Go | Django (current) |
|--------|-----|-----------------|
| Performance | Compiled, low memory, fast cold start | Interpreted, higher memory, slower |
| Concurrency | Native goroutines; ideal for sync operations | GIL limits true parallelism |
| Deployment | Single binary; no runtime dependencies | Requires Python, pip, virtualenv |
| Type safety | Strong static typing | Dynamic typing |
| API focus | Excellent HTTP stdlib; lean API servers | Full-stack framework; overhead for API-only |
| Learning curve for AI agents | Clear, explicit code; easy to generate | Magic (ORM, signals) harder for AI to reason about |
| Container size | ~20MB binary | ~500MB+ with Python + deps |
| Long-term maintenance | Stable stdlib; minimal dependency churn | Frequent Django version upgrades |

### Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration loading
│   ├── domain/                  # Domain entities (structs)
│   │   ├── person.go
│   │   ├── attendance.go
│   │   ├── campaign.go
│   │   ├── donation.go
│   │   ├── consent.go
│   │   └── audit.go
│   ├── handler/                 # HTTP handlers (controllers)
│   │   ├── person.go
│   │   ├── attendance.go
│   │   ├── campaign.go
│   │   ├── auth.go
│   │   ├── sync.go
│   │   └── report.go
│   ├── service/                 # Business logic
│   │   ├── person.go
│   │   ├── attendance.go
│   │   ├── campaign.go
│   │   ├── auth.go
│   │   ├── sync.go
│   │   └── report.go
│   ├── repository/              # Data access layer
│   │   ├── person.go
│   │   ├── attendance.go
│   │   └── ...
│   ├── middleware/               # HTTP middleware
│   │   ├── auth.go
│   │   ├── audit.go
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   └── campus.go
│   └── sync/                    # Offline sync engine
│       ├── engine.go
│       ├── conflict.go
│       └── protocol.go
├── migrations/                  # SQL migration files
├── scripts/                     # Setup and utility scripts
├── Dockerfile
├── go.mod
└── go.sum
```

### Key Libraries (minimal dependency approach)

| Purpose | Library | Rationale |
|---------|---------|-----------|
| HTTP Router | `net/http` + `chi` | Lightweight, stdlib-compatible |
| Database | `pgx` | High-performance PostgreSQL driver |
| Migrations | `golang-migrate` | SQL-based migrations |
| JWT | `golang-jwt` | Standard JWT handling |
| Validation | `go-playground/validator` | Struct tag validation |
| Logging | `slog` (stdlib) | Structured logging, built-in since Go 1.21 |
| Configuration | `envconfig` or `viper` | Environment-based config |
| Testing | `testing` (stdlib) + `testify` | Standard Go testing |
| UUID | `google/uuid` | UUID generation |

---

## Frontend: React PWA

### Why React

| Factor | React | Django Templates (current) |
|--------|-------|---------------------------|
| Offline support | Service Worker + IndexedDB native | Not possible without full rewrite |
| Mobile UX | Full control over responsive design | Admin UI is desktop-centric |
| Component reuse | Strong component model | Template inheritance is limited |
| State management | Local state + sync state in IndexedDB | Server-dependent |
| Ecosystem | Massive; PWA tooling is mature | Limited to Django ecosystem |
| AI code generation | Well-understood by coding agents | Django templates are less predictable |

### Project Structure

```
frontend/
├── public/
│   ├── manifest.json            # PWA manifest
│   ├── service-worker.js        # Offline caching
│   └── index.html
├── src/
│   ├── api/                     # API client layer
│   │   ├── client.ts            # HTTP client with auth
│   │   ├── person.ts
│   │   ├── attendance.ts
│   │   └── sync.ts
│   ├── components/              # Reusable UI components
│   │   ├── forms/
│   │   ├── layout/
│   │   ├── tables/
│   │   └── common/
│   ├── pages/                   # Route-level components
│   │   ├── Dashboard.tsx
│   │   ├── PersonList.tsx
│   │   ├── PersonForm.tsx
│   │   ├── TriageForm.tsx
│   │   ├── AttendanceForm.tsx
│   │   ├── CampaignList.tsx
│   │   └── Reports.tsx
│   ├── hooks/                   # Custom React hooks
│   │   ├── useAuth.ts
│   │   ├── useOffline.ts
│   │   └── useSync.ts
│   ├── store/                   # State management
│   │   ├── auth.ts
│   │   ├── offline.ts
│   │   └── sync.ts
│   ├── offline/                 # Offline-first logic
│   │   ├── db.ts               # IndexedDB wrapper
│   │   ├── queue.ts            # Sync queue
│   │   └── conflict.ts         # Conflict resolution
│   ├── utils/
│   ├── types/                   # TypeScript interfaces
│   └── App.tsx
├── package.json
├── tsconfig.json
├── vite.config.ts
└── Dockerfile
```

### Key Libraries

| Purpose | Library | Rationale |
|---------|---------|-----------|
| Build | Vite | Fast builds, excellent PWA plugin |
| Routing | React Router | Standard routing |
| UI Framework | Tailwind CSS | Utility-first; mobile-first by default |
| Forms | React Hook Form | Lightweight, performant form handling |
| State | Zustand or Context API | Simple state management; no Redux overhead |
| Offline DB | Dexie.js (IndexedDB wrapper) | Clean API over IndexedDB |
| PWA | vite-plugin-pwa | Service Worker generation |
| Charts | Recharts | Simple, React-native charts |
| Signature | react-signature-canvas | Touch signature capture for consent |
| HTTP | fetch (native) | No axios needed |

---

## Database: PostgreSQL

### Why PostgreSQL

- ACID transactions for data integrity
- JSON/JSONB for flexible audit log storage
- Full-text search for person lookup
- Row-level security for campus data segregation
- Mature, free, well-supported
- Excellent Go driver support (pgx)

### Key Design Decisions

1. **UUIDs as primary keys**: All tables use UUID PKs for offline record creation
2. **Campus-scoped queries**: Most queries include `campus_id` filter; indexes support this
3. **Audit table**: Append-only `audit_logs` table with JSONB for old/new values
4. **Soft deletes**: `is_active` boolean on all entities; physical deletion only for LGPD erasure
5. **Timestamps**: `created_at`, `updated_at` on all tables with database-level defaults

---

## Object Storage

For document and file attachments (consent forms, medical records, ID copies):

- **Development**: Local filesystem or MinIO (S3-compatible)
- **Production**: AWS S3, Google Cloud Storage, or equivalent
- Files are referenced by URL in the database; never stored as BLOBs
- Presigned URLs for secure, time-limited access

---

## Authentication and Authorization

### Auth Flow

```
1. User submits email + password
2. Server validates credentials
3. Server issues JWT access token (15min) + refresh token (7 days)
4. Client stores tokens (httpOnly cookie or secure storage)
5. Each API request includes access token in Authorization header
6. Middleware validates token and extracts user/campus/role
7. Handler checks role-based permissions before processing
```

### RBAC Model

```
User → access_profile (ADMIN, COORDINATOR, PROFESSIONAL, SECRETARY, VOLUNTEER)
     → campus_id (scopes data access)

Each access_profile maps to a permission set:
  ADMIN:       all permissions, all campuses (or scoped)
  COORDINATOR: manage campaigns, teams, view reports, manage attendance
  PROFESSIONAL: create/edit own attendance records, view assigned persons
  SECRETARY:   register persons, create triage, view attendance, basic reports
  VOLUNTEER:   create triage, basic data entry (limited scope)
```

### Security Layers

| Layer | Mechanism |
|-------|-----------|
| Transport | TLS 1.3 via reverse proxy |
| Authentication | JWT with short-lived tokens |
| Authorization | RBAC middleware on every handler |
| Data isolation | Campus-scoped queries |
| Audit | Every authenticated request logged |
| Rate limiting | Per-IP and per-user limits |
| Input validation | Struct validation on all inputs |
| CORS | Explicit origin whitelist |

---

## Offline-First Approach

See `12-offline-sync-strategy.md` for the full strategy. Summary:

1. **PWA with Service Worker**: Caches application shell and static assets
2. **IndexedDB**: Local database stores person records, triages, attendances offline
3. **Sync queue**: Offline mutations are queued and replayed when online
4. **Conflict resolution**: Last-write-wins with server timestamp; conflicts flagged for manual review
5. **Background sync**: Uses Background Sync API where available; falls back to periodic polling

---

## Integration Strategy

### WordPress Portal
- Public-facing API endpoints for:
  - Campaign listings (read-only)
  - Volunteer signup forms
  - Donation submission
- Authenticated via API key (not user JWT)
- Rate-limited and scoped to public data only

### Future Integrations
- Email service (SendGrid, SES) for password recovery and notifications
- SMS gateway for event reminders (future phase)
- Government reporting APIs (future phase)

---

## Deployment Architecture

```
┌─────────────────────────────────────┐
│          Cloud Provider              │
│  (Railway / Render / AWS / GCP)      │
│                                      │
│  ┌──────────┐  ┌──────────────────┐ │
│  │  Caddy /  │  │   Go API         │ │
│  │  Nginx    │──│   Container      │ │
│  │  (proxy)  │  │   (port 8080)    │ │
│  └──────────┘  └────────┬─────────┘ │
│                          │           │
│  ┌──────────────────┐   │           │
│  │  React PWA        │   │           │
│  │  (static files    │   │           │
│  │   on CDN/proxy)   │   │           │
│  └──────────────────┘   │           │
│                          │           │
│  ┌──────────────────────┴────────┐  │
│  │  PostgreSQL (managed)          │  │
│  │  + automated backups           │  │
│  └───────────────────────────────┘  │
│                                      │
│  ┌───────────────────────────────┐  │
│  │  Redis (managed, optional)     │  │
│  └───────────────────────────────┘  │
│                                      │
│  ┌───────────────────────────────┐  │
│  │  Object Storage (S3/GCS)       │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

### Cost Optimization for NGO Budget

| Service | Free/Low-Cost Options |
|---------|----------------------|
| Hosting | Railway free tier, Render free tier, Fly.io |
| Database | Supabase free tier (PostgreSQL), Neon |
| Object Storage | Cloudflare R2 (free egress), Backblaze B2 |
| CDN | Cloudflare (free tier) |
| CI/CD | GitHub Actions (free for public repos) |
| Monitoring | Grafana Cloud free tier |
| Email | SendGrid free tier (100 emails/day) |
