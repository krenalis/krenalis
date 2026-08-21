# HTTP behavior, retries, and errors

Use `ApplicationEnv.HTTPClient` for every provider request. It supplies endpoint-group rate limiting, OAuth injection, request capture, replay, retry behavior, and response-body draining.

## Build and close requests

Use `HTTPClient.GetBodyBuffer` for a request body and `BodyBuffer.NewRequest` to finalize it. For a bodyless request, use `http.NewRequestWithContext`. Always propagate the caller's context.

Set provider-required content type, version, conditional, and idempotency headers explicitly. `BodyBuffer.NewRequest` defaults to JSON headers and supplies `GetBody`; override `Content-Type` when the encoded format is not JSON.

Call `HTTPClient.Do`, check every expected success status, decode bounded responses, and `defer res.Body.Close()`. The framework's close wrapper drains up to its internal limit, so no separate manual drain is required. The client does not follow redirects.

## Model rate-limit buckets with endpoint groups

`EndpointGroup.Patterns` use `http.ServeMux` syntax and are matched against the actual method, host, and path. A nil `Patterns` value is the catch-all `/`. Verify that every request matches exactly one intended group. Configure patterns only for endpoints the connector can actually call; do not retain rejected discovery candidates in the spec merely to document their quotas.

If `ApplicationSpec.EndpointGroups` itself is nil, the runtime supplies a catch-all limiter of one request per second with burst one. Do not set a non-nil empty slice: it matches no requests. Every explicit group needs positive `RequestsPerSecond` and `Burst`; `MaxConcurrentRequests` may be zero for unlimited concurrency.

One endpoint group represents one shared limiter and retry policy. Combine patterns only when provider evidence says they share a quota bucket. Keep separate groups—even with identical numeric configuration—when the provider gives independent buckets. Never merge groups merely because their values happen to match.

Endpoint-group limiters are created per HTTP client/connection. They do not coordinate an account-wide provider quota across separate Krenalis connections that reuse the same credentials. Surface such a quota as a framework limitation instead of claiming the configured limits enforce it globally.

## Configure retries deliberately

With a nil `RetryPolicy`, the runtime uses `Retry-After` with exponential fallback for 429 and exponential backoff for 500, 502, 503, and 504. A non-nil policy replaces those defaults; an unmatched status is permanent except for the runtime's OAuth-aware 401 handling.

The client retries network and policy failures only when the body is replayable and the request is idempotent. Safe methods are idempotent; mutating methods require the presence of `Idempotency-Key` or `X-Idempotency-Key`. When the provider implements an idempotency header, send a stable key for the logical operation. When the operation is independently proven safe to retry but the marker must not be transmitted, `req.Header["Idempotency-Key"] = nil` marks it only for the framework. Never use either form merely to obtain retries without provider evidence.

Use `connectors.RetryAfterStrategy`, `HeaderStrategy`, `ExponentialStrategy`, or `ConstantStrategy` to encode documented behavior. Do not add a second generic retry loop inside the connector. Bounded asynchronous-job polling is a separate provider workflow and must still honor context and endpoint groups.

## Map responses without leaking data

Distinguish:

- transport or request-wide failures, returned as a normal error;
- per-record validation failures, returned as `connectors.RecordsError`;
- per-event validation failures, returned as `connectors.EventsError`;
- locally invalid items, handled with iterator `Discard` when possible.

As an intentional privacy policy, connector-authored errors must not include credentials, request bodies, user attributes, event properties, email addresses, IP addresses, or other payload data. Quote safe dynamic field/setting names with `connectors.QuoteErrorTerm`. Map status, documented error codes, and reviewed safe messages; do not blindly return a raw provider response body.

When one provider validation response definitively rejects every item in a consumed batch, a `RecordsError` or `EventsError` may map every consumed index to the safe error. If acceptance is ambiguous, return a request-level error instead of guessing per-item outcomes.

Test endpoint pattern selection, independent/shared buckets when relevant, default versus explicit retry policy, idempotent replay, cancellation, all success statuses, malformed and oversized error bodies, response closure, and error privacy.
