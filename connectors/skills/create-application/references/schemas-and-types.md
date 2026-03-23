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
- Use `Description` only when it adds information beyond the property name. Do not copy the property name into `Description`; if you have nothing more useful to say, leave it empty.
- Determine field types from official specifications and/or officially documented or observed payload shapes, not from field names. If a field's type remains ambiguous, treat that as an explicit assumption to verify rather than silently fixing the type by inference from the name.
- Be careful with reserved/invalid property names; prefer `types.IsValidPropertyName(...)` when mapping external field names.
- When building schemas from vendor-provided field lists (dynamic/custom fields), prefer `types.ObjectOf(...)` over `types.Object(...)` so in-flight schema expansions don't require a code release.

## Schema formatting (readability rule)

Do not condense schema composite literals into one-liners. For schema definitions, use multi-line composite literals in an idiomatic Go style so diffs and reviews are easy.

Example (preferred):

```go
return types.Object([]types.Property{
	{
		Name:         "event_name",
		Type:         types.String().WithMaxLength(255).WithPattern(eventNameRE),
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

## Express constraints in the schema (prefer schema over runtime checks)

If the application imposes constraints that are expressible in Krenalis schemas, encode them in the type returned by `RecordSchema` / `EventTypeSchema` instead of re-validating them in `Upsert` / `SendEvents`.

Common examples:

- Maximum string length: `types.String().WithMaxLength(255)`
- Maximum string bytes: `types.String().WithMaxBytes(1024)`
- String allowed values (enum-like): `types.String().WithValues("a", "b", "c")`
- String regex/pattern: `types.String().WithMaxLength(255).WithPattern(regexp.MustCompile("^[A-Za-z0-9_-]+$"))`
- Array size limits: `types.Array(types.String()).WithMinElements(1).WithMaxElements(100)`

This makes the constraint visible to Krenalis (and UIs), and avoids per-call defensive validation code in connectors.

If you need a constraint that is not covered above, check the `types.Type` methods in `tools/types/types.go` in this repo.

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
