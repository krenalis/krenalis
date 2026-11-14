---
name: release-upgrade
description: Record in core/internal/initdb/RELEASE-UPGRADE.md how a change alters Krenalis's PostgreSQL database, so that the updater can be written when the next release is prepared; also write the updater for a release and remove it afterwards.
---

# Release upgrade

Krenalis upgrades the PostgreSQL database of a release with an updater:
`core/internal/initdb/upgrade.go`, its test with the schema of the previous
release in `core/internal/initdb/testdata`, `core.UpgradeDB`, and the
`-upgrade-db` flag. The updater exists on `main` only while a release is
prepared. Between releases, changes record what the updater must do in
`core/internal/initdb/RELEASE-UPGRADE.md`, and the updater is written from
those notes when the next release is prepared.

## Base release

The notes and the updater upgrade from the base release: the last release
published and used in production, which is the one with the **Latest** badge
on GitHub. Pre-releases and aborted releases are not base releases, even when
their tag is more recent, so the base release cannot be determined from the
local tags.

Never guess the base release. When it is needed and the notes do not state it,
ask the user for its tag, then verify that the tag exists and is an ancestor of
`HEAD`:

```bash
git rev-parse --verify --quiet "refs/tags/<tag>"
git merge-base --is-ancestor <tag> HEAD
```

If a check fails, stop and report it without choosing another tag.

## Recording a change

Record a change when it requires an existing database to be upgraded: changes
to `schema.sql` and to the other SQL files of `core/internal/initdb`, as well
as changes to the format of data stored in the database, such as JSON in a
column, that leave the schema unchanged.

1. If `upgrade.go` exists, a release is being prepared, or the updater of the
   previous release was not removed: stop and ask the user how to proceed.
2. If `RELEASE-UPGRADE.md` does not exist, ask the user for the base release
   and create the file:

   ```markdown
   # Release upgrade

   Base release: `<tag>`

   What the updater of the next release must do to upgrade a database of the
   base release. See `.agents/skills/release-upgrade/SKILL.md`.
   ```

3. Add an entry, as a list item, in the same change that alters the database.

An entry describes what the upgrade must do, including what the diff of
`schema.sql` cannot show:

- a rename, which the diff shows as a removal and an addition;
- the value that existing rows take in a new column, when it differs from the
  column's default;
- the conversion of existing data;
- the order that the steps require.

For example:

```markdown
- Rename `pipelines.filter` to `pipelines.filters`, keeping its values.
- Add `connections.health`; existing rows take `'Healthy'`, while the
  default for new rows is `'Unknown'`.
```

The notes describe the net change since the base release, not the history of
`main`. When a change alters or reverts what an entry describes, rewrite or
remove that entry instead of adding a new one, and delete the file when no
entries remain.

## Writing the updater for a release

1. Read the base release from `RELEASE-UPGRADE.md` and confirm it with the
   user, because it must still be the latest release in production. If the
   file does not exist, ask the user for the base release.
2. Compare the notes with `git diff <base> HEAD -- core/internal/initdb/`.
   Every difference must be covered by an entry, and every entry by the
   diff or by a change in the format of stored data; ask the user about any
   mismatch. If the database has not changed, write no updater.
3. Restore `core.UpgradeDB` and the `-upgrade-db` flag, as in #2653, which
   restored what #2612 removed.
4. Write `Upgrade` in `upgrade.go` so that it upgrades a database of the base
   release to the current schema, in a single transaction, idempotently.
5. Save the schema of the base release in
   `core/internal/initdb/testdata/schema_<base>.sql` with
   `git show <base>:core/internal/initdb/schema.sql`, and write
   `upgrade_test.go` as in #2653: the upgraded schema must match a newly
   initialized one, the upgrade must be idempotent, and the test data must
   cover every step that transforms existing data.
6. Delete `RELEASE-UPGRADE.md` in the same change.

## Removing the updater after a release

Once the release is published, remove from `main` the updater and everything
that only it uses: `upgrade.go`, `upgrade_test.go`, the schema in
`testdata`, `core.UpgradeDB`, and the `-upgrade-db` flag, as in #2612.
