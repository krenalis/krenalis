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
