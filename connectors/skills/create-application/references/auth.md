# Authentication

Implement the authentication mechanism required by the selected provider capabilities. Verify whether credentials, scopes, hosts, and account identity differ by role or region.

## Connector-managed credentials

For API keys, bearer tokens, basic authentication, or provider-specific headers:

- model the credential in the settings UI and persist it through `ApplicationEnv.Settings`;
- validate only documented shape and size constraints;
- add it to requests immediately before `ApplicationEnv.HTTPClient.Do`;
- never put it in a URL, error, log, fixture, or returned preview;
- redact the complete secret-bearing header or query component in `PreviewSendEvents`.

Use an input of type `password` for secrets. Do not return an existing secret to the UI in plaintext when the settings workflow can preserve it safely without doing so. Do not invent a blank-value or sentinel-value preservation convention: first verify how the current create/update UI passes existing settings, deliberate clearing, and role state, then test that exact lifecycle.

## Framework OAuth support

Set `ApplicationSpec.OAuth.AuthURL` and `ApplicationSpec.OAuth.TokenURL`, plus role-specific `SourceScopes` and `DestinationScopes`. Set `EndpointGroup.RequireOAuth` only for patterns that need the access token. Registry validation requires at least one OAuth endpoint group when `AuthURL` is set and forbids `RequireOAuth` without OAuth support.

Implement:

```go
func (c *Connector) OAuthAccount(ctx context.Context) (string, error)
```

Return a stable provider account or tenant identifier after authorization. Use the framework HTTP client for the identifying request and propagate cancellation.

The current runtime implements an authorization-code exchange and refresh-token flow with client ID, client secret, redirect URI, scopes, and bearer-token injection. It exposes no connector hook for PKCE or a provider-specific token exchange. Treat providers that require those features, nonstandard token placement, or interactive steps as unsupported until the framework is extended; do not emulate a second OAuth lifecycle inside the connector.

Use `ApplicationEnv.HTTPClient.ClientSecret` or `AccessToken` only when a provider-specific, non-delivery operation genuinely needs the value. Ordinary OAuth API requests receive `Authorization: Bearer ...` automatically when their endpoint group has `RequireOAuth`.

## Verify auth behavior

Test missing and malformed settings, correct header placement, role-specific scopes or hosts, preview redaction, OAuth endpoint-group matching, and sanitized provider errors. Never require a real secret in an ordinary unit test.
