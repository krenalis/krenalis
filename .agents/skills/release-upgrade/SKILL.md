---
name: release-upgrade
description: Record in core/internal/initdb/RELEASE-UPGRADE.md how a change alters Krenalis's PostgreSQL database, so that the migration can be written when the next release is prepared; also write the migration for a release and remove it afterwards.
---

# Release upgrade

Krenalis migrates the PostgreSQL database of a release with the `-migrate-db`
flag, which calls `core.MigrateDB` and `initdb.Migrate`. These are permanent:
`Migrate` takes the advisory lock and the transaction, and calls `migrate` in
`core/internal/initdb/migrate.go`, which holds the migration of the release.
The migration, with its test and the schema of the previous release in
`core/internal/initdb/testdata`, exists on `main` only while a release is
prepared; otherwise `migrate` does nothing. Between releases, changes record
what the migration must do in `core/internal/initdb/RELEASE-UPGRADE.md`, and
the migration is written from those notes when the next release is prepared.

## Base release

The notes and the migration upgrade from the base release: the last release
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

1. If `migrate` in `migrate.go` does something, a release is being prepared,
   or the migration of the previous release was not removed: stop and ask the
   user how to proceed.
2. If `RELEASE-UPGRADE.md` does not exist, ask the user for the base release
   and create the file:

   ```markdown
   # Release upgrade

   Base release: `<tag>`

   What the migration of the next release must do to upgrade a database of the
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

## Writing the migration for a release

1. Read the base release from `RELEASE-UPGRADE.md` and confirm it with the
   user, because it must still be the latest release in production. If the
   file does not exist, ask the user for the base release.
2. Compare the notes with `git diff <base> HEAD -- core/internal/initdb/`.
   Every difference must be covered by an entry, and every entry by the
   diff or by a change in the format of stored data; ask the user about any
   mismatch. If the database has not changed, write no migration.
3. Write `migrate` in `migrate.go` so that it upgrades a database of the base
   release to the current schema, within the transaction it receives,
   idempotently: it must also leave unchanged a database already migrated or
   newly initialized.
4. Save the schema of the base release in
   `core/internal/initdb/testdata/schema_<base>.sql` with
   `git show <base>:core/internal/initdb/schema.sql`, and write
   `migrate_release_test.go` as `upgrade_test.go` in #2653: the migrated
   schema must match a newly initialized one, the migration must be
   idempotent, and the test data must cover every step that transforms
   existing data.
5. Delete `RELEASE-UPGRADE.md` in the same change.

## Removing the migration after a release

Once the release is published, empty the body of `migrate` so that it returns
`nil`, and remove from `main` everything that only the migration uses:
`migrate_release_test.go`, the schema in `testdata`, and any helpers. Keep
`-migrate-db`, `core.MigrateDB`, and `initdb.Migrate`.
