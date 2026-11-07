# Release upgrade

Base release: `v0.46.0`

What the updater of the next release must do to upgrade a database of the
base release. See `.agents/skills/release-upgrade/SKILL.md`.

- Add `identity_resolution_runs` and
  `identity_resolution_runs_workspace_start_idx`, as defined in `schema.sql`.
- Backfill one running entry in `identity_resolution_runs` for each
  workspace whose `ir_id` is not null, using `ir_id` as `id`, the workspace's
  `id` as `workspace`, and `ir_start_time` as `start_time`. Leave `end_time`
  and `error` null, and preserve the workspace's Identity Resolution state,
  so pending operations can be finalized after the upgrade. Do not fabricate
  completed runs from `ir_end_time`, which does not identify the operation
  or distinguish success from failure.
- Add `identity_resolution_metrics` as defined in `schema.sql`. Leave it
  empty: historical resolution counts were not recorded in Krenalis's
  database.
- Add `workspaces.profile_role_first_name`, `profile_role_last_name`,
  `profile_role_email`, `profile_role_country`, and `profile_role_photo` as
  `varchar NOT NULL DEFAULT ''`. Initialize all existing rows to `''`, so
  no property role is assigned; do not copy the old UI preferences.
- Add `workspaces.alter_profile_schema_role_first_name`,
  `alter_profile_schema_role_last_name`, `alter_profile_schema_role_email`,
  `alter_profile_schema_role_country`, and `alter_profile_schema_role_photo`
  as `varchar NOT NULL DEFAULT ''`. Initialize all existing rows to `''`,
  including workspaces with a profile schema alteration in progress; leave
  the rest of the pending operation unchanged.
- Drop `workspaces.ui_profile_image`, `ui_profile_first_name`,
  `ui_profile_last_name`, and `ui_profile_extra`, discarding their values.
- Replace every `{"kind":"year"}` type with a signed `int` of `bitSize: 16`,
  `semantics: "year"`, `minimum: 1`, and `maximum: 9999`, recursively in
  `workspaces.profile_schema`, `workspaces.alter_profile_schema_schema`,
  types in `workspaces.alter_profile_schema_operations`, and
  `pipelines.in_schema` and `pipelines.out_schema`. Preserve all property
  metadata and enclosing array, map, and object types. This representation
  keeps the existing Go and JSON values and the warehouse column types;
  no warehouse DDL or record conversion is required.
- In `pipelines.filter`, recursively convert conditions on properties whose
  input type was `year`: `IsBefore` to `IsLessThan`, `IsOnOrBefore` to
  `IsLessThanOrEqualTo`, `IsAfter` to `IsGreaterThan`, and `IsOnOrAfter` to
  `IsGreaterThanOrEqualTo`. Identify these properties from the original
  input schema before replacing its types. Preserve condition values,
  nested groups, and conditions on other types. Complete both schema and
  filter conversions in the same upgrade transaction before loading state.
