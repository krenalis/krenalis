# Release upgrade

Base release: `v0.48.0`

What the updater of the next release must do to upgrade a database of the
base release. See `.agents/skills/release-upgrade/SKILL.md`.

- Replace every `{"kind":"year"}` type with a signed `int` of `bitSize: 16`
  and `semantics: "year"`, recursively in
  `workspaces.profile_schema`, `workspaces.alter_profile_schema_schema`,
  types in `workspaces.alter_profile_schema_operations`, and
  `pipelines.in_schema` and `pipelines.out_schema`. Preserve all property
  metadata and enclosing array, map, and object types. Set `minimum: 1` and
  `maximum: 9999` explicitly to preserve the existing range. This
  representation keeps the existing Go and JSON values and remains
  compatible with the existing warehouse column types; no warehouse DDL
  or record conversion is required.
- In `pipelines.filter`, recursively convert conditions on properties whose
  input type was `year`: `IsBefore` to `IsLessThan`, `IsOnOrBefore` to
  `IsLessThanOrEqualTo`, `IsAfter` to `IsGreaterThan`, and `IsOnOrAfter` to
  `IsGreaterThanOrEqualTo`. Identify these properties from the original
  input schema before replacing its types. Preserve condition values,
  nested groups, and conditions on other types. Complete both schema and
  filter conversions in the same upgrade transaction before loading state.

- Preserve existing JSON String operands in `pipelines.filter`. They are now
  read as typed JSON Strings rather than both text and, optionally, a number.
  Do not infer numeric kinds or rewrite filters to preserve the previous
  behavior; no operand conversion is required for this change. Document in
  the release notes that existing filters may return different results under
  typed comparisons and the current rules for absent or inapplicable values.
  Ordinary integer operands remain JSON numbers at every bit size.
- Document the default event catalog's ingestion change in the release notes.
  In every workspace, the collector validates and normalizes traits and
  properties when a matching definition specifies a schema. A schema
  violation discards the entire event, including identity processing; the
  request still succeeds, with no dedicated log, metric or Admin observer
  entry. Normalization can change timestamps to UTC, decimal representation
  and the order of known object keys. It introduces no additional coercions
  and does not rewrite historical events. This ingestion change needs no
  database upgrade.
- Remove `workspaces.warehouse_mcp_settings` and
  `workspaces.kms_encrypted_warehouse_mcp_settings_key`, discarding the stored
  warehouse MCP credentials and their dedicated encryption key without
  conversion. Preserve access keys of type MCP. Perform the upgrade offline:
  stop all nodes before removing the columns, then restart all nodes on the
  same new version; old nodes still write both columns. MCP profile access now
  uses the main connection for every valid workspace-scoped MCP key, including
  existing keys without warehouse MCP settings and keys formerly restricted
  by read-only warehouse grants. Document this access change in the release
  notes so administrators can review and revoke existing keys as needed.
  Limits on the former role or dedicated warehouse do not transfer
  automatically to the main connection: review statement timeouts and the
  main Snowflake warehouse and its resource monitor separately. MCP argument,
  response and rate limits do not bound the duration or warehouse work of an
  individual query.
