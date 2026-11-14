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
- Create `workspace_environment` with values `production` and `development`.
  Add `workspaces.environment` as non-null with default `production`, so
  existing release workspaces retain production behavior.
- Create `simulated_system_status` with values `Preparing`, `Available`,
  `Failed`, and `Deleting`, and create `simulated_systems` as defined in
  `schema.sql`, including its checkpoint, writer generation, run claim,
  claim timestamp, primary key, workspace foreign key, workspace index,
  `simulated_systems_workspace_id_key`, and
  `simulated_systems_claimed_run_key`. The base release has no simulated
  systems or older simulated status values to convert.
- Add nullable `connections.simulated_system` and
  `pipelines_runs.simulated_system` columns. Preserve existing rows with
  null in these columns. Create
  `connections_workspace_simulated_system_fkey` on
  `(workspace, simulated_system)` referencing
  `simulated_systems(workspace, id)` with `ON DELETE RESTRICT`;
  `connections_workspace_simulated_system_idx` on the same columns; and
  `connections_simulated_system_role_key`, unique on
  `(simulated_system, role)` where `simulated_system IS NOT NULL`.
- Create `pipelines_one_live_simulated_system_run_idx`, unique on
  `pipelines_runs(simulated_system)` where
  `simulated_system IS NOT NULL AND node IS NOT NULL AND end_time IS NULL`.
- Add `CreateSimulatedSystem`, `DeleteSimulatedSystem`,
  `MarkSimulatedSystemDeleting`, `UpdateSimulatedSystem`, and
  `UpdateSimulatedSystemGeneration` to `notification_name`.
