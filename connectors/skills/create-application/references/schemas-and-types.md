# Schemas and types

Build schemas from provider evidence and the selected role. Do not expose response fields as writable merely because the provider returns them.

## Model properties accurately

Use `tools/types.Type` and `types.Property`. The property fields with framework meaning are:

- `CreateRequired` and `UpdateRequired` for destination writes;
- `ReadOptional` for source reads that may omit the property;
- `Nullable` only when null is a valid value;
- `Prefilled` for an event mapping expression, not a provider default;
- `DisplayName` for the human-readable label;
- `Description` for useful mapping context beyond that label.

When the provider supplies field labels or descriptions, map them to `DisplayName` and `Description` instead of deriving them. Leave `Description` empty when it would only repeat the label.

`types.AsRole` removes create/update requirements for a source and removes `ReadOptional` for a destination. It does not remove provider read-only properties. Use distinct source and destination schemas, or explicitly filter the shared base, when writability differs.

Return an object schema for records. Use `types.Object` only when locally defined properties are known valid; use `types.ObjectOf` when provider-derived names can fail validation. Follow [Property name syntax](#property-name-syntax) to validate or map external names, and test the resulting data behavior. Return a schema error for collisions or invalid metadata outside the documented name-syntax exception.

Use the narrowest accurate type and provider-documented bounds. Preserve decimal and timestamp precision instead of converting through `float64` or inventing `time.Now`. Configure `ApplicationSpec.TimeLayouts` only when the provider format differs from the framework defaults.

## Choose static or dynamic schemas

Use a static schema when fields are fixed. Fetch metadata dynamically only when tenant-specific fields are needed for the selected capability. Define how unsupported provider types, duplicates, invalid names, pagination, and metadata failures behave.

The core caches recent record schemas per application instance and role. Do not add a connector-local TTL cache solely to duplicate that ordinary runtime behavior; an inner TTL also cannot refresh through an outer cache. Surface any requirement for mid-instance metadata refresh as a framework concern. Add separate caching only for an evidenced caller or performance requirement, and make it concurrency-safe.

For event destinations:

- `EventTypes` returns the supported type IDs and mapping metadata;
- `EventTypeSchema` returns the values required for that type;
- return an invalid `types.Type{}` when the type needs no extra values;
- return `connectors.ErrEventTypeNotExist` for an unknown type.

## Static vs. dynamic values in panicking `types` calls

Many `tools/types` constructors and modifiers panic on an invalid argument instead of returning an error: `types.Object(...)`, `WithValues(...)`, `WithPattern(...)`, `WithMaxLength(...)`, and most other `With*` methods. Calling one of these with an argument that is static — a literal, or a value fully determined by connector code — is fine: a panic there means a bug in the connector, caught during development or testing.

Calling one with a dynamic argument — anything read from an API response, vendor metadata, or other external input — is not fine: a value the provider changes or corrupts would crash the connector at runtime. Before such a value reaches a panicking call, do one of:

- Validate the value exhaustively first, when that is possible without excessive complexity.
- Prefer the corresponding error-returning form when one exists, and propagate its error. Today the only such form in `tools/types` is `types.ObjectOf(...)`, the error-returning counterpart of `types.Object(...)`.

For a panicking call with no error-returning counterpart (`WithValues`, `WithPattern`, `WithMaxLength`, etc.), exhaustive validation of the dynamic value before the call is the only option.

Either way, do not handle a validation failure by silently dropping just that property or field and continuing to build the schema; that produces a schema Krenalis and its users cannot see is incomplete. Return an error from the schema method (`RecordSchema` / `EventTypeSchema`) so the connector reports a clear failure instead.

The one documented exception is an application property name that fails Krenalis's property name syntax with no viable mapping back to the application; see [Property name syntax](#property-name-syntax) below for when dropping that property is the correct, documented behavior instead of returning an error.

## Property name syntax

Krenalis property names must satisfy `types.IsValidPropertyName(...)`: an ASCII letter or underscore, followed by ASCII letters, digits, or underscores (`tools/types/properties.go`). Applications rarely use exactly this syntax for their own field names, so work out the relationship between the application's syntax and Krenalis's before writing the mapping code, in this order:

1. **Learn the application's property-name syntax** from its API/vendor documentation. If it is not documented, the only assumption you can make is that a property name can be an arbitrary sequence of Unicode characters.
2. **If the application's syntax is a subset of Krenalis's** (every name the application can produce is already a valid Krenalis property name), do not check the name with `types.IsValidPropertyName(...)`: it can never fail, so the check is redundant. Instead, validate the name against the application's own documented syntax. A name that fails that check means the connector's assumption about the application is wrong — a connector bug you want to surface — not a Krenalis validity problem.
3. **If the application's syntax only partially overlaps Krenalis's, and a bidirectional mapping between the two exists,** validate the name against the application's syntax as in the previous case, and additionally convert it in both directions: into a valid Krenalis property name when building the schema and reading values, and back to the original application name when writing values through `Upsert` / `SendEvents`.
4. **Otherwise**, validate the name with `types.IsValidPropertyName(...)`. When it fails, discard the property: do not include it in the schema.

This decision process concerns property *names* only; it does not apply to other fields such as `Description` or `DisplayName`.

## Express constraints in the schema (prefer schema over runtime checks)

If the application imposes constraints that are expressible in Krenalis schemas, encode them in the type returned by `RecordSchema` / `EventTypeSchema` instead of re-validating them in `Upsert` / `SendEvents`.

Common examples:

- Maximum string length: `types.String().WithMaxLength(255)`
- Maximum string bytes: `types.String().WithMaxBytes(1024)`
- String allowed values (enum-like): `types.String().WithValues("a", "b", "c")`
- String regex/pattern: `types.String().WithPattern(regexp.MustCompile("^[A-Za-z0-9_-]{1,255}$"))`
- Array size limits: `types.Array(types.String()).WithMinElements(1).WithMaxElements(100)`

This makes the constraint visible to Krenalis (and UIs), and avoids per-call defensive validation code in connectors.

If you need a constraint that is not covered above, check the `types.Type` methods in `tools/types/types.go` in this repo.

### String constraint combinations

For each string type, choose one of these constraint forms, or leave it unconstrained:

- Length limits: `WithMaxLength` (Unicode code points), `WithMaxBytes` (UTF-8 bytes), or both. Each limit must be in `[1, types.MaxStringLen]` and may be set only once.
- Allowed values: `WithValues(...)`, without a pattern or length limits. The list must be non-empty and contain valid UTF-8 strings; it may be set only once.
- Pattern: `WithPattern(...)`, without allowed values or length limits. The pattern must be non-nil and may be set only once.

Unsupported combinations panic in the Go type methods, regardless of the order of modifier calls. JSON deserialization rejects the same combinations with an error.

Choose a representation that preserves the provider's set of allowed values. A constraint already implied by that representation needs no separate modifier. For example, `WithPattern(regexp.MustCompile("^[A-Za-z0-9_-]{1,255}$"))` enforces both the allowed alphabet and a length of 1–255 characters; every matched character is ASCII and occupies one UTF-8 byte. This equivalence between character and byte counts does not hold for arbitrary Unicode strings. Likewise, `WithValues(...)` alone is sufficient when every listed value already satisfies the provider's pattern and length requirements.

If no supported schema form expresses the full rule, choose one that accepts all valid provider values and enforce the remaining requirements in the connector. Do not silently discard constraints or exclude valid values merely to fit a schema form.

For dynamic fields, validate metadata before constructing types and compile provider-supplied patterns with `regexp.Compile`, handling compilation errors. Successful compilation alone does not establish equivalence with the provider's regex semantics. Return an error from the schema method for malformed metadata or rules the connector cannot enforce faithfully; do not rely on recovering a constructor panic. Use `regexp.MustCompile` only for fixed patterns authored in the connector.

Test the schemas returned by `RecordSchema` / `EventTypeSchema` with accepted and rejected values, including length boundaries. Also test any requirements enforced separately in the connector. If metadata controls constraints, cover valid combinations and verify that invalid metadata returns an error rather than panicking.

## Encode values at the boundary

Return Go values that conform to the requested record schema. When encoding schema-governed values, use `types.Marshal` where it removes ambiguity; otherwise encode the natural provider payload directly with `BodyBuffer`. Do not build a second generic map solely to satisfy a stylistic rule.

Test schemas in every exposed role, including required/read-optional flags, provider read-only exclusions, invalid dynamic names, nullability, precision, and event prefilled expressions.
