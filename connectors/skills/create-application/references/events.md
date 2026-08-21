# Event destinations

Declare `TargetEvent`, implement `connectors.EventSender`, and choose a non-`None` `SendingMode` supported by the provider endpoint. Do not add event support solely because the provider has an API with an event-like name.

## Define event types

`EventTypes` returns the event types available to the connection. Keep each ID stable and unique; the public API limits it to 100 runes. Use `DefaultFilter` only when it accurately selects the corresponding received event.

`EventTypeSchema` describes only the mapped values needed in addition to `Event.Received`. Return:

- the exact object schema for a known type with extra values;
- invalid `types.Type{}` for a known type needing none;
- `connectors.ErrEventTypeNotExist` for an unknown ID.

Use `Event.Received` for standard event fields and `Event.Type.Values` for schema-governed mapped values. Do not mutate `Type.Values` before a possible postpone; clone maps when payload normalization would otherwise mutate caller state.

## Preview without delivery

`PreviewSendEvents` must build the HTTP request that would be sent without invoking the delivery endpoint. Return `nil, nil` if every event is discarded. `SendEvents` returns nil in that case. A distinct, documented, side-effect-free validation endpoint is connector-specific and must not replace returning the redacted delivery request.

Redact authentication as `[REDACTED]`. If the destination pipeline identifier contributes to an event ID or payload value, replace that contribution with `[PIPELINE]`. Redact secrets in headers, URLs, and bodies. Preserve all other request semantics so the preview is useful.

Preview and send may share a request builder, but preview must not consume mutable shared state or trigger a background delivery. The public contract requires both methods to be safe for concurrent calls on one connector instance.

## Consume and report events

Choose one iterator:

- `First` for an endpoint that accepts one event;
- `SameUser` when the provider request must contain events for one anonymous identity;
- `All` for a mixed-user batch endpoint.

Each yielded event is consumed unless postponed. The production iterator groups `SameUser` by anonymous ID. `Discard(err)` reports a local event failure without sending it. The first event in a sequence cannot be postponed; preflight it with `Peek` or discard a deterministically oversized/invalid first event instead of panicking.

Return `connectors.EventsError` with zero-based consumed-event indices for provider validation failures that can be attributed per event. Indices absent from the map are successes. A different non-nil error applies to all events consumed by the call and represents request-level failure.

Correlate batch results by preserved order or a documented stable key. Do not report asynchronous job acceptance as confirmed delivery when later failures cannot be mapped. Apply the selected iteration and batching guidance for item and byte limits.

## Test delivery semantics

Cover every event type and schema, unknown types, default mappings, identity variants, preview/send request equivalence, complete redaction, concurrency, item and byte boundaries, first-event overflow, postpone/discard, partial `EventsError`, request-level failures, and every documented success status.
