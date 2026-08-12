---
name: create-application
description: Create, extend, or complete a Krenalis Application connector in Go. Use when implementing or reviewing provider integrations for user import or upsert, event delivery, authentication, settings UI, schemas, HTTP behavior, or connector tests.
---

# Create an application connector

Implement only the provider capabilities requested by the task. Do not default to a source-user, destination-user, and destination-event combination.

## Establish the contract

1. Inspect the task, the target package if it exists, and the current public API in `connectors/applications.go`, `connectors/connectors.go`, `connectors/ui.go`, and `connectors/registry.go`.
2. Declare the capability matrix before choosing endpoints:

   | Role | Target | Required implementation |
   | --- | --- | --- |
   | Source | User | `RecordFetcher` |
   | Destination | User | `RecordUpserter` |
   | Destination | Event | `EventSender` and a non-`None` `SendingMode` |

   Source applications currently support only `TargetUser`. `RecordUpserter` embeds `RecordFetcher`; a destination-user provider therefore needs usable schema, fetch, and upsert behavior. A destination may combine `TargetUser` and `TargetEvent` only when both are useful and supported.
3. For a new provider or capability, follow [provider discovery](references/provider-discovery.md). If the request names only a provider, choose the smallest useful, well-supported capability set and state the assumption; do not manufacture parity with existing connectors.
4. Record material decisions in a compact evidence ledger: decision, source, and classification. Use these classifications explicitly when resolving conflicts:
   - framework contract;
   - tested runtime invariant;
   - repository convention;
   - intentional implementation policy;
   - provider requirement;
   - connector-specific heuristic;
   - legacy or unsupported pattern.
5. Resolve repository evidence in this order: current public API and registry validation, runtime consumers and focused tests, representative connectors, then human documentation. Treat an existing connector as an example, never as proof of a universal rule.

Do not conceal a framework mismatch with dummy capabilities or misleading stubs. Surface it and stop when it blocks correctness.

## Load only relevant guidance

Always read [application specification and packaging](references/application-spec.md) and [testing](references/testing.md). Then load only the references needed for the selected capabilities:

| Concern | Reference |
| --- | --- |
| Provider endpoints and feasibility | [Provider discovery](references/provider-discovery.md) |
| API key, bearer, basic, or OAuth | [Authentication](references/auth.md) |
| Connection configuration | [Settings UI](references/settings-ui.md) |
| User and event value shapes | [Schemas and types](references/schemas-and-types.md) |
| User import or upsert | [Users](references/users.md) |
| Event destination | [Events](references/events.md) |
| Multi-item consumption or request limits | [Iteration and batching](references/iteration-batching.md) |
| Requests, endpoint groups, retries, and errors | [HTTP behavior](references/http.md) |

## Implement in evidence-driven order

1. Select endpoints and authentication only after verifying their fit with the chosen framework methods.
2. Design `ApplicationSpec`, role-specific documentation, schemas, settings, endpoint groups, and outcome mapping before writing transport code.
3. Keep provider-derived constants and non-obvious behavior traceable to the relevant official documentation in nearby comments or the evidence ledger.
4. Register the package from `init`, implement the exact interfaces declared by the spec, and keep the constructor cheap. Load settings at the point they are needed unless constructor validation is essential.
5. Use `ApplicationEnv.HTTPClient` for provider requests and its `BodyBuffer` for request bodies. Close every response body and every `BodyBuffer`. Handle every documented success and failure status explicitly.
6. Keep connector-authored errors free of secrets and user or event data. Use `connectors.QuoteErrorTerm` for dynamic names intended for UI-facing errors; expose provider messages only after establishing that they cannot contain sensitive payload data.
7. Make shared connector state safe for the concurrency promised by the public interfaces. Honor `context.Context`; do not let background work outlive the method call.
8. Add ordinary Go comments for exported declarations and non-obvious invariants. Do not add comments mechanically to every unexported declaration.

Prefer direct, readable encoding with `BodyBuffer.Encode`, `EncodeKeyValue`, or its write methods according to the provider format. Neither direct token streaming nor building an intermediate payload is universally required; choose the clearest approach that preserves iterator and rollback semantics.

## Complete the integration

- Add only the role documentation the connector actually exposes. Follow the package's established embedded Markdown layout.
- For a built-in production connector, add its blank import to `main.go` so registration runs. Add it to another executable or test import list only when that executable needs it; `test/krenalistester/test_imports.go` is not a catalogue of every connector.
- Format changed Go files and run focused deterministic tests. Do not use the repository-wide commit workflow as a substitute for targeted validation.
- Re-read every changed file, check referenced paths and symbols, and inspect the final diff for unrelated changes.
- Report implemented capabilities, evidence-based choices, tests run, live tests intentionally skipped, and unresolved framework or provider ambiguities.
