# 06 - Tech Stack Evaluation

## Current Stack Assessment

### Current Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Python | 3.12 |
| Framework | Django | 5.2.6 |
| UI | Django Admin + Jazzmin | 3.0.1 |
| Database | SQLite (dev) / PostgreSQL (prod) | 15 |
| Server | Gunicorn | 23.0.0 |
| Data processing | pandas + numpy + scipy + plotly | Various |
| Container | Docker | - |
| CI/CD | GitHub Actions | - |

### Does the Current Stack Fit the New Product?

**No.** The current stack cannot meet the new product requirements for the following reasons:

1. **No offline support path**: Django is a server-rendered framework. Adding offline-first capability would require building an entirely separate API layer and frontend — effectively a full rewrite within the Django ecosystem, gaining none of Django's advantages.

2. **Admin UI is not suitable**: Django Admin, even with Jazzmin, is designed for database administrators — not for volunteers on mobile phones during a crowded social event. The UX gap is unbridgeable without a custom frontend.

3. **No API layer exists**: The application has zero REST or GraphQL endpoints. Every new requirement (mobile, PWA, WordPress integration, sync) needs an API that does not exist.

4. **Heavy dependencies for minimal value**: pandas, numpy, scipy, and plotly add ~200MB to the container for a single report view that could be written with SQL aggregations.

5. **Performance overhead**: Python's GIL and Django's ORM overhead are unnecessary for what is primarily a CRUD + sync API. The sync engine (handling multiple offline devices) benefits from Go's native concurrency.

6. **Deployment complexity**: Python applications require runtime, virtualenv, and careful dependency management. A Go binary is self-contained.

---

## Recommended Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Backend Language** | Go (Golang) | Performance, simplicity, single binary deployment, native concurrency |
| **HTTP Router** | chi | Lightweight, stdlib-compatible, middleware-friendly |
| **Database Driver** | pgx | High-performance native PostgreSQL driver for Go |
| **Migrations** | golang-migrate | SQL-file-based migrations; no ORM magic |
| **Auth** | Keycloak (OIDC) + coreos/go-oidc | External IAM with standard OIDC token validation |
| **Frontend Framework** | React 18+ | Component model, massive ecosystem, excellent PWA support |
| **Frontend Language** | TypeScript | Type safety, better IDE support, fewer runtime errors |
| **Build Tool** | Vite | Fast development server, excellent PWA plugin |
| **CSS Framework** | Tailwind CSS | Utility-first, mobile-first by default, small bundle |
| **Offline Storage** | Dexie.js (IndexedDB) | Clean API over IndexedDB for offline data |
| **PWA** | vite-plugin-pwa + Workbox | Service Worker generation and caching strategies |
| **Database** | PostgreSQL 16 | Relational integrity, JSON support, full-text search, RLS |
| **Object Storage** | S3-compatible (MinIO dev, S3/R2 prod) | File attachments, consent documents |
| **Cache** | Redis (optional) | Session cache, rate limiting (can defer to later phase) |
| **Container** | Docker | Consistent dev/prod environments |
| **CI/CD** | GitHub Actions | Free for public repos, already in use |
| **Reverse Proxy** | Caddy | Automatic HTTPS, simple config, Go-native |

---

## Alternative Comparison

### Backend Language

| Criteria | Go | Python/Django | Node.js/Express | Rust |
|----------|-----|--------------|-----------------|------|
| Performance | Excellent | Moderate | Good | Excellent |
| Concurrency (sync engine) | Excellent (goroutines) | Poor (GIL) | Good (event loop) | Excellent |
| Binary deployment | Yes (single binary) | No (runtime needed) | No (runtime needed) | Yes |
| Learning curve | Moderate | Low | Low | High |
| AI code generation quality | High | High | High | Moderate |
| Ecosystem maturity | High | Very High | Very High | Moderate |
| Container size | ~20MB | ~500MB+ | ~200MB+ | ~20MB |
| Hiring/volunteer dev pool | Growing | Large | Large | Small |
| Long-term stability | Excellent (stdlib-first) | Good (frequent upgrades) | Moderate (ecosystem churn) | Good |

**Decision: Go** — Best balance of performance, deployment simplicity, concurrency support for the sync engine, and AI-assisted development. The trade-off is a smaller community compared to Python/Node, but Go's stdlib covers most needs without external dependencies.

### Frontend Framework

| Criteria | React | Vue.js | Svelte | HTMX + Django |
|----------|-------|--------|--------|---------------|
| PWA support | Excellent | Good | Good | Poor |
| Offline-first tooling | Excellent | Good | Moderate | Not viable |
| Component ecosystem | Massive | Large | Growing | N/A |
| Mobile-first design | Excellent | Excellent | Good | Limited |
| TypeScript support | Native | Native | Native | N/A |
| AI code generation | Excellent | Good | Moderate | Moderate |
| Signature capture libs | Yes (react-signature-canvas) | Yes | Limited | No |

**Decision: React + TypeScript** — Best PWA/offline tooling, largest component ecosystem, strongest AI code generation support, and the user's preferred choice.

### Database

| Criteria | PostgreSQL | MySQL | SQLite | MongoDB |
|----------|-----------|-------|--------|---------|
| ACID compliance | Full | Full | Full | Partial |
| JSON support | Excellent (JSONB) | Good | Limited | Native |
| Full-text search | Built-in | Built-in | Extension | Built-in |
| Row-Level Security | Yes | No | No | No |
| UUID support | Native | Manual | Manual | Native (ObjectId) |
| Go driver quality | Excellent (pgx) | Good | Good | Good |
| Managed hosting options | Many (free tiers) | Many | N/A (embedded) | Limited free |
| Offline sync compatibility | Excellent | Good | Good | Good |

**Decision: PostgreSQL** — Best relational database for this use case. RLS supports campus data segregation. JSONB is ideal for audit log storage. Free managed options (Supabase, Neon) fit the NGO budget.

### CSS Framework

| Criteria | Tailwind CSS | Bootstrap | Material UI | Chakra UI |
|----------|-------------|-----------|-------------|-----------|
| Mobile-first | Default | Default | Partial | Yes |
| Bundle size | Small (purged) | Moderate | Large | Moderate |
| Customization | Excellent | Good | Complex | Good |
| Learning curve | Low | Low | Moderate | Low |
| AI code generation | Excellent | Good | Moderate | Good |

**Decision: Tailwind CSS** — Smallest bundle, best mobile-first support, excellent AI code generation (utility classes are unambiguous).

### Identity Provider

| Criteria | Custom JWT | Keycloak | Auth0 | AWS Cognito |
|----------|-----------|----------|-------|-------------|
| Cost | Free (dev time risk) | Free (self-hosted) | Free tier limited (7k MAU) | Pay-per-MAU |
| MFA support | Must build | Built-in (TOTP, WebAuthn) | Built-in | Built-in |
| Password policy | Must build | Built-in (configurable) | Built-in | Built-in |
| Account lockout | Must build | Built-in | Built-in | Built-in |
| SSO readiness | None | Built-in (SAML, OIDC) | Built-in | Built-in |
| OIDC/OAuth2 | Must build | Native | Native | Native |
| Self-hosted | N/A | Yes | No | No |
| Open source | N/A | Yes (Apache 2.0) | No | No |
| RAM requirement | N/A | 512MB-1GB | N/A | N/A |
| Vendor lock-in | None | Low (OIDC standard) | Medium | High (AWS) |
| NGO budget fit | Risky (dev cost) | Yes | Risky (growth) | Risky (growth) |
| Brute-force protection | Must build | Built-in | Built-in | Built-in |
| User federation | Must build | Built-in (LDAP, social) | Built-in | Built-in |

**Decision: Keycloak** — The only open-source, self-hosted option that is free at any scale. Fits the $5-50/month NGO budget. Standard OIDC protocol means the application is not locked to Keycloak and can migrate to any OIDC provider in the future.

---

## Migration Path from Current Stack

### Phase 0: Documentation (current)
- Complete architecture documentation
- No code changes to current system

### Phase 1: Go API Foundation
- Set up Go project structure
- Implement database schema and migrations
- Build auth (JWT + RBAC)
- Implement Person, Attendance, Campaign APIs
- Keep Django running in parallel

### Phase 2: React PWA
- Set up React + TypeScript + Vite project
- Build core UI components
- Implement offline storage with Dexie.js
- Connect to Go API
- Build triage and attendance forms

### Phase 3: Data Migration
- Write migration scripts (Python → Go or standalone)
- Migrate Person, Atendimento, User records
- Validate data integrity
- Decommission Django application

### What Can Be Reused from Current Stack

| Asset | Reuse Strategy |
|-------|---------------|
| PostgreSQL schema concepts | Reference for Go migration design |
| Permission taxonomy (17 permissions, 5 modules) | Port directly to Go RBAC |
| Audit logging pattern | Implement equivalent in Go middleware |
| Docker Compose structure | Adapt for Go + React + PostgreSQL |
| GitHub Actions CI | Adapt for Go build + test |
| Business rules (role hierarchy, workflow states) | Implement in Go service layer |

### What Cannot Be Reused

| Asset | Reason |
|-------|--------|
| Django views and templates | Replaced by React frontend |
| Django Admin customization | Replaced by custom UI |
| Django ORM models | Replaced by Go repository layer |
| pandas/numpy/scipy/plotly | Replaced by SQL aggregations + lightweight JS charts |
| Django signals | Replaced by explicit service layer calls |
| Django middleware | Replaced by Go middleware |
