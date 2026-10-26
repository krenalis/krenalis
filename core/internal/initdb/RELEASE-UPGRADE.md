# Release upgrade

Base release: `v0.46.0`

What the updater of the next release must do to upgrade a database of the
base release. See `.agents/skills/release-upgrade/SKILL.md`.

- Add `workspaces.profile_role_first_name`, `profile_role_last_name`,
  `profile_role_email`, `profile_role_country`, and `profile_role_photo` as
  `varchar NOT NULL DEFAULT ''`. Initialize all existing rows to `''`, so
  no Profile role is assigned; do not copy the old UI preferences.
- Add `workspaces.alter_profile_schema_role_first_name`,
  `alter_profile_schema_role_last_name`, `alter_profile_schema_role_email`,
  `alter_profile_schema_role_country`, and `alter_profile_schema_role_photo`
  as `varchar NOT NULL DEFAULT ''`. Initialize all existing rows to `''`,
  including workspaces with a profile schema alteration in progress; leave
  the rest of the pending operation unchanged.
- Drop `workspaces.ui_profile_image`, `ui_profile_first_name`,
  `ui_profile_last_name`, and `ui_profile_extra`, discarding their values.
