# MFA (strict tier)

Second-factor rules for the `strict` tier. The rules in SKILL.md still apply.

## Pending sessions

1. **Treat a session after the first factor as pending.** A successful second factor deletes it and issues a new full session.
2. **Allow pending sessions by action, not by path prefix.** Allow only `/auth/me`, `/auth/logout`, verifying a TOTP code, passkey, or recovery code, and enrolling the first factor. Everything else needs a full session: listing, renaming, removing, or adding factors, and regenerating recovery codes. A pending session that can regenerate codes bypasses MFA.
3. **Enroll from a pending session only while the user has no factor.** A pending session is only as strong as the password or SSO account, so otherwise a password thief adds their own factor. Check at `begin` and again in the `finish` transaction, holding `SELECT … FOR UPDATE` on the user row, so two concurrent enrollments cannot both pass.

## Factors and recovery

4. **Issue recovery codes at the first enrollment**, whichever factor it is. Hash them, and accept them from any enrolled user.
5. **Refuse removing the last factor.**
6. **Encrypt TOTP secrets at rest and reject reused codes.**
7. **Add an admin-only MFA reset** for a user who lost every factor and recovery code. In one audited transaction, wipe their TOTP, passkeys, and recovery codes and revoke their sessions; they enroll again at next sign-in. An admin cannot reset their own MFA. An operator recovers a locked-out sole admin in the database.

## Passkeys (when chosen)

8. **Use `github.com/go-webauthn/webauthn` with attestation `none` and no AAGUID allowlist**, so synced passkeys (iCloud, Google, 1Password) work.
9. **Require user verification** on registration and every assertion, so a stolen unlocked device is not enough. This rejects U2F-only keys; accept that. Set resident key `discouraged`: assertions list the user's credentials, so discoverable ones only fill the limited slots on hardware keys.
10. **Configure `auth.webauthn.rpId` and `auth.webauthn.origins` per environment.** The rpId is the exact browser host; origins are full origins. Default to `localhost` / `http://localhost:5173`. A mismatch fails every ceremony with a `SecurityError`, and passkeys never cross environments.
11. **Give each user a random WebAuthn handle**: 32 bytes made on first registration, never the email or row id.
12. **Store each credential as the library's `Credential` JSON**, unique by credential id. Store it plain; it is a public key.
13. **Keep ceremony state in Postgres, never in memory** (replicas). The library's `SessionData` is single-use, expires in 5 minutes, is bound to the user, and is swept by the cleanup job. Its random token travels in an HttpOnly, Secure, `SameSite=Strict` cookie scoped to the auth path; store only its SHA-256.
14. **Scope ceremonies to the session user.** Registration excludes their existing credentials. Assertion lists only their credentials, shares the TOTP throttle and lockout, and saves the updated credential and last-used time.
15. **Reject when `credential.Authenticator.CloneWarning` is set.** The library flags a backward sign count but does not fail the assertion. It stays clear when both counts are 0, which synced passkeys always send.
16. **Audit** `auth.passkey_register` / `auth.passkey_remove`, and `"method":"passkey"` on login, enroll, and failure events.
17. **Use the browser's WebAuthn JSON helpers, no npm dependency**: `PublicKeyCredential.parseCreationOptionsFromJSON` / `parseRequestOptionsFromJSON` and `credential.toJSON()`. Feature-detect them. Map `NotAllowedError` (cancelled) and `InvalidStateError` (already registered) to readable messages. The MFA page offers "Use passkey" first when enrolled, with the code form as fallback; a Passkeys page lists, renames, and removes them.

## Verification procedure

1. Curl the pending-session limits: a pending session is refused on factor removal and recovery-code regeneration.
2. Curl the admin MFA reset: the user's sessions are gone, and their next sign-in asks them to enroll.
3. Passkeys: curl cannot run a ceremony, so unit test with a software authenticator — an ECDSA P-256 key signing CBOR authenticator data. Cover register, assert, a backward sign count, another user's credential (`401`), the enrollment and last-factor guards, and two concurrent enrollment finishes from one pending session (exactly one succeeds).
4. Passkeys: check in a real browser on `localhost`. The Chrome DevTools WebAuthn virtual authenticator works without hardware.
