# Schemas and types

Build schemas from provider evidence and the selected role. Do not expose response fields as writable merely because the provider returns them.

## Model properties accurately

Use `tools/types.Type` and `types.Property`. The property fields with framework meaning are:

- `CreateRequired` and `UpdateRequired` for destination writes;
- `ReadOptional` for source reads that may omit the property;
- `Nullable` only when null is a valid value;
- `Prefilled` for an event mapping expression, not a provider default;
- `Description` for useful mapping context.

`types.AsRole` removes create/update requirements for a source and removes `ReadOptional` for a destination. It does not remove provider read-only properties. Use distinct source and destination schemas, or explicitly filter the shared base, when writability differs.

Return an object schema for records. Use `types.Object` only when locally defined properties are known valid; use `types.ObjectOf` when provider-derived names can fail validation. Provider property names must satisfy `types.IsValidPropertyName`. Skip or deterministically map invalid and colliding names only when the resulting data behavior is explicit and tested.

Use the narrowest accurate type and provider-documented bounds. Preserve decimal and timestamp precision instead of converting through `float64` or inventing `time.Now`. Configure `ApplicationSpec.TimeLayouts` only when the provider format differs from the framework defaults.

## Choose static or dynamic schemas

Use a static schema when fields are fixed. Fetch metadata dynamically only when tenant-specific fields are needed for the selected capability. Define how unsupported provider types, duplicates, invalid names, pagination, and metadata failures behave.

The core caches recent record schemas per application instance and role. Do not add a connector-local TTL cache solely to duplicate that ordinary runtime behavior; an inner TTL also cannot refresh through an outer cache. Surface any requirement for mid-instance metadata refresh as a framework concern. Add separate caching only for an evidenced caller or performance requirement, and make it concurrency-safe.

For event destinations:

- `EventTypes` returns the supported type IDs and mapping metadata;
- `EventTypeSchema` returns the values required for that type;
- return an invalid `types.Type{}` when the type needs no extra values;
- return `connectors.ErrEventTypeNotExist` for an unknown type.

## Encode values at the boundary

Return Go values that conform to the requested record schema. When encoding schema-governed values, use `types.Marshal` where it removes ambiguity; otherwise encode the natural provider payload directly with `BodyBuffer`. Do not build a second generic map solely to satisfy a stylistic rule.

Test schemas in every exposed role, including required/read-optional flags, provider read-only exclusions, invalid dynamic names, nullability, precision, and event prefilled expressions.
