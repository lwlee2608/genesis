# SSO (Google, Apple)

Rules for Google and Apple sign-in. The rules in SKILL.md still apply.

## Rules

1. **Use `golang.org/x/oauth2` and `github.com/coreos/go-oidc/v3`.** Check `state`, PKCE, and `nonce`, and verify the ID token.
2. **Key identities by `(provider, subject)` in their own table, never by email.** Make `password_hash` nullable for SSO-only users.
3. **Link to an existing user only when both sides' emails are verified.** Otherwise an attacker who pre-registers the victim's email keeps access.

## Apple

4. **Set the state cookie to `SameSite=None; Secure`.** Apple posts the callback cross-site (`form_post`), so a `Lax` cookie never arrives.
5. **Mint the client secret at runtime from the `.p8` key.** It is an ES256 JWT valid at most 6 months.
6. **Save the user's name on first sign-in.** Apple never sends it again.
