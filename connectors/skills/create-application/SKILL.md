---
name: create-application
description: Create an application connector
license: MIT
---

# Prompt: Create a Meergo Application Connector (Go)

You are running in a local repo that contains the Meergo source code. Your task is to implement a **new Meergo Application connector** (a Go package) that integrates a SaaS product via HTTP APIs.

This prompt is a reusable base. When the user provides a concrete product/API later, you must adapt the connector accordingly.

This skill uses a `references/` directory for on-demand details. Keep your working context small: read the reference file(s) only for the capabilities you are actually implementing.

## Scope (hard constraint)

This prompt is **only** for building **Application connectors** (SaaS HTTP APIs) under `connectors/<code>/`.

- Do not implement other connector types (SDK, webhook, database, file, file-storage, etc.).
- Do not add new frameworks or large dependencies to the repo. Prefer stdlib + existing Meergo helpers.

## 1) Discovery (required): decide what to build

Sometimes the user will only provide the application name (e.g. "Build a connector for <App>"). In that case, you must first **study the application API** to decide:

- whether the connector can be a **Source** (fetch users/records), a **Destination** (upsert users/records), and/or a **Destination (events)** (send events)
- whether it must support **OAuth 2.0** or can use another auth method (API key, Basic, bearer token, etc.)
- which endpoints exist, their pagination and rate limits, and the supported data model

### Discovery workflow (hard requirement)

Accurate discovery is a hard gate.
Do not proceed to implementation if discovery is incomplete, weakly evidenced, or internally contradictory.

Discovery may and should use broader web search to find the official sources.
Implementation decisions must then be based only on official sources:

1. official machine-readable API specification (OpenAPI/Swagger first; equivalent official spec artifacts second)
2. official human documentation (guides/reference pages/changelogs)
3. official SDK implementation, if one exists

Do not treat blog posts, forum answers, or unofficial code snippets as implementation evidence.

Use the sources in this order and for these roles:

- API specification: primary endpoint inventory source
  - extract methods, paths, parameters, status codes, request/response schema shapes, auth schemes, pagination/filtering parameters, and event ingestion paths including batch/bulk/import variants
- Documentation: primary semantic/explanatory source
  - extract field semantics, identifier meaning, required scopes/permissions, rate limits and bucket semantics, region/base-URL rules, constraints, caveats, and workflow notes not obvious from the spec
- Official SDK: confirmation source
  - use it to confirm endpoint families, operation grouping/naming, request shape when spec/docs are incomplete, and extra signal for batch/bulk support

Do not read all sources exhaustively. Extract only the facts needed to decide capabilities and implementation shape. Prefer short inventories and proof tables over long narrative notes.

Required sequence:

1. Find the official API specification first.
2. If found, use it as the primary endpoint inventory source.
3. Cross-check it with the official documentation and resolve mismatches explicitly.
4. Inspect the official SDK, if one exists, as a confirmation pass.
5. Only if no usable specification exists, switch to documentation-first discovery and then check the official SDK, if one exists.

Do not skip step 1. Do not rely on documentation alone before checking whether a machine-readable specification exists.

General exclusion rule:

- Do not exclude a capability, endpoint, payload shape, auth mode, pagination mode, validation endpoint, rate-limit rule, batch mode, or any other API requirement until you have checked all available official sources in the required order.
- A feature may be treated as unsupported or absent only with explicit negative evidence across all available official sources that could reasonably mention it.
- If one source shows support and another only fails to mention it, treat the feature as potentially supported and resolve the mismatch explicitly instead of excluding it.

If no specification is found, record explicit negative evidence in the design summary showing what was searched and where.

Discovery must be performed in two separate phases.

#### Phase 1: structural inventory

Before drawing any conclusion, build a complete endpoint inventory for the relevant API reference area.

1. Identify the relevant reference section(s) for the capability family you are studying.
2. Enumerate all child pages or operations in that section.
3. Open each page or operation and extract:
   - page title
   - HTTP method
   - exact path
   - short capability hint
4. If the reference is split across multiple sections, merge them into one inventory.
5. Do not classify or exclude capabilities yet.

When a machine-readable specification exists:

- use it to enumerate operations comprehensively for the relevant tags/path prefixes
- then reconcile the resulting inventory with the human reference pages

When a machine-readable specification does not exist:

- enumerate the human reference section(s) directly from the docs navigation/index
- then confirm with the official SDK, if one exists

The required Phase 1 output is an endpoint inventory table with one row per documented endpoint:

| Page title | Method | Path | Capability hint |
|------------|--------|------|-----------------|

#### Phase 2: capability classification

Only after Phase 1 is complete, classify the inventory into capabilities.

For each relevant capability family, explicitly search the inventory and official sources for variants such as:

- create
- create batch
- bulk
- import
- export
- async job
- list
- search
- update
- delete
- webhook

Also explicitly check for path variants such as:

- `/batch`
- `/bulk`
- `/import`
- `/export`
- `/job`
- `/async`

Treat related endpoints as distinct capabilities when they have distinct semantics, for example:

- `/events`
- `/events/batch`
- `/events/export`
- `/events/import`

For each main capability you may later implement (`Records`, `Upsert`, `SendEvents`), do not stop at "does an endpoint exist?".
You must also identify the relevant candidate endpoints for that capability family and compare their execution model and outcome semantics.

At minimum, compare candidates along these axes:

- request model: synchronous, asynchronous with polling, or asynchronous with callback/webhook-only completion
- outcome visibility: immediate, pollable by a documented endpoint, callback-only, or ambiguous
- outcome granularity: whole-request only, per-item/per-event, or ambiguous
- contract fit: whether the endpoint can satisfy the Meergo method contract when the method returns

This is a hard requirement for capability families where APIs often expose both direct and bulk/job-based variants, especially:

- user create/update vs import/bulk/job endpoints
- event send vs batch/import/job endpoints

If multiple candidate endpoints exist for one capability family, explicitly compare them before choosing one.
Do not pick the first endpoint that "does the thing".

Before concluding that a capability is absent, perform a targeted sweep for each critical capability family:

- user listing / export
- user create
- user update
- user identifiers and lookup paths
- event ingestion
- batch event ingestion
- pagination and incremental sync
- rate limits / endpoint buckets
- validation / preview endpoints

For each capability family, record one of:

- `Verified`
- `Not found`
- `Ambiguous`

Do not stop after one failed search or one missing page.
For capability families that strongly affect implementation quality, keep searching until you either find support or can justify negative evidence from all available official sources.

The most important quality-sensitive capability families are:

- batch event ingestion
- batch user operations, if the API appears to support bulk user import/update
- incremental sync for records
- endpoint-specific rate limits

#### Phase 3: contract-fit evaluation

After classifying capabilities, evaluate whether each candidate endpoint is actually compatible with the Meergo interface you would use it for.

This phase is separate from endpoint discovery and capability existence.
An endpoint may exist and still be a poor or invalid implementation choice for a Meergo method.

For every chosen capability and every serious alternative considered, build a short candidate comparison table with this shape:

| Meergo method | Candidate endpoint | Delivery model | Outcome retrieval | Outcome granularity | Contract fit | Decision |
|---------------|--------------------|----------------|-------------------|---------------------|--------------|----------|

Where:

- `Delivery model` is one of:
  - `Sync`
  - `Async + poll`
  - `Async + callback only`
- `Outcome retrieval` states how the final result is known:
  - immediate response
  - documented polling endpoint
  - callback/webhook only
  - ambiguous from official sources
- `Outcome granularity` states whether the API provides:
  - whole-request outcome only
  - per-record/per-event outcome
  - ambiguous outcome mapping
- `Contract fit` is one of:
  - `Compatible`
  - `Conditionally compatible`
  - `Incompatible`

Use these rules:

- `Sync` endpoints are generally compatible candidates, subject to the usual validation/error semantics.
- `Async + poll` endpoints must be taken seriously and compared against sync endpoints when they exist.
  - They are `Conditionally compatible` only if the connector can recover a sufficiently deterministic final outcome by polling official endpoints within the same Meergo method call.
  - If the documented outcome remains too ambiguous, too delayed, or cannot be mapped back to consumed records/events with acceptable confidence, mark them `Incompatible` or keep them separate as a bulk/export job concept rather than the primary Meergo method implementation.
- `Async + callback only` endpoints are `Incompatible` for `Upsert` and `SendEvents`.
  - If the final outcome is available only by delivering it to an external webhook/callback endpoint, the connector cannot rely on that endpoint to satisfy the Meergo method contract when the method returns.
  - Do not treat such endpoints as valid primary implementations of `Upsert` or `SendEvents`.

Do not reject async endpoints merely because they are async.
Reject them only when their documented completion/result model is not usable within the Meergo contract, or when another candidate is clearly a better contract fit.

When comparing candidate endpoints, explicitly check and record:

- whether a documented job-status/job-result endpoint exists
- whether the connector can read the final result directly, without requiring third-party infrastructure
- whether the final result can be obtained within a bounded polling window suitable for the Meergo method call
- whether the result includes per-item/per-event failures, or only a global success/failure state
- whether global import/export options change semantics in ways that make the endpoint a poor fit for generic `Upsert` / `SendEvents`

If an endpoint is excluded because of contract mismatch, say so explicitly.
Do not write only "async, so excluded".
Write the concrete reason, for example:

- final outcome is available only via callback/webhook
- no documented endpoint exists to retrieve job outcome
- polling exists but does not expose usable per-item/per-event results
- job-level semantics do not match generic `Upsert` / `SendEvents`

### Default capability policy (when the user only says "build the connector")

If the user does not explicitly request roles/targets:

- Preferred default (when the API and docs are clear enough): implement the "common triad":
  - Source `TargetUser` (import users/records)
  - Destination `TargetUser` (export/upsert users/records)
  - Destination `TargetEvent` (send events) if the application clearly supports server-side event ingestion
- Conservative fallback (when docs are incomplete/ambiguous): implement **only** Destination `TargetUser` end-to-end, and omit Source/Events. Call out what is missing and why.

Never "half-implement" a declared capability: if you declare it in `ApplicationSpec`, implement the full interface contract.

### Discovery checklist and output

Discover, at minimum, these facts:

- sources:
  - official API docs URL(s) and base URL / environment / region rules, if applicable
- auth:
  - supported auth methods and required scopes/permissions
- user model:
  - what the app calls a "user"
  - which field is the application's **User ID** (`Record.ID`) and how it is obtained
  - stable identifiers and required fields for create/update
  - which alternate identifiers the API may accept, and whether they are truly unique/immutable
- user capabilities:
  - list users endpoint(s)
  - incremental sync support
  - pagination controls and limits
  - create/update endpoint(s) and required fields
- event capabilities:
  - event endpoint(s) and required properties
  - whether the API supports sending multiple events in one request
  - whether the API has a validation/dry-run endpoint suitable for preview
  - search event-related sources with keywords such as `batch`, `bulk`, `import`, `job`, `multiple events`, `partial success`, `eventIndex`
- operational constraints:
  - documented rate limits
  - rate-limit bucket mapping per endpoint family
  - raw limit expressions and windows
  - max batch size / max request body size
  - idempotency requirements

Before writing code, produce a short `Connector Design` summary. Keep it compact and structured. Prefer short tables or bullet inventories to long prose.

At minimum, include these blocks:

- `Sources checked`
- `Endpoint inventory`
- `Capability matrix`
- `Candidate endpoint comparison`
- `Capabilities chosen`
- `Rate-limit proof`
- `Events delivery proof` when events are implemented
- `Schemas plan`
- `Documentation plan`
- `Excluded items and why`
- `Open questions`

The summary must also include:

- chosen `ApplicationSpec` roles/targets (`AsSource` / `AsDestination`, `TargetUser`, `TargetEvent`)
- what `Record.ID` represents for this app and how updates will use it; what the matching property is, if any, and how it maps to the app User ID
- auth choice and rationale
- endpoint group plan (patterns + rate limits + retry policy)
- a concrete endpoint list (method + host + path) for each supported capability
- discovery evidence showing how specification, documentation, and official SDK were used; or explicit negative evidence for any missing source

The `Endpoint inventory` section must be a complete table with this shape:

| Page title | Method | Path | Capability hint |
|------------|--------|------|-----------------|

The `Capability matrix` section must be a complete table with this shape:

| Capability | Exists | Method | Path | Evidence page |
|------------|--------|--------|------|---------------|

`Exists` may be only:

- `Verified`
- `Not found`
- `Ambiguous`

The `Candidate endpoint comparison` section must compare the primary chosen endpoint(s) with any serious alternatives for the same Meergo capability, especially import/bulk/job variants for `Upsert` and `SendEvents`.

Use the same table shape defined in Phase 3.

If any capability or API behavior is excluded, include a short `Why excluded?` note.

Choose the evidence style that matches the reason:

- if the capability appears absent, include explicit negative evidence from all available official sources that could reasonably describe it
- if the capability exists but is excluded because the endpoint is a poor fit for the Meergo contract, include explicit positive evidence of the endpoint's documented behavior plus the concrete contract-mismatch reason

For contract-mismatch exclusions, ground the explanation in the actual Meergo method contract:

- `Upsert` must return the outcome of the records it consumed in that call; per-record failures should be representable as `connectors.RecordsError`
- `SendEvents` must return the outcome of the events it consumed in that call; per-event failures should be representable as `connectors.EventsError`
- a generic non-`RecordsError` / non-`EventsError` return is still part of the outcome returned by the method call itself, not a signal that completion may be delegated elsewhere

Therefore, an endpoint is a poor fit for the primary `Upsert` / `SendEvents` implementation when its documented completion model does not let the connector determine a usable outcome for the consumed records/events before the method returns.

If a capability is excluded not because it is absent, but because the endpoint is a poor fit for the Meergo contract, say that explicitly in `Why excluded?`.
Typical examples:

- endpoint exists, but final outcome is callback-only
- endpoint exists and is pollable, but does not provide usable per-item/per-event results
- endpoint exists, but its job/import semantics are too global to serve as the connector's primary `Upsert` / `SendEvents` implementation

If events are implemented with single-event requests, include a specific `Why not batch?` note with explicit negative evidence from all available official sources:

- no usable batch endpoint/payload in the API specification
- no batch support described in the documentation
- no batch-capable event sending path visible in the official SDK

Batch event sending is the most important instance of the general exclusion rule above. It may be excluded only after this three-source negative check. If any of the three sources shows batch support, treat batch as supported and design `SendEvents` around it.

Likewise, if user upsert/import APIs expose both direct record endpoints and bulk/job-based endpoints, compare them explicitly against the `Upsert` contract before choosing one.
Do not exclude a bulk/job-based user endpoint only because it is bulk or async.
Exclude it only with a concrete contract-fit reason, or choose it if it is the best supported fit.

Do not present a conclusion without both the complete endpoint inventory and the capability matrix.
Do not present a final implementation choice for `Upsert` or `SendEvents` without the candidate endpoint comparison table when official sources show multiple plausible endpoint families for that capability.

If multiple endpoint groups end up with identical `RequireOAuth`, `RateLimit`, and `RetryPolicy`, merge them into a single group with `Patterns` containing the union of patterns.

If there are open questions, ask the **minimum** set needed to proceed. Do not guess on auth, IDs, required fields, or rate limits.

### Autonomy vs confirmation (after discovery)

After producing the "Connector Design" summary:

- Proceed to implementation **without asking for confirmation** if:
  - discovery is accurate enough to justify the chosen implementation path
  - auth method is unambiguous and implementable with Meergo (OAuth auth-code+refresh, or a well-defined token/API key)
  - base URL (including region/tenant behavior) is unambiguous
  - the chosen capability set can be implemented end-to-end without inventing required fields/IDs
  - rate limits are documented, or you can safely use the conservative default (`1 rps, burst 1`) and rely on 429 handling
- Stop and ask the user the smallest set of questions if any of these are unclear:
  - whether discovery is complete enough to rule features in or out
  - which auth method to use (OAuth vs key/token), or required scopes/permissions
  - which stable identifier to use for upserts (email vs numeric ID vs external ID) and whether it is writable
  - which base URL/region/tenant host to call
  - required fields for create/update
  - whether an event endpoint is real ingestion vs "test/validate" only (if implementing events)
  - whether events batching is supported when docs/sources are incomplete or contradictory
  - whether endpoint families share the same rate-limit bucket when the docs are ambiguous

If something is optional (e.g. implementing Source vs Destination), prefer omitting it rather than guessing.

### Go version and standard library (required)

Meergo is compiled with the Go version specified by the `go` directive (and, if present, the `toolchain` directive) in `go.mod`.

Before implementing the connector:

- Read `go.mod` and target that Go version.
- Prefer the **standard library** and language features available in that version over custom helper functions.
- If you are about to write a small utility (e.g. "is status allowed", "contains", "min/max", map key checks), first check whether there is an idiomatic stdlib equivalent (commonly: `slices`, `maps`, `cmp`, `errors`, `net/http` helpers, etc.).
- If you are not sure an API exists in the target Go version, verify locally (e.g. `go doc slices`, `go test` / `go test ./...` to compile) instead of guessing.

### Minimum documentation needed (if the user must provide it)

If you cannot access the official docs, ask the user to provide at least:

- authentication docs (OAuth vs token; required headers; scopes/permissions; token refresh behavior)
- users/contacts API docs:
  - list endpoint + pagination
  - updated-since / incremental sync support (or confirmation it does not exist)
  - create/update endpoint(s) + required fields + identifiers
- events API docs (if events are expected) + any validation/dry-run endpoint
- rate limiting docs (limits and relevant headers like `Retry-After`)

### Decision heuristics (when multiple options exist)

- **OAuth vs API key / token**:
  - Prefer **OAuth** when the integration is installed by end users and credentials must be granted per-account/tenant/workspace.
  - Prefer **API keys / personal tokens** when the app explicitly recommends them for server-to-server integrations, or when OAuth requires flows Meergo does not support.
  - If the app supports both and the choice affects product UX, ask the user which onboarding they want (OAuth install vs paste-a-key), but propose a default based on the docs.
- **Rate limits**:
  - If the docs specify rate limits, encode them in `EndpointGroups`.
  - Do not keep duplicate endpoint groups with identical `RequireOAuth`, `RateLimit`, and `RetryPolicy`; merge them and combine patterns.
  - Identical rate limits alone do **not** require merge if `RequireOAuth` or `RetryPolicy` differ.
  - If rate limits are not documented, use a conservative default (e.g. `RequestsPerSecond: 1`, `Burst: 1`, `MaxConcurrentRequests: 0`) and call it out in the design summary as an assumption to confirm.
  - If rate limits vary by plan/tier and you cannot reliably detect the plan at runtime, default to the **minimum documented non-trial tier** for safety (or the minimum across all tiers if there is no clear baseline). Call this out explicitly in the design summary and connector docs.
  - If bucket mapping is unclear, fail closed: keep endpoint groups separate and use conservative per-group limits rather than merging groups by assumption.
- **Event `SendingMode`**:
  - Default to `connectors.Server` for SaaS HTTP APIs.
  - Use `connectors.Client` only when the application explicitly requires client-side sending semantics.
  - Use `connectors.ClientAndServer` only when both modes are meaningful and supported by the application.

### Docs incomplete or contradictory

- Prefer the actual code in this repo (types/contracts/validation panics) over external docs if they disagree.
- If docs contradict on auth, identifiers, or endpoint semantics: stop and ask the user (do not guess).
- If docs contradict on non-critical details (rate limits, optional fields): choose the safer conservative behavior, document the assumption, and proceed.

## 2) Where to implement it in the repo (always)

Implement the connector as a Go package under:

`<MEERGO_REPO_ROOT>/connectors/<code>/`

Follow existing patterns (see packages like `connectors/hubspot`, `connectors/klaviyo`, `connectors/mailchimp`, `connectors/mixpanel`, `connectors/posthog`, `connectors/googleanalytics`).

### Registration import (critical)

Your connector registers itself in `init()`, but that `init()` runs **only if the package is imported** somewhere in the Meergo binary.

Therefore, after creating `connectors/<code>/...`, add a blank import in the main build entrypoint `main.go`:

```go
_ "github.com/krenalis/krenalis/connectors/<code>"
```

(If your tests use a dedicated imports file, add the blank import there too so tests see the connector.)

## 3) Always-on implementation rules (apply to every Application connector)

### Spec vs implementation must match

Meergo validates `connectors.ApplicationSpec` vs implemented interfaces at registration time (panics on mismatch). Details, including OAuth consistency rules and endpoint group validity rules:

- [references/application-spec.md](references/application-spec.md)

When writing `ApplicationSpec` literals (and nested structs), omit zero-value fields instead of assigning explicit zero values.

Authentication details (OAuth vs non-OAuth, `OAuthAccount`, secret redaction rules):

- [references/auth.md](references/auth.md)

### HTTP: ALWAYS use env.HTTPClient and BodyBuffer

All HTTP calls must go through:

```go
res, err := c.env.HTTPClient.Do(req)
```

Build request bodies via `BodyBuffer` (pooled buffers, optional gzip, and retriable bodies). Full guidance (including redirects and flush/truncate patterns):

- [references/http.md](references/http.md)

Performance rule: stream JSON request bodies into `BodyBuffer` (avoid allocating intermediate payload `map[string]any` / slices / structs just to marshal). Only violate this with a clear, documented reason in code.

### Record.ID semantics for users (critical)

For `TargetUser`:

- `Record.ID` MUST be the application's **User ID** (Meergo terminology): the application's unique user identifier (the canonical ID assigned by the application, typically server-generated at create time).
- Empty `Record.ID` means "create". Non-empty `Record.ID` means "update that existing application user".
- Do not overload `Record.ID` with email/ext_id/phone or other natural keys, even if the vendor API accepts them as alternate identifiers. Model those as attributes and/or matching properties instead.
- In code, prefer using the helper methods on `connectors.Record`:
  - `record.IsCreate()` instead of `record.ID == ""`
  - `record.IsUpdate()` instead of `record.ID != ""`
- In `Upsert`, do not validate `record.ID` shape (e.g. numeric-only) and do not reject records based on it. If you need a typed ID for your API client, you may parse it, but parsing errors should be treated as an internal connector inconsistency/bug.

Details and examples:

- [references/users.md](references/users.md)

### Schemas and Meergo types (always relevant)

Meergo schemas use `github.com/krenalis/krenalis/tools/types`.

Guiding rules:

- Use `types.Object(...)`, `types.String()`, `types.Boolean()`, `types.Int(32)`, `types.Decimal(p,s)`, `types.DateTime()`, `types.Map(types.JSON())`, etc.
- Mark optionality correctly (`ReadOptional`, `CreateRequired`, `UpdateRequired`) and use `Nullable` only when JSON null is allowed.
  - These flags are role-dependent. A shared schema is fine only when source and destination differ by role-dependent flags alone.
  - `ReadOptional` describes the read path, not the fact that a property may be used for destination matching.
  - If the vendor metadata or docs indicate that a property is read-only or not writable, the destination schema must exclude it even if the source schema includes it.
- Avoid invalid property names (`types.IsValidPropertyName(...)`).

Full schema and value-mapping guidance (import/export canonical types, `types.Marshal`, missing vs null):

- [references/schemas-and-types.md](references/schemas-and-types.md)

### Iteration, batching, retries, and errors

Meergo record/event sequences have strict consumption semantics. Violating them can panic or drop data. Canonical iteration rules and batching pattern:

- [references/iteration-batching.md](references/iteration-batching.md)

Idempotency and retry behavior, and how to return per-item failures (`RecordsError` / `EventsError`), plus preview redaction rules:

- [references/retries-errors.md](references/retries-errors.md)

Never include PII in connector-authored error messages. Include a value only if both conditions hold: the context guarantees that the value cannot contain PII, even by mistake; and without that value the user cannot determine what the message refers to or how to fix it. When you author connector-side error messages in code, wrap property names, property paths, and any allowed setting names or setting values shown verbatim in `«...»` (for example `«metadata»`, `«address.first_name»`, `«base_url»`). Use `connectors.QuoteErrorTerm(...)` when the quoted term may contain `»`. Do this only for messages written in the connector code itself, not for raw error text returned by the upstream API. If you assign a message to `record.Err`, pass it to `events.Discard(err)`, or store it in `connectors.EventsError`, keep it fixed: do not interpolate values that vary with the specific bad input.

### Settings UI (only if HasSettings is true)

If `HasSettings: true` for Source and/or Destination, implement `ServeUI`. Otherwise do not implement `ServeUI` (registration can panic if you do). Details:

- [references/settings-ui.md](references/settings-ui.md)

### Users and events capabilities (load only what you implement)

- Users (RecordFetcher / RecordUpserter): [references/users.md](references/users.md)
- Events (EventSender): [references/events.md](references/events.md)
  - includes canonical `SendEvents` / `PreviewSendEvents` examples for:
    - one event per request
    - batch of events for one user
    - batch of events from mixed users
    - body-size-limited batching with `Postpone`
    - local validation vs `EventsError`

### Concurrency and state

Meergo can call `Upsert` / `SendEvents` concurrently on the same connector instance. Keep per-call state in locals and avoid shared mutable state (or guard it with a mutex). Do not create your own concurrency for processing records/events (no goroutines/worker pools).

## 4) Testing expectations (required)

Tests are part of the connector deliverable, not optional polish.

For every new Application connector, add request-level unit tests that cover at least:

- settings validation (`ServeUI("save", ...)`) when `HasSettings` is true
- request building (method/path/headers/body) and preview redaction (`[REDACTED]`, `[PIPELINE]`)
- batching/iteration behavior (`Postpone` / `Discard`) when batching exists
- per-item error mapping (`RecordsError` / `EventsError`) when the API supports it

For connectors implementing the common triad (`TargetUser` source + destination and `TargetEvent` destination), also add one hybrid live test that exercises:

- `Upsert` create path and update path
- `Records` pagination
- `SendEvents`
- run-scoped cleanup that does not affect unrelated existing data

Live tests must be skip-safe and CI-safe:

- gate them with explicit env vars and `t.Skip` when env vars are missing
- keep destructive cleanup scoped to run markers only
- poll for eventual consistency where needed (read-after-write and event visibility)
- if event verification is not implemented, report the degradation explicitly in the test/report

Keep this section high-level and follow the operational checklist in:

- [references/testing.md](references/testing.md)

## 5) Your output (what to implement)

When asked to implement a connector for a specific app, produce:

- a new package directory under `connectors/<code>/`
- Go code implementing only the interfaces implied by the declared spec
- clear, minimal documentation strings and code comments, following the Go standard library style
  - document package comments, exported declarations, and also non-exported package-level functions and non-exported methods
  - for functions and methods, exported or not, write a short leading comment whose first sentence starts with the identifier name and states what it does
  - keep these comments concise and factual; do not restate the full signature or add boilerplate prose
  - small local helpers inside a function do not need comments unless the logic is non-obvious
- safe, schema-correct behavior
- a test suite aligned with implemented capabilities (unit tests required; hybrid live test required for full triad connectors unless an explicit exception applies)

If the user did not specify whether the connector is source, destination, users, events, OAuth, etc.:

- perform **Discovery (required)** (section 1)
- propose a concrete default connector design based on the app API (roles/targets + OAuth yes/no)
- ask only the minimum clarifying questions that cannot be inferred reliably
- then implement the connector accordingly

Before finishing, ensure:

- registration spec matches implemented methods
- `ApplicationSpec.Code` is non-empty and contains only `[a-z0-9-]`
- OAuth + endpoint groups satisfy registry validation (if applicable)
- `ServeUI` exists iff `HasSettings` is true
- requests use BodyBuffer + `env.HTTPClient`
- embedded connector documentation matches actual implemented behavior (do not claim unsupported capabilities)
- package comments and function/method comments exist for exported and non-exported declarations that need them, with concise Go-style wording
- schemas match real API behavior: properties that may be absent on the read path are marked `ReadOptional`, and role-dependent schema flags are used coherently
- shared schemas may rely on Meergo applying the correct role semantics, while separate source/destination schemas avoid irrelevant flags unless a documented reason justifies them
- destination matching via `Records()` does not by itself justify source/read-only flags in a destination-only schema
- `Record.Attributes` never uses type-incompatible placeholders
- connector-authored error messages never include PII, and any echoed value satisfies the two conditions above (guaranteed non-PII and necessary to understand/fix the error)
- any raw upstream API error text that may contain PII is either not exposed, or is exposed only with a local `// TODO: please ...` comment requesting human review
- messages assigned to `record.Err`, passed to `events.Discard(err)`, or stored in `connectors.EventsError` are fixed and do not interpolate input-specific values
- tests are included and aligned with implemented capabilities; if you intentionally omit a test category, state why

### Final compliance checklist (required)

Before presenting the final result, run this checklist against your own implementation and report it:

- `Upsert` JSON is streamed directly into `BodyBuffer` (no intermediate payload map/struct just to encode)
- `SendEvents`/`PreviewSendEvents` JSON is streamed directly into `BodyBuffer` (no intermediate payload map/struct just to encode)
- if an events batch endpoint exists, `SendEvents` does not default to one-event requests
- `PreviewSendEvents` request shape matches one `SendEvents` call
- discovery used API specification first (or includes explicit negative evidence that no usable spec was found)
- `RecordSchema(..., role)` returns the correct schema for the requested role; non-writable/read-only properties are excluded from the destination schema when applicable
- response bodies are always closed, and connectors do not add manual drain code just to make connection reuse work
- package comments and leading Go-style comments cover exported declarations and the non-exported package-level functions and methods whose behavior is not obvious from the code

If any checklist item is violated, include a short `Skill deviations` section in the output with:

- violated rule
- exact file/line where it happens
- why the deviation is necessary
- why the compliant alternative is not feasible

## References

- [references/meergo-context.md](references/meergo-context.md)
- [references/application-spec.md](references/application-spec.md)
- [references/auth.md](references/auth.md)
- [references/settings-ui.md](references/settings-ui.md)
- [references/http.md](references/http.md)
- [references/schemas-and-types.md](references/schemas-and-types.md)
- [references/users.md](references/users.md)
- [references/events.md](references/events.md)
- [references/iteration-batching.md](references/iteration-batching.md)
- [references/retries-errors.md](references/retries-errors.md)
- [references/testing.md](references/testing.md)
