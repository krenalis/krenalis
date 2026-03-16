# Testing expectations (Application connectors)

Use this file as the single operational checklist for connector tests.

## Baseline policy

For every connector, unit tests are required.

Focus first on:

- settings validation (`ServeUI("save", ...)`)
- request building (preview) including redaction behavior
- per-item error mapping (`RecordsError` / `EventsError`)
- batching behavior with Postpone/Discard

For connectors implementing users (`Records` + `Upsert`) and events (`SendEvents`), testing must combine:

- request-level unit tests for deterministic payload/headers behavior
- one hybrid live test to validate integration behavior against the real API

## Prefer core/testconnector helpers

For connector tests, prefer `github.com/meergo/meergo/core/testconnector` instead of ad-hoc test scaffolding.

Use these helpers as defaults:

- `testconnector.NewApplication(code, settings)` to create a connector instance with JSON settings.
- `testconnector.NewEventsIterator(events)` to build a `connectors.Events` sequence for `SendEvents` / `PreviewSendEvents`.
- `testconnector.ReceivedEvent(map[string]any)` to build `connectors.ReceivedEvent` test inputs.
- `testconnector.TransformEvent(schema, event, mapping)` to produce schema-aligned event values for tests.
- `testconnector.CaptureRequestContextKey` to capture the built HTTP request from context in send/preview tests.
- `testconnector.DecodeNDJSON(body, encoding)` when asserting NDJSON payloads (including gzip).

Only avoid these helpers when a test requires behavior they do not cover.

## Minimum test checklist (required)

- registration compiles and does not panic (spec matches implemented interfaces; EndpointGroups are valid)
- settings UI:
  - `ServeUI("load", ...)` returns UI + settings
  - `ServeUI("save", ...)` validates and persists (including invalid settings cases)
  - for security-sensitive settings, include boundary tests for both min and max length (for example API keys/tokens)
- request building:
  - at least one test that asserts the built HTTP request path/method/headers
  - preview redacts secrets (`[REDACTED]`) and replaces pipeline IDs with `[PIPELINE]` when applicable
- endpoint groups:
  - if multiple endpoint groups are configured, add assertions that representative requests match the intended group patterns (so endpoint families are not accidentally merged)
- batching/iteration:
  - at least one test around `Postpone`/`Discard` behavior for body-size or max-items limits
- error mapping:
  - at least one test that returns `RecordsError` or `EventsError` and asserts index mapping stability

If the application supports event batch ingestion, add event-specific tests:

- one `SendEvents` test proving multiple events are sent in a single HTTP request
- one limit test proving overflow events are postponed (`events.Postpone()`) when max-events/body-size would be exceeded
- one test for per-event error mapping from API responses when partial success is supported (for example index-based errors)

Use patterns already present in connector tests, and consider injecting `context` keys if needed for deterministic behaviors (some existing connectors do this for tests).

## Hybrid live tests (required for full application connectors)

When the connector supports `Records`, `Upsert`, and `SendEvents`, add one live test that validates the complete flow against the vendor API.

Prefer using shared helpers under `connectors/internal/livetest`:

- `livetest.Run(t, adapter, cfg)` for the orchestration
- `livetest.DefaultCaseConfig()` for sane defaults
- `livetest.NewRunID(...)` and adapter-level run markers to isolate data

Adapter responsibilities (implement all):

- create a configured connector (`NewConnector`)
- provide user schema (`UserSchema`)
- build create records and update records (`BuildCreateRecords`, `BuildUpdateRecords`)
- find and cleanup users created by the current run only (`FindUsersByRunID`, `CleanupUsersByRunID`)
- build sendable events and expected verification facts (`BuildEvents`)
- optionally verify events from vendor-side read endpoints (`SupportsEventVerification`, `VerifyEvents`)

Minimum live assertions:

- `Upsert` create path creates all expected users
- `Upsert` update path updates at least one previously created user
- `Records` pagination reads at least the configured minimum pages
- `SendEvents` sends all requested events

Live test safety rules:

- gate live tests with explicit env vars and `t.Skip` when not configured
- never target broad destructive cleanup; delete only run-scoped data
- handle eventual consistency with polling (both for read-after-write and event visibility)
- keep API key/token permissions documented in the connector README/test comments

Execution integration:

- place live tests in the connector package (`connectors/<code>`) so they are part of normal `go test` discovery
- ensure they are skip-safe when env vars are missing, so `go run test/commit/commit.go` can run without forcing all credentials
- they are skipped together with other connector tests when `-no-connector-tests` is used

## Explicit exceptions

You may skip the hybrid live test only when at least one of these is true:

- the connector does not implement the full triad (`Records` + `Upsert` + `SendEvents`)
- the user explicitly requests to skip live tests
- vendor/API constraints make a safe live test impossible (for example no isolatable test data and no scoped cleanup path)

If skipped, state the reason explicitly in the implementation summary.
