---
name: genesis-auth
description: Use when adding authentication to a project scaffolded with genesis — login, sessions, SSO, admin users, MFA, passkeys. Asks for an auth tier (basic/standard/strict), derives the mechanism from requirements, and implements it in place following the project's existing conventions.
user-invocable: true
disable-model-invocation: true
argument-hint: "[basic | standard | strict] [google | apple]... [passkey]"
---

# genesis-auth

Adds authentication to a genesis project. The user picks a tier and states requirements; the skill derives the mechanism — never quiz the user on implementation choices like JWT vs cookie — then writes the code directly into the target project.

This skill is a guideline, not a copy job. There is no reference implementation to clone: read the target project's existing code and write auth that matches it.

**Prerequisite:** the server needs the genesis DB layer — `internal/db` with goose migrations, sqlc, and the `users` table. The genesis CLI does not generate it. If it is missing, stop and tell the user to add it first with `/genesis sqlc`.

## Tiers

| Tier | Use case | Includes |
|---|---|---|
| `basic` | Internal tools | Single admin seeded from env var, password login, cookie session in Postgres. No signup, no email. |
| `standard` | Normal apps | Email+password users with email verification, optional Google/Apple SSO, sessions in Postgres, password reset, admin/user roles. |
| `strict` | Sensitive data | `standard` + MFA (TOTP, optionally a passkey instead), rate limiting + temporary lockout, session rotation on privilege change, audit log. Beyond this, recommend an external IdP (Keycloak, Auth0, WorkOS) instead of more homegrown code. |

## Question flow

1. **Read the arguments first.** `basic`/`standard`/`strict` answers the tier; `google`/`apple` answers SSO; `passkey` answers the second factor and implies `strict`. Ask only what is still missing.
2. **Ask remaining questions in ONE AskUserQuestion call**, with the recommended option first and marked "(Recommended)":
   - Tier: `basic` | `standard (Recommended)` | `strict`
   - Clients: `browser only (Recommended)` | `also mobile/API consumers`
   - SSO providers (multiSelect): `google` | `apple` — an empty selection means no SSO.
   - Second factor (`strict` only): `TOTP or passkey (Recommended)` | `TOTP only`. Passkeys resist phishing; TOTP codes do not.

   `basic` is browser-only with no SSO: when the arguments already say `basic`, skip Clients and SSO; if the user picks `basic` in the same call, ignore those answers. Ask Second factor only when the tier is or may become `strict`; ignore it if the user picks a lower tier. If the arguments combine `basic` with a provider, or `basic`/`standard` with `passkey`, flag the conflict in the confirmation.
3. **Derive the mechanism — do not ask about it:**
   - browser only → cookie (HttpOnly, Secure, SameSite=Lax) + server-side session token in Postgres. The web app reaches the API same-origin via the Vite proxy / nginx, so add no CORS. Lax still sends cookies on cross-site GETs: keep state changes off GET, and reject cookie-authenticated mutations with a foreign `Origin`.
   - mobile/API consumers → same cookie flow for the web, plus opaque bearer tokens from the same sessions table for mobile, and API keys for machine consumers. Not JWT: it cannot be revoked on logout, reset, or role change.
   - passkey → WebAuthn as a second factor next to TOTP, never passwordless login: the user signs in with password or SSO, then picks TOTP or a passkey at setup, and either one is enough. Passwordless adds a public login endpoint and account discovery; recommend an external IdP if the user insists.
4. **Confirm with the derived plan in prose, not more pickers.** Example: "basic tier: seeded admin, cookie sessions in Postgres, no SSO, server only (no web login UI) — proceed?" If the user overrides a mechanism (e.g. "actually JWT") or asks for the web UI, name the trade-off and honor it.

## Implementation rules

1. **Read the target project before writing anything.** Its router, config struct, migration numbering, and package layout decide the shape of the new code. Match them; do not impose a layout from this document.
2. **Extend the existing users table, don't invent a parallel one.** A genesis server already ships `internal/db/migrations/0001_create_users.sql` with a unique `username`, `password_hash`, and a `user_role` enum. Add columns and new tables in a new numbered migration. For `standard`+, add a unique `email` stored lowercased, log in by email, and set `username` to that email on signup. On a populated table, never assume usernames are emails: backfill before enforcing `NOT NULL`, and surface missing addresses and case collisions to the user instead of guessing.
3. **Follow the established layout** for a genesis server:
   - `internal/auth/` — password hashing, session/token issue and verify, provider clients. No gin types in here.
   - `internal/api/http/handler/auth.go` — login, logout, signup, reset endpoints.
   - `internal/api/http/dto/auth.go` — request/response structs.
   - `internal/api/http/middleware/auth.go` — session/bearer verification, role checks.
   - `internal/db/migrations/NNNN_*.sql` — goose `-- +goose Up` / `Down` blocks, both directions.
   - `internal/db/queries/*.sql` — sqlc named queries; run `make generate` after editing (`make dep` installs sqlc).
4. **Wire everything.** Register handlers and middleware in `SetupRoute`, add any new dependency to the `Services` struct, and construct it in `main.go`. Auth code that compiles but is never mounted looks done and isn't.
5. **Config goes through the existing mechanism.** Define `auth.Config` in `internal/auth` and compose it into the struct in `cmd/<app>/config.go`, like `db.Config`. Put non-secret defaults in `application.yml` and every key in `.env.example` using the `.` → `_` upper-case form (`auth.sessionTtl` → `AUTH_SESSIONTTL`). Keep secrets (seed password, client secrets, SMTP password, keys) out of `application.yml` and tag them `mask:"true"`: the loader dumps the whole config at debug, the default level. Match the loader's field binding (see the `adder` skill); never read `os.Getenv` directly.
6. **Use the standard library and existing deps first.** `golang.org/x/crypto/bcrypt` for passwords, `crypto/rand` for tokens. Add a dependency only when the tier genuinely needs it (OIDC, TOTP, WebAuthn), and run `go mod tidy`.
7. **Never log or return secrets.** No password, hash, session token, or reset token in logs or error responses. Store only SHA-256 hashes of session, reset, verification, and API tokens (32 bytes from `crypto/rand`); bcrypt is for passwords only, since token hashes must be indexable for lookup. Don't reveal whether an account exists: return the same error for an unknown user and a wrong password (compare against a dummy hash so timing matches), and the same response to every reset request.
8. **Reset atomically.** Validate/consume the reset token, update the password, and revoke sessions in one transaction. Lock or conditionally consume the token so concurrent requests cannot both succeed.
9. **Use `TIMESTAMPTZ` for every timestamp column.** The genesis `users` table already uses it (`created_at`, `updated_at`), and sqlc maps it to `pgtype.Timestamptz`. Do the same for session/reset expiry and lifecycle columns; never mix in plain `TIMESTAMP`, which drops the offset and breaks expiry checks on a non-UTC database.
10. **Expiry is not cleanup.** Wire bounded periodic deletion of expired sessions and expired/used reset records, with suitable indexes.
11. **Validate bcrypt's byte limit.** Reject passwords over 72 bytes with a client-validation error on signup/reset; character-count validators are insufficient.
12. **Send email through a `Mailer` interface** in `internal/auth` with an SMTP implementation. When SMTP is not configured, reset and verification requests return 503 and write no token rows; never log the link as stand-in delivery. Startup must not depend on the log level.
13. **SSO.** Use `golang.org/x/oauth2` + `github.com/coreos/go-oidc/v3`; check `state`, PKCE, and `nonce`, and verify the ID token. Key identities by `(provider, subject)` in their own table, never by email, and make `password_hash` nullable for SSO-only users. Link to an existing user only when both sides' emails are verified — otherwise an attacker who pre-registers the victim's email keeps access. Apple posts the callback cross-site (`form_post`), so its state cookie needs `SameSite=None; Secure`; its client secret is an ES256 JWT valid at most 6 months, so mint it at runtime from the `.p8` key; it sends the user's name only on first sign-in.
14. **Seed the first admin from config, in every tier.** Create it only when no `Admin` user exists and never overwrite one; if none exists and the seed password is empty, fail startup with a clear error. Signup always creates `User`; never accept `role` from the request.
15. **Strict extras.** Keep rate-limit and lockout state in Postgres (in-memory breaks with replicas), and set gin's trusted proxies to the real proxy chain — by default gin trusts all, so `ClientIP()` reads a spoofable `X-Forwarded-For`. Lockouts expire and back off; a permanent one lets anyone lock out the admin. Encrypt TOTP secrets at rest, reject reused codes, and hash recovery codes.
16. **MFA enrollment.** A session after the first factor is *pending* until a second factor succeeds, which deletes it and issues a new full session. Allow it by action, not by path prefix: `/auth/me`, `/auth/logout`, verifying a TOTP code, passkey, or recovery code, and enrolling the first factor — nothing else. Listing, renaming, removing, or adding factors and regenerating recovery codes need a full session; a pending session that can regenerate codes bypasses MFA. A pending session is only as strong as the password or SSO account, so it may enroll a factor only while the user has none — otherwise a password thief adds their own. Check this at `begin` and again inside the `finish` transaction, holding `SELECT … FOR UPDATE` on the user row, so two concurrent enrollments cannot both pass. Issue recovery codes at the first enrollment, whichever factor it is, and accept them from any enrolled user. Refuse removing the last factor.
    For a user who has lost every factor and recovery code, add an admin-only MFA reset. It wipes their TOTP, passkeys, and recovery codes and revokes their sessions in one transaction, and is audited; they enroll again at next sign-in. An admin cannot reset their own MFA. A locked-out sole admin is recovered by an operator in the database.
17. **Passkeys** (when chosen):
    - `github.com/go-webauthn/webauthn`. Attestation `none`, no AAGUID allowlist, so synced passkeys (iCloud, Google, 1Password) work. User verification `required` on registration and every assertion, so a stolen unlocked device is not enough; this rejects U2F-only security keys, which is accepted. Resident key `discouraged`: assertions list the user's credentials, so discoverable ones only fill the limited slots on hardware keys.
    - Config `auth.webauthn.rpId` (exact browser host) and `auth.webauthn.origins` (full origins), defaulting to `localhost` / `http://localhost:5173`, set per env in deploy config. A mismatch fails every ceremony with a `SecurityError`, and passkeys never cross envs.
    - Migration: `users.webauthn_handle BYTEA UNIQUE`, 32 random bytes made on first registration — never the email or row id. `user_passkey(user_id, credential_id BYTEA UNIQUE, credential JSONB, name, synced, created_at, last_used_at)`; `credential` is the library's `Credential` JSON, stored plain — it is a public key. `webauthn_challenge(token_hash BYTEA PK, user_id, session JSONB, expires_at TIMESTAMPTZ)`.
    - Ceremony state: the library's `SessionData` goes in `webauthn_challenge`, never in memory (replicas). Single-use (delete on take), 5-minute TTL, bound to the user id, swept by the cleanup job. Its random token travels in an HttpOnly, Secure, `SameSite=Strict` cookie scoped to the auth path; store only its SHA-256.
    - Registration excludes the user's existing credentials. Assertion lists only the session user's credentials, shares the TOTP throttle and lockout, and saves the updated credential (sign count, flags) and `last_used_at`.
    - Sign count: reject when `credential.Authenticator.CloneWarning` is set after validation — the library flags it but does not fail the assertion. It stays clear when both counts are 0, which synced passkeys always send.
    - Audit `auth.passkey_register` / `auth.passkey_remove`, and `"method":"passkey"` on login, enroll, and failure events.
    - Web: the browser's own `PublicKeyCredential.parseCreationOptionsFromJSON` / `parseRequestOptionsFromJSON` and `credential.toJSON()` — no npm dependency. Feature-detect them, and map `NotAllowedError` (cancelled) and `InvalidStateError` (already registered) to readable messages. The MFA page offers "Use passkey" first when enrolled, with the code form as fallback; a Passkeys page lists, renames, and removes them.

## Verification procedure

1. `make build` passes in the target server.
2. `make generate` is clean and the generated code is in the tree.
3. A migration file exists — up *and* down — for every new table or column the code queries.
4. Every new config key appears in `.env.example`, non-secret ones also in `application.yml`, and secret fields carry `mask:"true"`.
5. For cookie sessions: the cookie is set with `HttpOnly`, `Secure`, and `SameSite`.
6. Session tokens are stored hashed, with an expiry column, and logout deletes the row.
7. Run it — a passing build or `[no test files]` is not verification. Start the project's docker-compose Postgres, run the server with `LOG_LEVEL=info`, and curl the flows: login, logout, a protected route, a role check. Migrate both a fresh database and one with existing users (they can still log in), and recheck expiry after `ALTER DATABASE ... SET timezone = 'Asia/Singapore'`.
8. Passkeys: curl cannot run a ceremony, so unit test it with a software authenticator — an ECDSA P-256 key signing CBOR authenticator data — covering register, assert, a backward sign count, another user's credential (`401`), the enrollment and last-factor guards, a pending session refused on remove and recovery-code regeneration, and two concurrent enrollment finishes from one pending session (exactly one succeeds). Curl the admin MFA reset: the user's sessions are gone and their next sign-in asks them to enroll. Then check it in a real browser on `localhost` (Chrome DevTools → WebAuthn virtual authenticator works without hardware).
9. `standard`+: of two concurrent resets with one token, exactly one succeeds; a failed reset rolls back; 40 × `é` (80 bytes) is rejected with a 4xx; expired rows get cleaned up.

## Common mistakes to watch for

- **Re-asking what arguments already answered.** `/genesis-auth basic` should ask nothing except the final confirmation.
- **Migrations without a `Down` block**, or editing an already-applied migration instead of adding a new one.
- **Hand-editing `internal/db/sqlc/`.** It is generated; change the `.sql` query and regenerate.
- **Letting a pending session add a second passkey or TOTP** next to an existing factor. Enrollment from a pending session is for the first factor only.
