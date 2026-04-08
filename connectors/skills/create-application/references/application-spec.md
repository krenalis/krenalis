# ApplicationSpec: what to fill, and how

Krenalis validates `connectors.ApplicationSpec` vs implementation at registration time and can panic when they do not match. Treat spec fields as a contract.

## Literal style: omit zero values

When writing `connectors.ApplicationSpec` (and nested structs like `OAuth`, `EndpointGroup`, `RateLimit`, `RetryPolicy`, etc.), omit fields that would be set to their Go zero value.

- Do not write explicit zero-value assignments such as:
  - `Patterns: nil`
  - `RetryPolicy: connectors.RetryPolicy{}`
  - `TimeLayouts: connectors.TimeLayouts{}`
  - empty strings, `false`, `0`, empty/nil slices or maps
- Include a field only when you are setting a meaningful non-zero value.

## Capability contract (spec vs interfaces)

- `Code` must be non-empty and contain only characters in `[a-z0-9-]`.
- If you declare `AsSource.Targets` includes `TargetUser`, your connector type must implement `connectors.RecordFetcher` (`RecordSchema` + `Records`).
- If you declare `AsDestination.Targets` includes `TargetUser`, your connector type must implement `connectors.RecordUpserter` (`RecordSchema` + `Records` + `Upsert`).
- If you declare `AsDestination.Targets` includes `TargetEvent`, your connector type must implement `connectors.EventSender` and you must set `SendingMode != None`.
- If `HasSettings` is true in `AsSource` or `AsDestination`, your type must implement `ServeUI`.
- If you implement `ServeUI` but `HasSettings` is false for both roles, registration panics.

OAuth consistency rules (note: OAuth is experimental — see `references/auth.md`):

- If you set `ApplicationSpec.OAuth.AuthURL != ""`, your type must implement `OAuthAccount(ctx) (string, error)`.
- If OAuth is supported, at least one `EndpointGroup` must have `RequireOAuth: true` (otherwise registration panics).
- You cannot set `RequireOAuth: true` in any endpoint group if OAuth is not supported.

## Minimal Registration skeleton

```go
func init() {
    connectors.RegisterApplication(connectors.ApplicationSpec{
        Code:       "<code>", // non-empty; only [a-z0-9-]
        Label:      "<Human Name>",
        Categories: connectors.CategorySaaS,

        // Include only the roles/targets your connector actually implements.
        //
        // AsSource: &connectors.AsApplicationSource{
        //     Targets:     connectors.TargetUser,
        //     HasSettings: true,
        //     Documentation: connectors.RoleDocumentation{
        //         Summary:  "Import <X> as users from <App>",
        //         Overview: "<longer markdown-like description>",
        //     },
        // },
        //
        // AsDestination: &connectors.AsApplicationDestination{
        //     Targets:     connectors.TargetUser | connectors.TargetEvent,
        //     HasSettings: true,
        //     SendingMode: connectors.Server, // Client / Server / ClientAndServer; must not be None if TargetEvent is set
        //     Documentation: connectors.RoleDocumentation{
        //         Summary:  "Export users to <App> and/or send events",
        //         Overview: "<longer description>",
        //     },
        // },

        // Optional: override user terms, only if you support TargetUser.
        // UserID should refer to the application's User ID (i.e. what you put in Record.ID).
        // Terms: connectors.ApplicationTerms{
        //     User:   "User",   // or "Contact", "Profile", ...
        //     Users:  "Users",
        //     UserID: "User ID",
        // },

        // Optional: OAuth + endpoint groups (rate limiting, retry, and whether OAuth is required for those calls).
        // OAuth: connectors.OAuth{ ... },
        // EndpointGroups: []connectors.EndpointGroup{ ... },

        // Optional: custom time parsing formats for datetime/date/time values returned as strings/bytes
        // inside Record.Attributes / EventType.Values.
        //
        // Prefer setting TimeLayouts over writing per-field parsing code in the connector when the API
        // returns non-ISO date/time strings.
        //
        // If left empty, Krenalis parses time strings as ISO 8601.
        //
        // Note: this does not apply to Record.UpdatedAt (which is already a time.Time).
        // Omit TimeLayouts entirely when defaults are fine.
        // TimeLayouts: connectors.TimeLayouts{
        //     DateTime: "2006-01-02 15:04:05",
        // },
    }, New)
}
```

## EndpointGroups (rate limit + retry)

Endpoint groups let Krenalis enforce client-side rate limits and apply retry strategies per API endpoint group.

Key points:

- Patterns are `net/http` ServeMux-style and may include method and host prefixes.
  - Examples:
    - `"/api/profiles/"` (path prefix)
    - `"GET /3.0/lists/"` (method + path)
    - `"GET login.example.com/oauth2/metadata"` (method + host + path)
- If `EndpointGroups` is nil, Krenalis defaults to a single `"/"` group (matches all requests) with a conservative rate limit (1 rps, burst 1).
- Never set `EndpointGroups` to an empty slice (`[]connectors.EndpointGroup{}`): it registers no patterns and requests fail at runtime.
- `Patterns` must be either nil or contain at least one pattern (an explicit empty slice panics at registration time).
  - To match all, omit the `Patterns` field (equivalent to nil). Do not write `Patterns: nil`.
- If you have multiple endpoint groups:
  - ensure patterns cover **all hosts** your connector calls (API host, OAuth host, validation host, upload host, etc.), otherwise requests fail at runtime with "no endpoint group matches this request".
  - do not register the same pattern twice across groups (ServeMux handler registration may panic).
- Every endpoint group must set a valid `RateLimit`:
  - `RequestsPerSecond > 0`
  - `Burst > 0`
  - `MaxConcurrentRequests >= 0` (0 means unlimited)
- Retry only applies to idempotent requests with retriable bodies (see `references/retries-errors.md`).
- If you enable OAuth, at least one endpoint group must set `RequireOAuth: true`.

### Deriving endpoint groups from vendor rate limits

Create endpoint groups that mirror the vendor's distinct rate-limit buckets.

- If the vendor specifies different limits per endpoint family, create multiple groups.
- If the vendor specifies one global limit, one group is fine (but cite the source showing it is truly global).
- If rate limits are not documented, use a conservative default (e.g. `1 rps`, `burst 1`) and rely on 429 handling, but call it out as an assumption.
- Merge groups when their effective configuration is identical (`RequireOAuth`, `RateLimit`, and `RetryPolicy` are all the same): use one group with combined `Patterns`.
- Identical `RateLimit` alone does not imply merge if `RequireOAuth` or `RetryPolicy` differ.
- If docs are ambiguous about bucket sharing, fail closed: keep groups separate and use conservative limits.

### Rate-limit proof table (required when limits are documented)

Before coding, build a short table in your design summary with one row per endpoint family/group:

- group/pattern
- vendor bucket scope (global vs specific endpoint family)
- raw documented limits + window
- converted Krenalis values (`RequestsPerSecond`, `Burst`, `MaxConcurrentRequests`)
- `RequireOAuth` and a short `RetryPolicy` summary (so merge/split decisions are explicit)
- source links

This table is mandatory whenever you define non-default rate limits.

### Converting per-hour/per-minute limits to Krenalis RateLimit

Krenalis models rate limits as a token bucket with:

- `RequestsPerSecond` (float64): average rate
- `Burst` (int): maximum accumulated tokens (short burst capacity)

Rules:

- Convert all limits to per-second averages:
  - `X per hour` -> `X / 3600` rps
  - `X per minute` -> `X / 60` rps
- If the vendor provides multiple limits for the same bucket (e.g. per-second AND per-hour), set `RequestsPerSecond` to the minimum.
- Choose `Burst` conservatively:
  - strict low average limits (e.g. 100/hour): keep `Burst` small (often 1)
  - explicit per-second limits (e.g. 10/sec): set `Burst` around that limit

Always include the derived values and original limits in your "Connector Design" summary.

### Traceability in code comments (recommended)

For each non-default endpoint group, add a short comment near `EndpointGroups` showing:

- the vendor limit you mapped (for example `120 req/min`)
- the converted Krenalis values
- a doc link

This keeps future rewrites from silently changing the mapping logic.

### Recommended endpoint group structure for OAuth connectors

> **Reminder:** OAuth is experimental in Krenalis. See `references/auth.md` for details.

If you enable `ApplicationSpec.OAuth`, it is often useful to separate endpoint groups by whether requests should carry OAuth access tokens:

- OAuth token/metadata endpoints: typically `RequireOAuth: false` (calls happen before you have an access token)
- Application API endpoints: typically `RequireOAuth: true`

Keep patterns specific by host/path so both groups can coexist without duplicating `"/"`:

- `POST login.example.com/oauth/token`
- `GET login.example.com/oauth/metadata`
- `GET api.example.com/v1/`
- `POST api.example.com/v1/`

When the application's host varies by region/tenant (e.g. `api.eu.example.com` vs `api.us.example.com`), prefer method + path patterns without a host to keep matching stable:

- `POST /oauth/token`
- `GET /v1/`
- `POST /v1/`

Note: host-less patterns apply to **all** hosts. Use them only if the same rate limits, retry policy, and OAuth requirements apply across the involved hosts.

### Retry policy guidance

- If the vendor docs specify retry behavior (status codes, headers, backoff), encode it in `RetryPolicy`.
- If the vendor docs do **not** specify how to retry, prefer omitting `RetryPolicy` and rely on Krenalis's default behavior for idempotent requests.
- If the vendor uses a documented non-standard header for 429 resets (or similar), it is reasonable to add a small targeted policy for that status code (e.g. header-based strategy), and still omit other codes unless justified.
