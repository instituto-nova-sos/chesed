# Runbook — Password Recovery Verification

**Story:** S12.4 — Password-recovery verification (Keycloak-owned configuration only)
**Scope:** A **verification** procedure proving that password recovery is fully
owned by Keycloak. There is **no application code** to write for this story —
Keycloak issues, sends, and processes the reset. This runbook confirms the
existing configuration is correct in dev and wired for production.

**Grounded on:**
- `keycloak/realm-export.json` — realm flags (`resetPasswordAllowed`, `SEND_RESET_PASSWORD`, `verifyEmail`)
- `docker-compose.prod.yml` — production SMTP wiring (`KC_SPI_EMAIL_SENDER_DEFAULT_*`)
- `keycloak/init-realm.sh` — dev SMTP (Mailpit) and dev-only overrides
- `docker-compose.yml` — dev Mailpit service (inbox on `:8025`, SMTP on `:1025`)
- `docs/20-keycloak-configuration.md` — realm / SMTP configuration reference

---

## 0. Ownership statement (read first)

> **Keycloak owns 100% of password recovery.** Per `CLAUDE.md` architecture
> guardrails, all credential handling is Keycloak's responsibility: the Go API
> never implements login, password hashing, token issuance, or password-reset
> email. The "Forgot password?" flow is entirely a Keycloak realm feature over
> Keycloak's SMTP transport.
>
> Therefore S12.4 is a **verification task, not an implementation task.**

### Explicitly out of scope this sprint

- **No application-side email code.** The backend does not send, template, or
  intercept any email. Do not add mail libraries, SMTP clients, or reset
  endpoints to the Go API.
- **No event reminders.** There is explicitly no event-reminder work in this
  sprint; any reminder/notification feature is **deferred** and must not be
  built here.

---

## 1. Confirm realm flags in `keycloak/realm-export.json`

The exported realm (the source of truth, imported at Keycloak startup via
`start --import-realm`) must contain the following. Verify with a quick grep or
by inspecting the file:

```bash
grep -nE '"resetPasswordAllowed"|"verifyEmail"|"SEND_RESET_PASSWORD"' keycloak/realm-export.json
```

Expected:

| Setting | Expected value | Meaning |
|---------|----------------|---------|
| `resetPasswordAllowed` | `true` | Renders the "Forgot password?" link on the login page and enables the reset-credentials flow. |
| `verifyEmail` | `true` | Production requires verified email (reset links go to a verified address). |
| `enabledEventTypes` includes `SEND_RESET_PASSWORD` | present | The realm audits when a reset email is dispatched. `RESET_PASSWORD` and `UPDATE_PASSWORD` are also enabled, capturing completion. |

The `SEND_RESET_PASSWORD` required action is part of Keycloak's built-in
reset-credentials flow; it is triggered automatically when a user submits the
"Forgot password?" form. Presence of the event type in `enabledEventTypes`
confirms the realm records the dispatch for the audit trail.

**Pass criteria:** all three rows match. If any differ in a running instance,
re-import `realm-export.json` (do not hand-edit prod through the admin console
without exporting the change back to the file — `CLAUDE.md` guardrail 11).

---

## 2. Confirm production SMTP wiring

In production, Keycloak's email transport is configured via environment in
`docker-compose.prod.yml` (service `keycloak`), mapped from `.env.prod`:

```
KC_SPI_EMAIL_SENDER_DEFAULT_HOST               ← SMTP_HOST
KC_SPI_EMAIL_SENDER_DEFAULT_PORT               ← SMTP_PORT
KC_SPI_EMAIL_SENDER_DEFAULT_FROM               ← SMTP_FROM
KC_SPI_EMAIL_SENDER_DEFAULT_FROM_DISPLAY_NAME  ← SMTP_FROM_DISPLAY_NAME
KC_SPI_EMAIL_SENDER_DEFAULT_USER               ← SMTP_USER
KC_SPI_EMAIL_SENDER_DEFAULT_PASSWORD           ← SMTP_PASSWORD
KC_SPI_EMAIL_SENDER_DEFAULT_ENABLE_STARTTLS    ← SMTP_STARTTLS
KC_SPI_EMAIL_SENDER_DEFAULT_ENABLE_SSL         ← SMTP_SSL
```

**Verify (production server):**

```bash
# All SMTP_* set in .env.prod (no placeholders left):
grep -E '^SMTP_' .env.prod

# Confirm the running Keycloak container received them:
docker compose -f docker-compose.prod.yml --env-file .env.prod \
  exec keycloak env | grep KC_SPI_EMAIL_SENDER_DEFAULT_
```

Provider notes: STARTTLS on port 587 (`SMTP_STARTTLS=true`, `SMTP_SSL=false`) or
SSL on 465 (invert). Common providers: SendGrid (`smtp.sendgrid.net:587`, user
`apikey`), AWS SES, Mailgun. SMTP credentials come from `.env.prod` /
secrets manager only — never hard-coded (`CLAUDE.md` MUST NOT #9). Full setup in
`docs/20-keycloak-configuration.md`.

**Live send test (production):** use the Keycloak admin console →
*Realm settings → Email → Test connection* to send a probe email to the admin
address, confirming the transport works before relying on user-initiated resets.

---

## 3. Dev verification walkthrough (Mailpit)

In development, Keycloak sends to **Mailpit** instead of a real provider. The dev
compose stack (`docker-compose.yml`) runs Mailpit with the web inbox on
**http://localhost:8025** and SMTP on `:1025`; `keycloak/init-realm.sh` points
the realm's SMTP server at `host=mailpit, port=1025`.

> Dev note: `init-realm.sh` disables `verifyEmail` and MFA and seeds test users
> for convenience. `resetPasswordAllowed` stays `true`, so the reset flow is
> fully exercisable in dev. Production keeps `verifyEmail: true` and conditional
> MFA.

### Steps

1. **Start the dev stack** and run the realm init (idempotent):

   ```bash
   docker compose up -d
   ./keycloak/init-realm.sh          # configures Mailpit SMTP + seeds users
   ```

   Confirm Mailpit SMTP is registered on the realm (init prints
   `SMTP configured (host=mailpit, port=1025)`).

2. **Trigger "Forgot password"** from the login page. Open the app login
   (redirects to Keycloak) or go directly to the account/login URL for the
   `chesed` realm, then click **"Forgot password?"** and submit a seeded user's
   email, e.g. `coordinator@chesed.test`.

   - Keycloak looks up the user and (if found) dispatches a reset email. It
     returns the same neutral "check your email" message whether or not the
     address exists — this non-enumeration behavior is expected and correct.

3. **Check the Mailpit inbox** at **http://localhost:8025**. A new message from
   the configured `from` address (dev: `noreply@chesed.test`) titled for a
   password reset should appear within a few seconds. Open it and click (or copy)
   the reset link.

4. **Complete the reset.** The link opens Keycloak's *Update Password* form.
   Set a new password that satisfies the realm policy
   (`length(8) and digits(1) and lowerCase(1) and notUsername and
   passwordHistory(3)` — from `realm-export.json`). Submit.

5. **Confirm login with the new password.** Log in as the user with the new
   credentials to prove the reset took effect.

### Dev pass criteria

- [ ] "Forgot password?" link is present on the Keycloak login page
      (proves `resetPasswordAllowed: true`).
- [ ] A reset email arrives in Mailpit at `:8025` within seconds
      (proves SMTP wiring + `SEND_RESET_PASSWORD`).
- [ ] The reset link opens Keycloak's Update Password form and enforces the
      realm password policy.
- [ ] Login with the new password succeeds.
- [ ] No application-side (Go API) code participated at any point.

---

## 4. Production pass criteria

- [ ] `realm-export.json` flags verified (§1): `resetPasswordAllowed: true`,
      `verifyEmail: true`, `SEND_RESET_PASSWORD` in `enabledEventTypes`.
- [ ] `KC_SPI_EMAIL_SENDER_DEFAULT_*` present in the running Keycloak container,
      sourced from `.env.prod` `SMTP_*` (§2).
- [ ] Keycloak admin *Test connection* email delivered successfully.
- [ ] A real end-to-end reset (to a verified test mailbox) delivers a link and
      completes a password change.
- [ ] Confirmed: no application email code and no event-reminder work shipped
      this sprint (both out of scope / deferred).

---

## 5. References

- `keycloak/realm-export.json` — realm flags & password policy (source of truth)
- `docker-compose.prod.yml` — `keycloak` service SMTP env wiring
- `keycloak/init-realm.sh` — dev Mailpit SMTP + dev overrides
- `docs/20-keycloak-configuration.md` — realm & SMTP configuration
- `deploy/.env.prod.template` — `SMTP_*` variable definitions
- `docs/runbooks/production-deployment.md` — where this verification fits in the deploy checklist
