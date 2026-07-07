# Runbook: Backup & Disaster Recovery

**Story:** S12.3 (Sprint 11 — Integration & Hardening)
**Scope:** PostgreSQL 16 logical backups, restore drills, and the disaster-recovery (DR) procedure for the Chesed platform.
**Companion doc:** [`docs/14-deployment-strategy.md`](../14-deployment-strategy.md) — section "Backup and Disaster Recovery" defines the authoritative RPO/RTO targets and the managed-snapshot / PITR layer. This runbook is the operational, script-level complement and must stay consistent with it.

---

## 1. Philosophy: external trigger, no in-app scheduler

Backups are triggered **externally** — by cron, a scheduled CI job, or a managed scheduler — exactly like the retention sweep (`POST /admin/retention/run`), which is invoked by an external cron rather than an in-app timer. The application deliberately contains **no** backup scheduler. This keeps the API stateless, keeps backup credentials out of the app, and lets operators run backups from a host that has the owner/admin database credentials.

The committable, testable piece lives in version control:

| Artifact | Purpose |
|----------|---------|
| [`scripts/backup.sh`](../../scripts/backup.sh) | `backup` (dump + checksum + prune) and `drill` (restore-verify) subcommands |
| [`scripts/restore.sh`](../../scripts/restore.sh) | Guarded, checksum-verified `pg_restore` into a target database |
| This runbook | Schedule, DR set, restore procedure, verification checklist |

---

## 2. The DR recovery set

A full recovery of Chesed requires **all** of the following. A database dump alone is not a complete backup.

1. **PostgreSQL logical dump** — `chesed-<UTC-timestamp>.dump` (custom format) plus its `.sha256` sidecar, produced by `scripts/backup.sh backup`. Includes application data **and** the co-located Keycloak database (users, sessions) when Keycloak shares the instance (see `docs/14`).
2. **Keycloak realm configuration** — [`keycloak/realm-export.json`](../../keycloak/realm-export.json), version-controlled in the repo. Recovery imports this JSON into a fresh Keycloak instance; the database dump restores the realm's runtime data.
3. **Object storage (documents)** — files stored in S3/MinIO (`S3_BUCKET`, e.g. `chesed-docs`). These are **not** in the SQL dump. In production this is covered by the object store's own versioning/replication; note it here so DR planning accounts for it. (Doc-only for Phase 1; wire bucket replication when the managed object store is provisioned.)
4. **Infrastructure as config** — `docker-compose.prod.yml`, `.env.prod` (secrets restored from the secrets manager, **never** from a backup), and TLS certs (re-issued via `scripts/init-letsencrypt.sh`).

---

## 3. RPO / RTO targets

These mirror `docs/14-deployment-strategy.md`. The logical dump in this runbook is the **weekly logical backup** tier; the managed snapshot + WAL/PITR tiers are provided by the managed PostgreSQL platform.

| Scenario | RTO | RPO | Primary mechanism |
|----------|-----|-----|-------------------|
| Application server failure | 5 min (auto-restart) | 0 (stateless) | Redeploy / container restart |
| Database failure | 15 min | 1 hour | Managed snapshot / PITR restore |
| Complete infrastructure loss | 2 hours | 24 hours | Rebuild + latest logical dump |

| Backup tier | Frequency | Retention | Owner |
|-------------|-----------|-----------|-------|
| Automated snapshot | Daily | 7 days | Managed PostgreSQL (Supabase/Railway) |
| **Logical dump (`backup.sh`)** | **Weekly** | **30 days** (`BACKUP_RETENTION_DAYS`) | This runbook / cron |
| Point-in-time recovery (WAL) | Continuous | 7 days | Managed PostgreSQL |

---

## 4. Connecting as the owner/admin (not the RLS role)

Chesed enforces row-level security via a **non-owner** role, `chesed_app` (Sprint 9.1). Backups and restores **must** use the **owner/admin** connection so the dump captures every row regardless of RLS and the restore can drop/recreate objects.

Provide the owner connection via **one** of:

- `ADMIN_DATABASE_URL` (preferred; alias `DATABASE_URL` also accepted), e.g.
  `postgres://chesed:chesed@db:5432/chesed?sslmode=disable`
- discrete libpq variables: `PGHOST` `PGPORT` `PGUSER` `PGDATABASE` `PGPASSWORD`

Never hardcode credentials — inject them from the secrets manager / environment at runtime.

---

## 5. Scheduled backup (external cron)

Run the backup from a host that has the owner credentials and `pg_dump` (PostgreSQL 16 client). Example weekly crontab entry (Sundays 03:00 UTC):

```cron
# m h dom mon dow  command
0 3 * * 0  ADMIN_DATABASE_URL="postgres://chesed:***@db:5432/chesed?sslmode=disable" \
           BACKUP_DIR="/var/backups/chesed" \
           BACKUP_RETENTION_DAYS=30 \
           /opt/chesed/scripts/backup.sh backup >> /var/log/chesed-backup.log 2>&1
```

Against the local compose stack you can run it through the `db` container's client tools, e.g.:

```bash
ADMIN_DATABASE_URL="postgres://chesed:chesed@localhost:5432/chesed?sslmode=disable" \
BACKUP_DIR="./backups" \
scripts/backup.sh backup
```

What `backup.sh backup` does, in order:

1. `pg_dump --format=custom --no-owner --no-privileges` → `${BACKUP_DIR}/chesed-<UTC-timestamp>.dump`
2. Writes a `.sha256` sidecar (`sha256sum -c` compatible).
3. Prunes dumps + sidecars older than `BACKUP_RETENTION_DAYS` (default 30).
4. Runs the **off-site copy hook** (a documented no-op unless enabled — see §7).

---

## 6. Restore drill (proves the dump is restorable)

The drill is the automated proof that a dump can actually be restored — a backup you have never restored is a hope, not a backup. It restores the **latest** dump into a **throwaway** database, runs smoke `SELECT count(*)` queries against core tables (`campus`, `person`, `triage`, `audit_log`), reports `PASS`/`FAIL`, and always drops the throwaway database.

It never touches the live database. Run it against the compose Postgres:

```bash
ADMIN_DATABASE_URL="postgres://chesed:chesed@localhost:5432/chesed?sslmode=disable" \
BACKUP_DIR="./backups" \
scripts/backup.sh drill
```

Expected tail on success:

```
... DRILL: PASS (dump ./backups/chesed-<ts>.dump is restorable and core tables are queryable)
```

This is what the `make backup-drill` target wraps so it can run in local quality gates.

---

## 7. Off-site copy (manual / env-gated)

Off-site replication is a deliberate, credentialed step and is **disabled by default** to keep the drill hermetic. To enable, set `BACKUP_OFFSITE_URI` and uncomment one transfer command in `offsite_copy_hook` inside `scripts/backup.sh`:

```bash
# S3
export BACKUP_OFFSITE_URI="s3://my-bucket/chesed/"
# aws s3 cp "$dump" "$BACKUP_OFFSITE_URI"

# or rclone
export BACKUP_OFFSITE_URI="remote:chesed-backups/"
# rclone copy "$dump" "$BACKUP_OFFSITE_URI"
```

Credentials come from the environment / instance role / secrets manager — **never** from the script. Copy both the `.dump` and its `.sha256`.

---

## 8. Restore procedure (destructive)

`scripts/restore.sh` performs the actual recovery. It is **destructive** and refuses to run without `--confirm`. It verifies the `.sha256` **before** touching the target.

```bash
scripts/restore.sh \
  --confirm \
  --dump ./backups/chesed-<UTC-timestamp>.dump \
  --target "postgres://chesed:chesed@localhost:5432/chesed?sslmode=disable"
```

- Target precedence: `--target` > `RESTORE_TARGET_URL` > `ADMIN_DATABASE_URL` > `DATABASE_URL`.
- Under the hood: `pg_restore --clean --if-exists --no-owner --no-privileges -d "$TARGET_URL" <dump>` (idempotent on re-run).
- `--skip-checksum` exists for emergencies but is **not** recommended.

### Full DR sequence (complete infrastructure loss)

1. **Provision** the managed PostgreSQL 16 instance (or the compose `db`) and the `chesed` owner role.
2. **Restore data:** run `scripts/restore.sh --confirm --dump <latest> --target <admin-url>`.
3. **Re-apply schema roles/grants:** run database migrations (`make migrate-up`) to recreate the `chesed_app` RLS role and grants — the dump uses `--no-owner --no-privileges`, so RLS wiring comes from migrations.
4. **Keycloak:** stand up a fresh Keycloak, import [`keycloak/realm-export.json`](../../keycloak/realm-export.json); the restored database supplies realm runtime data.
5. **Object storage:** restore/attach the documents bucket (versioned replica in production).
6. **Secrets & TLS:** inject `.env.prod` from the secrets manager; re-issue certs via `scripts/init-letsencrypt.sh`.
7. **Verify:** hit `GET /api/v1/health`, confirm audit logs and core table counts match the pre-incident drill numbers.

---

## 9. Mapping to the managed cloud layer (doc-only)

| Layer | Provided by | Managed here? |
|-------|-------------|---------------|
| Daily snapshot (7-day) | Managed PostgreSQL platform | No — platform config |
| Continuous WAL / PITR (7-day) | Managed PostgreSQL platform | No — platform config |
| Weekly logical dump (30-day) | `scripts/backup.sh` + cron | **Yes** |
| Restore verification | `scripts/backup.sh drill` / this runbook | **Yes** |
| Object-storage replication | S3/MinIO versioning | No — platform config |

The scripts in this runbook are the portable, provider-independent safety net that survives even a full loss of the managed platform. The snapshot/PITR tiers give the low-RPO fast path; the logical dump gives the escape hatch.

---

## 10. Monthly restore-verification checklist

Perform on the first business day of each month (aligns with `docs/14` "Monthly: restore backup and verify data integrity").

- [ ] Confirm the most recent weekly dump exists and its `.sha256` verifies (`sha256sum -c chesed-<ts>.dump.sha256`).
- [ ] Run `scripts/backup.sh drill` (or `make backup-drill`) → verdict is **PASS**.
- [ ] Compare drill smoke counts against expected magnitude (no unexpected zero counts for `campus` / `person`).
- [ ] Confirm dumps older than `BACKUP_RETENTION_DAYS` were pruned (retention working).
- [ ] Confirm the off-site copy exists for the latest dump (if `BACKUP_OFFSITE_URI` is configured).
- [ ] Confirm `keycloak/realm-export.json` reflects the current realm (re-export if realm config changed).
- [ ] Record the drill result, dump timestamp, and operator in the ops log.

**Quarterly:** perform a full DR drill (§8) into an isolated environment and time it against the RTO targets in §3.

---

## 11. Safety notes

- Scripts are POSIX `bash` with `set -euo pipefail`; both are idempotent (re-running a backup makes a new timestamped file; restore/drill are safe to re-run).
- The drill always drops its throwaway database via an `EXIT` trap, even on failure.
- No secrets are stored in the scripts or this runbook; all credentials come from the environment.
- Restore logs redact `user:pass@` from connection URLs.
