# Testing and validation

Make ordinary tests deterministic, credential-free, and network-free. Test the selected capability deeply instead of requiring a fixed source/users/events test triad.

## Build focused unit tests

Use standard Go tests, `httptest`, or a controlled transport. `core/testconnector` currently provides useful public helpers:

- `NewApplication` for constructing a registered application with settings;
- `CaptureRequestContextKey` for capturing a request;
- `DecodeNDJSON` for NDJSON bodies;
- `ReceivedEvent` and `TransformEvent` for event fixtures;
- `NewEventsIterator` for simple event iteration.

There is no `connectors/internal/livetest` package and no `livetest.*` API. Do not reference or recreate that nonexistent harness.

Use helpers only for behavior they model faithfully. In particular, the current `testconnector.NewEventsIterator` seeds `SameUser` from `UserID` but compares subsequent events by `AnonymousID`, unlike production grouping. Do not use it alone to prove `SameUser` behavior when those identifiers differ; supplement it with a focused local fake or the production iterator tests in `core/internal/collector/sender`.

Include tests relevant to the capability:

- registration and declared-interface alignment;
- settings load/save/validation and secret handling;
- source/destination schemas and dynamic-field edge cases;
- pagination, cursor, `updatedAt`, IDs, timestamps, and final `io.EOF`;
- create/update consumption, batching, postpone/discard, and `RecordsError` indices;
- event type schemas, mappings, preview redaction, batching, and `EventsError` indices;
- endpoint matching, retries, idempotency, status mapping, response closure, and cancellation;
- concurrency or `go test -race` when the implementation has shared mutable state.

Assert request method, URL, headers, body, ordering, and error privacy. Exercise boundary values, not only a happy-path fixture.

## Keep live tests opt-in

Add a live test only when it validates a provider behavior that deterministic tests cannot establish and its operational risk is acceptable. Name it clearly, read credentials from explicit environment variables, and call `t.Skip` when any prerequisite is absent. Use a unique run marker, the smallest safe dataset, and cleanup when the API permits it.

Do not assume existing connector packages are safe precedents: some ordinary tests currently fail when credentials are missing, while newer live tests skip correctly. Never make a normal `go test` depend on real credentials, account state, quota, or internet access.

## Validate proportionally

At minimum:

1. run `gofmt` on changed Go files;
2. run `go test ./connectors/<code>`;
3. run focused runtime tests for any iterator, HTTP, schema, or registry contract relied upon;
4. compile the built-in entry point after adding its blank import when registration/packaging changed;
5. re-read changed files and verify every referenced symbol, path, embed, and command;
6. inspect `git diff --check`, `git status --short`, and the final diff scope.

Do not run a broad mutating commit helper merely to claim validation. Report exact commands, results, skipped live coverage, and any test that could not run.
