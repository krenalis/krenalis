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
