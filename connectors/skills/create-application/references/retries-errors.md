# Idempotency, retries, and error handling

## Idempotency and retries

Krenalis's connector HTTP client retries **only idempotent** requests, and only if it can recreate the body (via `req.GetBody`).

Mark idempotency by:

- using a safe HTTP method (GET/HEAD/OPTIONS/TRACE), OR
- setting `Idempotency-Key` / `X-Idempotency-Key`.

To be retriable, a request body must be retriable too: either the request has no body, or `req.GetBody` is set. `BodyBuffer.NewRequest(...)` sets `GetBody` for you.

If the API does not require sending the header, but you still want retries, set it to nil:

```go
req.Header["Idempotency-Key"] = nil
```

## Error handling: what to return, and when

General:

- If `env.HTTPClient.Do(req)` returns an error, return it as-is (do not wrap).
- For non-2xx responses:
  - decode API error payload when possible
  - return a meaningful Go error
- Never include PII in connector-authored error messages.
- Include a value in a connector-authored error message only if both conditions hold:
  - the context guarantees that the value cannot contain PII, even by mistake
  - without that value the user cannot determine what the message refers to or how to fix it
- If you assign a message to `record.Err`, pass it to `events.Discard(err)`, or store it in `connectors.EventsError`, keep it fixed: do not interpolate values that vary with the specific bad input. Prefer messages like `Date is not a valid date`, not messages like `Date '2026/13/44' is not a valid date`.
- For connector-authored error messages, wrap property names, property paths, and any setting names or setting values shown verbatim in `«...»` (for example `«metadata»`, `«address.first_name»`, `«base_url»`).
- If you can avoid exposing raw upstream API error text, prefer a connector-authored message that gives enough context without risking PII leakage.
- If you expose raw upstream API error text and you are not certain it cannot contain PII, add a local comment at the use site in this exact form: `// TODO: please review whether this upstream API error may contain PII`.
- Do not rewrite upstream API error text just to apply this quoting style; preserve vendor messages as they are if you choose to expose them.

If you need to quote arbitrary text in connector-authored error messages, use `connectors.QuoteErrorTerm(...)`:

```go
err := fmt.Errorf("%s is invalid", connectors.QuoteErrorTerm("base_url"))
```

This is safe for property names, property paths, setting names, and echoed setting values.

Use the helper only when the quoted term may contain `»`. If the quoted term cannot contain `»`, write it directly as `«...»` even when the value is computed at runtime.

Per-item failures:

- Before sending a request:
  - use `records.Discard(err)` / `events.Discard(err)` for local validation failures (non-retryable per-item errors)
- After sending a request:
  - For Upsert: return `connectors.RecordsError{index: err}` only for per-record validation/acceptance failures you can map to specific indices.
  - For SendEvents: return `connectors.EventsError{index: err}` only for per-event validation/acceptance failures you can map to specific indices.
  - If the API rejects the whole batch with a single error (no per-item details), return a `connectors.RecordsError` / `connectors.EventsError` that maps **every consumed item index** to that error (since the whole batch failed).
  - If the API's response is ambiguous (you cannot know which items were accepted/rejected), return a generic error (do not guess per-item outcomes).

Important: `events.Discard(err)` and `connectors.EventsError` are for different phases and should not be mixed.

- Use `events.Discard(err)` only for **pre-send** local validation failures (e.g., missing required fields for the chosen event type, invalid value types for the schema).
- Use `connectors.EventsError` only for **post-send** API rejections you can map back to specific indices (e.g., API returns per-item errors).

## Preview behavior (events)

- `PreviewSendEvents` returns `(*http.Request, error)`.
- If you discard all events during preview, return `(nil, nil)` (this is a supported behavior).
- Ensure secrets are redacted.
- If your preview payload includes `event.DestinationPipeline` or uses it to build an identifier, replace it with `"[PIPELINE]"` in the preview request.
