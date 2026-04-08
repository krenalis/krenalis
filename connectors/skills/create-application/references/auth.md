# Authentication (OAuth and non-OAuth)

Choose one primary authentication mechanism based on the vendor API and desired onboarding UX.

## OAuth 2.0

> **OAuth is experimental.** OAuth credentials (client ID, client secret) are currently passed to Krenalis via environment variables and are therefore stored unencrypted. Prefer non-OAuth authentication (API keys/tokens) when the provider supports it.

If the application requires OAuth 2.0:

- Populate `connectors.OAuth` with:
  - `AuthURL`, `TokenURL`
  - scopes split by role (`SourceScopes` vs `DestinationScopes`)
  - redirect URL restrictions if needed (`Disallow127_0_0_1`, `DisallowLocalhost`)
- `AuthURL` should be the provider's base authorization endpoint. Krenalis automatically appends `response_type=code`, `client_id`, `redirect_uri`, `state`, and (if present) `scope`.
- Use Krenalis's built-in OAuth only if the application supports the **authorization code** flow with **refresh tokens**.
  - If the API supports only client credentials or otherwise cannot issue refresh tokens, do not use `ApplicationSpec.OAuth`; fall back to a non-OAuth auth method (API key / token) or ask the user how they want to authenticate.
- If the provider requires **PKCE** for the authorization code flow, verify that Krenalis supports it for application connectors. If not supported, do not use `ApplicationSpec.OAuth`; fall back to a non-OAuth method or ask the user for the intended auth approach.
- Ensure `EndpointGroups` include patterns for:
  - the OAuth token endpoint (`TokenURL`) (token exchange happens before you have an access token)
  - any OAuth metadata endpoints your flow needs
  - application API endpoints
  - Choose `RequireOAuth` for each group according to whether calls should include access tokens; token/metadata endpoints typically do not need them.

Implement:

```go
func (c *MyConnector) OAuthAccount(ctx context.Context) (string, error)
```

This must return a stable non-empty UTF-8 account identifier (e.g. tenant ID, portal ID, account ID). Choose the identifier by consulting the application's docs and/or its "current account" endpoint. Ensure the endpoint you call is covered by your endpoint groups and required OAuth scopes.

Then, use `env.HTTPClient.Do(req)` for API calls; the HTTP client handles adding the Authorization header and refreshing tokens.

## Non-OAuth authentication (API keys / tokens / Basic)

If the connector does **not** use `ApplicationSpec.OAuth`, it must still authenticate requests. Use these rules:

- Store credentials in connector settings (JSON) and expose them via `ServeUI` if configuration is needed.
- Validate credential shape on `"save"` and make security checks complete (not partial). Use `connectors.NewInvalidSettingsError(...)` for user-facing errors.
- Never include PII in connector-authored validation errors. Echo a setting value only if both conditions hold: the context guarantees that the value cannot contain PII, even by mistake; and without that value the user cannot determine what the message refers to or how to fix it.
- In connector-authored validation errors, wrap setting/property names and any echoed setting values in `«...»` (for example `«api_key»`, `«base_url»`).
- `connectors.QuoteErrorTerm(...)` is the default helper when the quoted term may contain `»`.
- Use `connectors.QuoteErrorTerm(...)` only when the quoted term may contain `»`. If the quoted term cannot contain `»`, write it directly as `«...»` even when the value is computed at runtime.
- Do not add explicit UTF-8 checks for setting strings: Krenalis already guarantees strings are valid UTF-8.
- For security-sensitive strings (API keys, tokens, secrets), validate both lower and upper bounds (min and max length), not only minimum length.
- If the provider documents allowed characters/prefixes/formats, validate them; if not documented, keep validation conservative but still enforce an explicit max length to avoid unbounded inputs.
- Validate base URLs strictly (scheme/host/path constraints according to provider requirements).
- Never echo secrets back in UI responses, preview requests, or error strings. If you need to build a preview request, redact secrets in headers and query parameters by replacing them with `"[REDACTED]"`.
- Prefer sending secrets in headers (e.g. `Authorization: Bearer <token>`, `X-API-Key: <key>`) rather than in query strings unless the API requires query auth.
- Keep the auth mechanism explicit in the design summary (header name, scheme, and whether per-tenant base URLs exist).
