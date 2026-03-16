# HTTP (env.HTTPClient + BodyBuffer)

## Use env.HTTPClient

All HTTP calls must go through:

```go
res, err := c.env.HTTPClient.Do(req)
```

Do not use `http.DefaultClient` for connector calls.

Always close response bodies (`defer res.Body.Close()` or `_ = res.Body.Close()`), including on non-2xx responses, to avoid resource leaks.
Do not manually drain response bodies.

Meergo's connector HTTP client does **not** follow redirects automatically. If the API returns redirects that must be followed, handle them explicitly (read `Location`, rebuild the request, and call `Do` again). Follow at most 2 redirects (max two hops). If more redirects are returned, stop and return an error.

## Build request bodies with BodyBuffer

Use:

```go
bb := c.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding) // or connectors.Gzip
defer bb.Close()

// Write JSON manually (WriteString/WriteByte) and/or use Encode/EncodeKeyValue.
// Call bb.Flush() at safe points.

req, err := bb.NewRequest(ctx, method, url)
```

BodyBuffer advantages:

- pooled buffers (performance)
- optional gzip encoding
- `NewRequest` sets `GetBody`, enabling safe retries for idempotent requests
- prevents writes after `NewRequest` (correctness)

### Performance policy (hard requirement): no intermediate payload objects

When building JSON request bodies in `Upsert`, `SendEvents`, and `PreviewSendEvents`, stream JSON directly into the `BodyBuffer`.

- MUST stream JSON directly into `BodyBuffer` in `Upsert`, `SendEvents`, and `PreviewSendEvents`.
- MUST NOT allocate "payload objects" (for example `map[string]any`, `[]any`, or ad-hoc structs) just to `json.Marshal` / `bb.Encode` them.
- Prefer manual JSON framing (`{`, `}`, `[`, `]`) + `bb.EncodeKeyValue(...)` for object key/value pairs.
- Comma rule for object fields:
  - do not write commas manually between consecutive `bb.EncodeKeyValue(...)` calls
  - write commas manually only when you interrupt that sequence with other writes
- If you must allocate an intermediate structure (for example because the API requires a dynamic object you must compute and you cannot stream it safely), it must be a deliberate exception:
  - add a short code comment explaining why the allocation is necessary and why streaming is not feasible there
  - mention the same exception in the connector design summary under a "Skill deviations" note

Do not do this:

```go
payload := map[string]any{
	"event_name": eventName,
	"properties": eventProps,
}
_ = bb.Encode(payload)
```

Do this instead:

```go
bb.WriteByte('{')
_ = bb.EncodeKeyValue("event_name", eventName)
_ = bb.EncodeKeyValue("properties", eventProps)
bb.WriteByte('}')
```

## Flush/Truncate usage

Use `bb.Flush()` when you finish writing a chunk you won't later modify. For large batch bodies:

- track size checkpoints with `size := bb.Len()`
- if you exceed a limit, do:
  - `bb.Truncate(size)` (or a computed safe point)
  - `records.Postpone()` or `events.Postpone()`
  - `break`
