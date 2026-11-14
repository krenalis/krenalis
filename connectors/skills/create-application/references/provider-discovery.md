# Provider discovery

Research only the capabilities selected for this task. The goal is a defensible endpoint and data model, not a complete inventory of the provider API.

## Use authoritative provider evidence

Prefer, in order:

1. the provider's current official API reference or machine-readable specification;
2. official guides for authentication, pagination, limits, retries, idempotency, and error formats;
3. an official SDK only to resolve behavior the API documentation leaves ambiguous.

Record the exact document or endpoint used for material decisions. Reconcile contradictions rather than averaging them. Ask the user only when the unresolved choice materially changes capabilities, data semantics, security, or operational cost; otherwise state the conservative assumption.

Do not require every evidence-source category, exhaustive endpoint tables, or negative proof that no other endpoint exists.

## Evaluate the selected capability

For user import, establish:

- stable provider identifier and update timestamp semantics;
- pagination/cursor behavior and ordering guarantees;
- server-side incremental filtering, precision, inclusivity, and timezone;
- field discovery, types, nullability, and deleted or suppressed records.

For user upsert, establish:

- create and update identity rules and writable fields;
- single, batch, bulk, or asynchronous endpoint behavior;
- idempotency support and per-record versus request-level errors;
- maximum item and byte limits.

For event delivery, establish:

- accepted event types, identity requirements, timestamps, and property format;
- server/client suitability and the correct `SendingMode`;
- single, batch, bulk, or asynchronous outcomes;
- per-event errors, payload limits, and idempotency behavior.

For authentication and transport, establish credential shape, role-specific OAuth scopes when applicable, base URLs or regions, rate-limit bucket boundaries, retry headers, and version headers.

## Select an endpoint that fits one method call

Prefer an endpoint that reports a deterministic accepted or rejected outcome for the consumed records or events within the connector call. A synchronous batch with per-item results is usually a better fit than an asynchronous job.

Use an asynchronous job only when the provider supports bounded, context-aware polling to a meaningful terminal result and the latency and quota cost are acceptable. If the provider only acknowledges a job or reports final failure through a callback, document the delivery ambiguity instead of treating acceptance as confirmed delivery.

One provider request per framework call is a useful simplification, not a universal rule. Permit bounded extra calls for documented identity resolution, pagination, validation, or terminal polling when they preserve correct consumption and error mapping.

Compare only serious candidates. A compact decision note should identify the chosen endpoint, rejected material alternative, and the decisive constraint.
