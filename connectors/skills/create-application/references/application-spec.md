# Application specification and packaging

Treat `connectors.ApplicationSpec` and `connectors.RegisterApplication` as the registration contract. Confirm every field and interface against `connectors/applications.go` and `connectors/registry.go`; do not infer fields from another connector type.

## Declare only implemented capabilities

Set:

- `Code`, `Label`, and `Categories`;
- `AsSource` only for implemented source-user behavior;
- `AsDestination` only for implemented user and/or event behavior;
- `Terms` only when the connector exposes users and provider-specific terminology improves the UI;
- `OAuth`, `EndpointGroups`, and `TimeLayouts` only when supported by evidence.

For each non-nil role, set a non-zero supported `Targets` bitmask and follow the capability matrix established by the main workflow. Do not declare a role merely to make an interface shape look convenient.

Registration validation panics on capability/interface mismatches. Add a compile-time assertion for each substantial interface when it makes the implementation easier to audit, but rely on a registration test as well.

## Register and construct

Register once from `init`:

```go
func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       "provider-code",
		Label:      "Provider",
		Categories: connectors.CategorySaaS,
		// Add only evidenced role, auth, endpoint, and documentation fields.
	}, New)
}
```

Follow the built-in convention `func New(*connectors.ApplicationEnv) (*Connector, error)` unless the implementation has a concrete reason to use another type accepted by `RegisterApplication`. Retain the environment when later methods need it; do not create a separate `http.Client`, rate limiter, or retry loop. Avoid network calls in `New` unless construction cannot be correct without them.

## Package the connector

Use `connectors/<code>` for a built-in connector. Follow nearby packages for copyright header, package documentation, trademark disclaimer, and file organization, but split files only when complexity warrants it.

Embed concise role documentation through `connectors.RoleDocumentation`. Event-only connectors commonly share `documentation/overview.md`; connectors with distinct source and destination behavior commonly use role subdirectories. The layout is a repository convention, not a framework requirement.

Add a built-in production package to the blank imports in `main.go`. Registration happens only when the package is imported. `test/krenalistester/test_imports.go` contains the additional packages needed by its test executable, not a catalogue of every connector; change it only when that executable needs the new package.

## Check runtime compatibility

Before finishing, compare the spec with its runtime consumers. Known current mismatches must be surfaced rather than worked around:

- `connectors.RegisterApplication` accepts source-only applications, but `core/internal/state/load.go` currently dereferences `AsDestination` while loading connector state.
- Public comments around `Records.Peek` and the first-record `Records.Postpone` restriction do not fully match the tested `core/internal/connections/appwriter` iterator. Follow the tested runtime behavior for implementation, and report the documentation mismatch when it affects the design.
- Public `connectors.Record.Err` is intended to report a provider item failure, but the current application-record runtime does not copy a connector-provided `Record.Err`. Do not rely on it for observable source item errors until that mismatch is resolved.

Do not add an invented destination, misleading stub behavior, or a silent panic merely to pass registry validation.
