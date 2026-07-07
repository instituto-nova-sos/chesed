# Runbook — Production Deployment

**Story:** S11.6 — Production deployment runbook
**Scope:** Step-by-step production deployment over the existing Chesed deployment
mechanics. This runbook documents how to *operate* the deployment tooling that
already lives in the repository; it does not introduce new deployment code.

**Grounded on:**
- `docker-compose.prod.yml` — production topology (db, keycloak, migrate, api, frontend, certbot)
- `scripts/deploy.sh` — server-side deploy driver (git checkout → TLS bootstrap → build → up → health checks)
- `scripts/init-letsencrypt.sh` — Let's Encrypt bootstrap
- `scripts/ssl-renew.sh` — nginx reload after certbot renewal
- `docker/postgres/init-app-role.sh` — creates the `chesed_app` RLS role
- `.github/workflows/deploy.yml` — the PAUSED CD workflow
- `deploy/.env.prod.template` — full production env template
- `docs/14-deployment-strategy.md` — environments, hosting options, backups, monitoring
- `docs/20-keycloak-configuration.md` — realm / SMTP configuration

---

## 0. Execution model — cloud deployment is MANUAL (read first)

> **The autonomous delivery pipeline CANNOT deploy to production.**
>
> Per `CLAUDE.md` ("Autonomous Delivery & Push Boundary"), the AI agent and the
> `make deliver` pipeline stop at `READY-FOR-PR`. They never `git push`, open a
> PR, merge, or reach any cloud environment — the GitHub PAT has no push/PR/merge
> permission. **Every step in this runbook is performed by a human operator** on
> the production server (or, once re-enabled, by the GitHub Actions CD workflow
> that a human explicitly triggers). Nothing here is automated by the agent.

The deployment flow, end to end:

```
human operator on prod server
        │
        ▼
scripts/deploy.sh <commit-sha>
        ├─ git fetch + checkout <commit-sha>
        ├─ first deploy only → scripts/init-letsencrypt.sh   (TLS bootstrap)
        ├─ docker compose build
        ├─ docker compose up -d
        │       └─ migrate service runs migrations, then api starts
        ├─ health-check retry loop  (nginx :80  +  API https://localhost/api/v1/health)
        └─ docker image prune
```

---

## 1. Prerequisites

### 1.1 Server provisioning

- A Linux host (VM or dedicated) reachable on the public internet, ports **80**
  and **443** open inbound. Recommended: 2 vCPU / 4 GB RAM minimum — the compose
  memory limits sum to ~2.3 GB (db 512M, keycloak 1G, api 256M, frontend 128M,
  migrate 128M, certbot 64M) and Keycloak needs headroom.
- SSH access for the operator (and, if the CD workflow is re-enabled, a deploy
  user + key — see §8).
- The repository cloned to a working directory on the server. `deploy.sh` assumes
  the repo lives at a fixed path; the paused CD workflow expects `/opt/chesed`.
  If you clone elsewhere, adjust the CD workflow's `cd` step accordingly.

### 1.2 Docker

- Docker Engine + the Docker Compose v2 plugin (`docker compose`, not the legacy
  `docker-compose`). Verify:

  ```bash
  docker --version
  docker compose version
  ```

- The operator's user must be able to run `docker` (in the `docker` group or via
  `sudo`). `init-letsencrypt.sh` writes into named volumes and is typically run
  with `sudo`.

### 1.3 DNS / domain

- An A (and optionally AAAA) record for `DOMAIN_NAME` pointing at the server's
  public IP **before the first deploy**. Let's Encrypt validates the domain over
  HTTP-01, so DNS must resolve and port 80 must reach nginx.
- If Keycloak is served under `/auth` on the same domain (the template default),
  the reverse proxy config in `frontend/nginx.conf` handles the path routing —
  no extra DNS record is needed.

---

## 2. Secrets & environment checklist

Copy the template and fill in **every** value. Never commit the result.

```bash
cp deploy/.env.prod.template .env.prod
# edit .env.prod on the server; generate secrets with: openssl rand -base64 32
chmod 600 .env.prod
```

`scripts/deploy.sh` and `scripts/init-letsencrypt.sh` both `source .env.prod`
from the repo root and abort if `DOMAIN_NAME` (and, for TLS, `CERTBOT_EMAIL`)
are missing.

### Required variables (must all be set)

| Group | Variable | Notes |
|-------|----------|-------|
| TLS / domain | `DOMAIN_NAME` | Public hostname; DNS must point here. |
| TLS / domain | `CERTBOT_EMAIL` | Real inbox for expiry notices. |
| PostgreSQL | `POSTGRES_USER` | Owner role. |
| PostgreSQL | `POSTGRES_PASSWORD` | **Secret.** Owner role password. |
| PostgreSQL | `POSTGRES_DB` | Database name. |
| PostgreSQL | `POSTGRES_APP_PASSWORD` | **Secret. Sprint 11 checklist item.** Password for the non-owner RLS role `chesed_app`. Do NOT rely on the dev default. |
| Keycloak | `KEYCLOAK_ADMIN` | Admin username. |
| Keycloak | `KEYCLOAK_ADMIN_PASSWORD` | **Secret.** |
| Keycloak | `KC_HOSTNAME` | Public Keycloak URL (e.g. `https://domain/auth`). |
| SMTP | `SMTP_HOST` / `SMTP_PORT` | Keycloak email transport. |
| SMTP | `SMTP_FROM` / `SMTP_FROM_DISPLAY_NAME` | Sender identity. |
| SMTP | `SMTP_USER` / `SMTP_PASSWORD` | **Secret.** |
| SMTP | `SMTP_STARTTLS` / `SMTP_SSL` | Exactly one true. |
| API | `SERVER_PORT` | Default 8080. |
| API | `DATABASE_URL` | Owner connection (migrate + BypassRLS admin pool). |
| API | `APP_DATABASE_URL` | Non-owner `chesed_app` connection (RLS-subject). |
| API | `KEYCLOAK_URL` / `KEYCLOAK_REALM` / `KEYCLOAK_CLIENT_ID` | OIDC settings. |
| API | `LOG_LEVEL` | `info` in prod. |
| API | `OIDC_SKIP_ISSUER_CHECK` | **MUST be `false` in prod.** |
| Object storage | `S3_ENDPOINT` / `S3_BUCKET` | S3-compatible target. |
| Object storage | `S3_ACCESS_KEY` / `S3_SECRET_KEY` | **Secret. The API will not boot without these** (`config.go` marks them `required`). See §2.1. |
| Object storage | `S3_USE_SSL` | `true` for managed cloud storage. |
| Public API (Sprint 11) | `PUBLIC_CORS_ORIGINS` | **Sprint 11.** Comma-separated WordPress origin allowlist. |
| Public API (Sprint 11) | `HSTS_ENABLED` | **Sprint 11.** `true` in prod. |
| Public API (Sprint 11) | `PUBLIC_RATE_LIMIT_RPM` | **Sprint 11.** Public-API requests-per-minute cap. |
| Frontend (build args) | `VITE_KEYCLOAK_URL` / `VITE_KEYCLOAK_REALM` / `VITE_KEYCLOAK_CLIENT_ID` / `VITE_API_BASE_URL` | Baked into the bundle at build time. |

### 2.1 Known integration gaps to confirm before deploy

Two variable sets are required by the application but are **not yet wired into
`docker-compose.prod.yml`'s `api` service**. Confirm they are injected (extend
`api.environment` or add a compose override) or the deploy will fail / misbehave:

1. **Object storage credentials** (`S3_ACCESS_KEY`, `S3_SECRET_KEY`, etc.) —
   `config.Load()` validates these as `required`; the API container exits at boot
   if they are absent.
2. **Sprint 11 public-API variables** (`PUBLIC_CORS_ORIGINS`, `HSTS_ENABLED`,
   `PUBLIC_RATE_LIMIT_RPM`) — the current `api` in `docker-compose.prod.yml`
   uses the hardcoded dev CORS origin. Wire these into the `api` service as they
   are integrated so the WordPress-facing surface applies the correct allowlist,
   HSTS, and throttle.

Treat both as pre-deploy verification items; they do not change the deployment
*procedure* below.

---

## 3. TLS bootstrap (first deploy only)

TLS solves a chicken-and-egg problem: nginx needs a certificate to start on 443,
but certbot needs nginx on 80 to answer the ACME challenge. `init-letsencrypt.sh`
resolves it in five steps (dummy cert → start nginx → drop dummy → request real
cert → reload nginx).

You normally do **not** run this by hand: `scripts/deploy.sh` detects a missing
`/etc/letsencrypt/live/chesed/fullchain.pem` and invokes it automatically on the
first deploy. To run it standalone (e.g. re-bootstrap after wiping the
`certbot-conf` volume):

```bash
# Test against Let's Encrypt STAGING first to avoid rate limits:
LETSENCRYPT_STAGING=1 sudo bash scripts/init-letsencrypt.sh
# Then the real run:
sudo bash scripts/init-letsencrypt.sh
```

Requires `DOMAIN_NAME` and `CERTBOT_EMAIL` in `.env.prod`, DNS resolving to the
host, and port 80 reachable. The certificate is issued under cert-name `chesed`.

**Renewal** is automatic: the `certbot` service runs `certbot renew` every 12h.
nginx must be reloaded to pick up a renewed cert — add `scripts/ssl-renew.sh` to
the host crontab (twice daily):

```cron
0 0,12 * * * /path/to/chesed/scripts/ssl-renew.sh >> /var/log/chesed-ssl-renew.log 2>&1
```

---

## 4. First deploy vs. subsequent deploys

Both use the same command. `deploy.sh` branches internally on whether TLS certs
already exist.

```bash
# From the repo root on the production server:
bash scripts/deploy.sh <commit-sha>     # deploy a specific reviewed commit
bash scripts/deploy.sh                   # defaults to latest origin/main
```

**Always pass the explicit `<commit-sha>`** you validated (the commit that
reached `READY-FOR-PR` and was merged to `main` by a human). This is the exact
argument the CD workflow passes (`bash scripts/deploy.sh ${{ github.sha }}`).

What `deploy.sh` does, in order:

1. Aborts if `.env.prod` or `DOMAIN_NAME` is missing.
2. `git fetch origin main` and checks out the target SHA (or `origin/main`).
3. **First deploy only:** detects no cert under `/etc/letsencrypt/live/chesed`
   and runs `init-letsencrypt.sh`. On subsequent deploys this is skipped.
4. `docker compose -f docker-compose.prod.yml --env-file .env.prod build`.
5. `... up -d` — starts the whole stack. Startup ordering is enforced by
   `depends_on` + healthchecks: `db` healthy → `keycloak` healthy and `migrate`
   completes → `api` starts → `frontend` starts.
6. Runs the health-check loops (§6).
7. `docker image prune -f`.

**Zero-config subsequent deploy:** just re-run `deploy.sh <new-sha>`. Compose
rebuilds changed images and recreates only the affected containers; the `pgdata`
and `certbot-conf` volumes persist, so data and certificates survive.

---

## 5. Database migration application

Migrations are **not** a manual step. The one-shot `migrate` service runs
`sh ./migrations/run.sh` against `DATABASE_URL` (the **owner** connection, so it
bypasses RLS) after `db` is healthy and **before** `api` starts:

```
api.depends_on:
  migrate: { condition: service_completed_successfully }
```

If `migrate` exits non-zero, `api` never starts and the deploy fails at the API
health check. Inspect migration output with:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod logs migrate
```

The `chesed_app` RLS role is created separately at **first DB initialization**
by `docker/postgres/init-app-role.sh` (mounted into
`/docker-entrypoint-initdb.d/`), using `POSTGRES_APP_PASSWORD`. RLS grants and
policies themselves are applied by migration `000028`. Because init scripts only
run on an empty `pgdata` volume, rotating `POSTGRES_APP_PASSWORD` on an existing
database requires `ALTER ROLE chesed_app PASSWORD ...` manually (the script's
`ALTER` branch also runs it on re-init).

---

## 6. Health-check verification

`deploy.sh` already gates success on these two loops (20 retries, 10s apart).
After a deploy, confirm manually as well:

```bash
# Frontend (nginx) — plain HTTP health endpoint:
curl -sf http://localhost/nginx-health && echo "frontend OK"

# API through nginx over TLS (-k tolerates the localhost cert):
curl -sfk https://localhost/api/v1/health && echo "api OK"

# From outside the server (what real users and the CD external check hit):
curl -sf "https://$DOMAIN_NAME/api/v1/health"
curl -sf -o /dev/null "https://$DOMAIN_NAME/"
```

A healthy API returns `200 { "status": "ok", "db": "ok", "version": "..." }`
(see `docs/14-deployment-strategy.md` → Health Checks). If a check fails,
`deploy.sh` dumps the last 50 lines of `frontend` / `api` logs and 30 of
`migrate`; reproduce with:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
docker compose -f docker-compose.prod.yml --env-file .env.prod logs --tail=100 api
docker compose -f docker-compose.prod.yml --env-file .env.prod logs --tail=100 keycloak
```

---

## 7. Post-deploy: Keycloak realm & SMTP verification

The realm is imported on Keycloak startup (`start --import-realm` from
`keycloak/realm-export.json`). Production settings that MUST be active (they are
already in the exported realm): `verifyEmail: true`, `resetPasswordAllowed:
true`, browser flow `Browser with Conditional MFA`. The dev-only overrides in
`keycloak/init-realm.sh` (disable email verification / MFA, seed test users)
must **not** be run against production.

Verify SMTP and password recovery per
`docs/runbooks/password-recovery-verification.md` and the realm/SMTP setup in
`docs/20-keycloak-configuration.md`.

---

## 8. Re-enabling the paused GitHub Actions deploy workflow (manual)

CI/CD is **intentionally paused** (`CLAUDE.md` → "CI/CD Status — PAUSED",
2026-06-03). `.github/workflows/deploy.yml` currently triggers only on
`workflow_dispatch`; the `push` trigger is commented out. **Do not silently
re-enable it.** Re-enabling is a deliberate, human-approved operation:

1. Get explicit approval to leave cost-control mode (the policy change, not just
   the edit).
2. In `.github/workflows/deploy.yml`, restore the original trigger by
   uncommenting the `push:` block under `on:` (the banner comment at the top of
   the file documents this) and, if desired, removing `workflow_dispatch`.
3. Restore the CI workflows the `ci-gate` job polls for (`Backend`, `Frontend`,
   `Security`) by un-pausing their `on:` triggers too — `deploy.yml` waits for
   them and aborts if a required one is missing.
4. Configure the repository/environment secrets the `deploy` job needs:
   `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY`, `DEPLOY_PORT`, `DOMAIN_NAME`.
5. Ensure the `production` GitHub Environment (with any required reviewers) is
   configured — the `deploy` job runs under `environment: production`.
6. Confirm the server-side repo path matches the workflow (`cd /opt/chesed`).

Once re-enabled, a push to `main` waits for CI (`ci-gate`), SSHes to the server,
runs `bash scripts/deploy.sh <sha>`, and performs an external health check
against `https://$DOMAIN_NAME/api/v1/health`. Until then, deploys are the manual
`scripts/deploy.sh` invocation in §4.

> Even with the workflow re-enabled, the **AI agent still never pushes, merges,
> or triggers deploys.** Re-enabling only changes what *humans/CI* can do.

---

## 9. Rollback procedure

Because deploys are immutable image builds pinned to a commit SHA, rollback is
re-deploying the previous known-good SHA.

```bash
# 1. Identify the previous good commit (from deploy logs / git history):
git log --oneline -n 10

# 2. Re-run the deploy pinned to that SHA. This rebuilds and recreates
#    containers from the older code; volumes (pgdata, certbot-conf) persist.
bash scripts/deploy.sh <previous-good-sha>

# 3. Verify health (see §6).
curl -sf "https://$DOMAIN_NAME/api/v1/health"
```

**Migration-aware rollback caveat:** `golang-migrate` migrations have `.up.sql`
and `.down.sql` pairs, but the `migrate` service only applies *up* migrations.
If the previous release predates a migration that ran in the failed release,
rolling *code* back does **not** roll the *schema* back. Options:

- **Preferred — roll forward:** fix the defect and deploy a new SHA.
- **Down migration (last resort, destructive):** apply the specific down
  migration manually against the owner connection, understanding it may drop
  columns/data. Take a database snapshot first.

Data-layer recovery (snapshots, PITR, backups) is covered in
`docs/14-deployment-strategy.md` → "Backup and Disaster Recovery" (RTO/RPO
targets, restore procedures).

**Emergency stop / restart:**

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod restart api
docker compose -f docker-compose.prod.yml --env-file .env.prod down   # stop all (volumes kept)
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d  # bring back up
```

`down` without `-v` preserves `pgdata` and `certbot-conf`. **Never** pass `-v`
in production — it destroys the database and TLS certificates.

---

## 10. Deployment checklist (operator)

- [ ] Server provisioned; Docker + Compose v2 installed; ports 80/443 open.
- [ ] DNS A/AAAA record for `DOMAIN_NAME` resolves to the server.
- [ ] `.env.prod` created from `deploy/.env.prod.template`; every variable in §2
      filled; `chmod 600`; `OIDC_SKIP_ISSUER_CHECK=false`.
- [ ] Sprint 11 vars set: `PUBLIC_CORS_ORIGINS`, `HSTS_ENABLED=true`,
      `PUBLIC_RATE_LIMIT_RPM`, plus `POSTGRES_APP_PASSWORD`.
- [ ] Object storage credentials confirmed injected into the `api` service (§2.1).
- [ ] Target commit SHA is a human-merged commit on `main`.
- [ ] Run `bash scripts/deploy.sh <sha>`; first deploy performs TLS bootstrap.
- [ ] `migrate` logs show migrations applied cleanly.
- [ ] Frontend + API health checks green (local and external).
- [ ] Keycloak realm imported; `verifyEmail`/MFA/reset settings correct (§7).
- [ ] Password-recovery verification passed
      (`docs/runbooks/password-recovery-verification.md`).
- [ ] `ssl-renew.sh` cron entry present.
- [ ] Rollback SHA noted for fast revert.
