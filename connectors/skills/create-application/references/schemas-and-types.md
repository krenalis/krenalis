# Schemas and data types

Krenalis uses `github.com/krenalis/krenalis/tools/types` to represent schemas.

You will define:

- **Record schema** (users) via `RecordSchema(...) (types.Type, error)`
- **Event type schema** (extra per-event values) via `EventTypeSchema(...) (types.Type, error)`

Guidelines:

- Use `types.Object([]types.Property{...})`, `types.String()`, `types.Boolean()`, `types.Int(32)`, `types.Decimal(p,s)`, `types.DateTime()`, `types.Map(types.JSON())`, etc.
- The connector will run only on 64-bit architectures, so `int`/`uint` are 64-bit in practice. Prefer `int`/`uint` over `int64`/`uint64` in connector code (IDs, counters, array indices, pagination, etc.) unless you have a strong reason to require a fixed-width type (for example: the API/library type is explicitly `int64`, you are manipulating unix timestamps as `int64`, or you need to document fixed-width semantics in the code).
- Mark optionality correctly:
  - **Read path:** `ReadOptional` (the property value may be omitted)
  - **Create:** `CreateRequired` (the property value is required when creating the record)
  - **Update:** `UpdateRequired` (the property value is required when updating the record)
  - Use `Nullable` if the property value may be **JSON null** (`nil` in Go).
- These are role-dependent flags. Krenalis applies the correct role semantics so the flags that do not matter for the chosen role are ignored.
- `ReadOptional` describes the read path, not the fact that a property may be used for destination matching.
- A shared schema is appropriate only when source and destination differ by role-dependent flags alone.
- If some fields are read-only or otherwise not writable, the destination schema must exclude them even if the source schema includes them.
- If you keep separate source and destination schemas in code, keep them role-coherent for readability: avoid `ReadOptional: true` in a destination-only schema, and avoid `CreateRequired` / `UpdateRequired` in a source-only schema unless there is a specific, documented reason.
- Concretely: if you build a schema inside a `role == connectors.Source` branch, do not use destination-only flags unless there is a specific, documented reason. If you build a schema inside a `role == connectors.Destination` branch, do not use source/read-only flags unless there is a specific, documented reason.
- Use `DisplayName` for the human-readable label of the property, and `Description` only when it adds information beyond that label. Do not copy the label into `Description`; if you have nothing more useful to say, leave it empty.
- When the application exposes its own label or description for a field, map them to `DisplayName` and `Description` instead of deriving them.
- Determine field types from official specifications and/or officially documented or observed payload shapes, not from field names. If a field's type remains ambiguous, treat that as an explicit assumption to verify rather than silently fixing the type by inference from the name.
- Do not default to a single check when mapping an external field name to a Krenalis property name; the right approach (verify against the application's own syntax, convert bidirectionally, or validate and drop) depends on the relationship between the application's naming syntax and Krenalis's. See [Property name syntax](#property-name-syntax) below.
- When building schemas from vendor-provided field lists (dynamic/custom fields), prefer `types.ObjectOf(...)` over `types.Object(...)` so in-flight schema expansions don't require a code release. See [Static vs. dynamic values in panicking `types` calls](#static-vs-dynamic-values-in-panicking-types-calls) below for the general rule.

## Schema formatting (readability rule)

Do not condense schema composite literals into one-liners. For schema definitions, use multi-line composite literals in an idiomatic Go style so diffs and reviews are easy.

Example (preferred):

```go
return types.Object([]types.Property{
	{
		Name:         "event_name",
		Type:         types.String().WithPattern(eventNameRE),
		CreateRequired: true,
	},
	{
		Name:         "value",
		Type:         types.Decimal(10, 2),
		ReadOptional: true,
	},
}), nil
```

That example illustrates read-side optionality. If you reuse the same static schema for both roles, do so only when source and destination differ by role-dependent flags alone. If the API distinguishes readable fields from writable ones, `RecordSchema(..., role)` must reflect that distinction for the requested role. If you keep a separate destination-only schema, omit `ReadOptional: true` unless there is a specific, documented reason to keep it. Symmetrically, if you keep a separate source-only schema, omit `CreateRequired` / `UpdateRequired` unless there is a specific, documented reason to keep them. Destination matching through `Records()` does not change this readability rule.

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

## Record attribute values (import/export)

Krenalis supports a canonical set of value types for each schema property type. Your connector must map between:

- API payload values (mostly strings / numbers / timestamps)
- `connectors.Record.Attributes` and `connectors.Event.Type.Values`

When in doubt, follow patterns in existing connectors.

### Import (connectors producing values, e.g. Records)

When filling `Record.Attributes` from API responses:

- For `string`/`text` you should return a Go `string` or `[]byte`.
- For `bool` return `bool`.
- For numeric types return `int`, `uint`, `float64`, or `decimal.Decimal` depending on the schema.
- For `datetime`/`date`/`time` fields you may return:
  - `time.Time`, or
  - a `string`/`[]byte` exactly as returned by the API; Krenalis will parse it (ISO 8601 by default, or using `ApplicationSpec.TimeLayouts` if you set it at registration), or
  - for `datetime`, a Unix-epoch value (string or float64) if `TimeLayouts.DateTime` uses `"unix"`, `"unixmilli"`, `"unixmicro"`, or `"unixnano"`.
- For `uuid` you may return a UUID string or a 16-byte slice (vendor dependent).
- For `ip` you may return a string, `net.IP`, or `netip.Addr`.
- Missing vs null:
  - **missing** means omit the key from `Attributes`
  - **null** means present the key with `nil` value and the schema property must be `Nullable`
  - note: for some nullable types, an empty string is interpreted as `nil` (see `create-integration/data-values.md`); do not rely on this unless your API actually returns empty strings for missing values.

Practical rule: for time-typed *attributes* (`datetime`/`date`/`time` in your schema), prefer returning the upstream string and configuring `ApplicationSpec.TimeLayouts` when needed, instead of writing custom parsing code in the connector for each attribute.

### Export (connectors receiving values, e.g. Upsert, SendEvents)

Krenalis passes canonical types to connectors:

- `string`, `bool`, `int`, `uint`, `float64`, `decimal.Decimal`, `time.Time` (UTC)
- `uuid`: `string`
- `json`: `json.Value` (note: missing key vs `json.Value("null")` are distinct)
- `ip`: `string`
- `array(T)`: `[]any`
- `object` and `map(T)`: `map[string]any`

When encoding `EventType.Values` with a known schema, prefer schema-aware marshaling:

```go
params, err := types.Marshal(event.Type.Values, event.Type.Schema)
```

This encodes values according to schema expectations.

Krenalis guarantees that the values you receive in `event.Type.Values` (in `SendEvents` / `PreviewSendEvents`) conform to `event.Type.Schema` (including required fields for event sending, i.e. `CreateRequired`).

For `Upsert`, values in `Record.Attributes` follow the canonical Go types described above for the given schema; do not add defensive type-checks for basic types. Only validate additional API-specific rules that cannot be expressed in the schema.

Note: `types.Marshal` does not validate; its behavior is undefined if you pass values that do not match the schema (for example, if the connector mutates values to incompatible types).
