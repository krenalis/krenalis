---
name: release-upgrade
description: Record in core/internal/initdb/RELEASE-UPGRADE.md how a change alters Krenalis's PostgreSQL database, so that the migration and the finalization can be written when the next release is prepared; also write them for a release, increment the schema version, and empty them afterwards.
---

# Release upgrade

Krenalis's PostgreSQL database has a schema version: an integer, unrelated to
the Krenalis version, recorded in the single row of the `schema_version` table
together with whether it has been finalized. A database without that table,
that is, a database of `v0.48.0`, is at schema version 1, finalized. The
constant `schemaVersion` in `core/internal/initdb/migrate.go` is the schema
version of the current code; it is incremented only by the releases that
change the database, so a release that does not change it needs no
migration.

A release that changes the database upgrades it in two steps, both permanent
flags of Krenalis:

1. `-migrate-db` (`core.MigrateDB`, `initdb.Migrate`) runs `migrate` in
   `migrate.go`. It migrates a finalized database of the previous schema
   version and records the new schema version, not finalized. It runs while
   the Krenalis instances of the previous release are still running, so
   `migrate` must not break them (see "Compatibility with the previous
   release").
2. `-finalize-db` (`core.FinalizeDB`, `initdb.Finalize`) runs `finalize` in
   `migrate.go`, once no instance of the previous release is running. It
   removes what only those instances needed and records the schema version
   as finalized. After it, the previous release can no longer run on the
   database.

Both run in a single transaction, under the advisory lock shared with the
initialization and with `lock_timeout` set to 5 seconds, and check the recorded
schema version, so they do nothing when run again. At startup,
`initdb.CheckSchemaVersion` lets Krenalis run only on its own schema version,
or on the next one not yet finalized. A database at schema version N must be
finalized before the release with schema version N+1 can migrate it, so the
releases that change the database are deployed one at a time, in order.

`migrate` and `finalize` hold the upgrade of a single release. Together with
their test and the schema of the previous release in
`core/internal/initdb/testdata`, they exist on `main` only while a release is
prepared; otherwise both do nothing. Between releases, changes record what the
upgrade must do in `core/internal/initdb/RELEASE-UPGRADE.md`, and `migrate` and
`finalize` are written from those notes when the next release is prepared. So a
commit of `main` that changes the database cannot be deployed on an existing
database.

## Base release

The notes and the upgrade start from the base release: the last release
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

## Compatibility with the previous release

During a rolling deployment, the instances of the previous release keep running
on a migrated database until they are replaced, and they run on it again after
a rollback. So:

- `migrate` must not break the previous release: add tables, nullable columns
  or columns with a default, and new data formats alongside the old ones.
- `finalize` holds what would break it: dropping columns, tables, triggers or
  old data formats, and adding constraints that the previous release could
  violate.
- The new release must work both on the migrated database and on the
  finalized one, because the finalization may come long after the deployment.

This rule is mandatory from the first release deployed in production. Before
that, a change may break the previous release, but its entry must say so.

## Recording a change

Record a change when it requires an existing database to be upgraded: changes
to `schema.sql` and to the other SQL files of `core/internal/initdb`, as well
as changes to the format of data stored in the database, such as JSON in a
column, that leave the schema unchanged.

1. If `migrate` or `finalize` in `migrate.go` does something, a release is
   being prepared, or the upgrade of the previous release was not removed:
   stop and ask the user how to proceed.
2. If `RELEASE-UPGRADE.md` does not exist, ask the user for the base release
   and create the file:

   ```markdown
   # Release upgrade

   Base release: `<tag>`

   What the migration and the finalization of the next release must do to
   upgrade a database of the base release. See
   `.agents/skills/release-upgrade/SKILL.md`.

   ## Migration

   ## Finalization
   ```

3. Add an entry, as a list item, under the section of the step that must do
   it, in the same change that alters the database. When a change needs both
   steps, as a removed column that the previous release still reads, add an
   entry to each.

An entry describes what the upgrade must do, including what the diff of
`schema.sql` cannot show:

- a rename, which the diff shows as a removal and an addition;
- the value that existing rows take in a new column, when it differs from the
  column's default;
- the conversion of existing data;
- the order that the steps require;
- that it breaks the previous release, when it does (see "Compatibility with
  the previous release").

For example:

```markdown
## Migration

- Add `connections.health`; existing rows take `'Healthy'`, while the
  default for new rows is `'Unknown'`.

## Finalization

- Drop `pipelines.legacy_filter`, which the current code no longer reads.
```

The notes describe the net change since the base release, not the history of
`main`. When a change alters or reverts what an entry describes, rewrite or
remove that entry instead of adding a new one, and delete the file when no
entries remain.

## Writing the upgrade for a release

1. Read the base release from `RELEASE-UPGRADE.md` and confirm it with the
   user, because it must still be the latest release in production. If the
   file does not exist, ask the user for the base release.
2. Compare the notes with `git diff <base> HEAD -- core/internal/initdb/`.
   Every difference must be covered by an entry, and every entry by the
   diff or by a change in the format of stored data; ask the user about any
   mismatch. If the database has not changed, write nothing and leave
   `schemaVersion` unchanged.
3. Check that `schemaVersion` is the schema version of the base release:
   `git show <base>:core/internal/initdb/migrate.go`, or 1 if the base release
   has no `schemaVersion`. Then increment it by one.
4. Write `migrate` from the entries under "Migration", and `finalize` from
   those under "Finalization", each within the transaction it receives. They
   need not be idempotent, since the recorded schema version prevents running
   them twice, but every statement is subject to the 5-second `lock_timeout`.
   Ask the user before writing a long backfill or a statement that cannot run
   in a transaction, such as `CREATE INDEX CONCURRENTLY`.
5. Save the schema of the base release in
   `core/internal/initdb/testdata/schema_<base>.sql` with
   `git show <base>:core/internal/initdb/schema.sql`, and write
   `migrate_release_test.go`, following
   `git show v0.48.0:core/internal/initdb/upgrade_test.go`:
   - Create the base database from that schema. If it has the `schema_version`
     table, record in it the schema version of the base release, finalized, as
     its initialization did.
   - Run `Migrate`, then `Finalize`; the resulting schema must match a newly
     initialized one, ignoring the order of the columns.
   - The test data must cover every step that transforms existing data.
   - Where the current code reads or writes what the upgrade changes, test it
     both after `Migrate` and after `Finalize`.
6. Delete `RELEASE-UPGRADE.md` in the same change.

## Removing the upgrade after a release

Once the release is published, empty the bodies of `migrate` and `finalize` so
that they return `nil`, and remove from `main` everything that only the upgrade
uses: `migrate_release_test.go`, the schema in `testdata`, and any helpers.
Keep `schemaVersion` at its new value, and keep `-migrate-db`,
`-finalize-db`, `core.MigrateDB`, `core.FinalizeDB`, `initdb.Migrate`,
`initdb.Finalize`, and `initdb.CheckSchemaVersion`.
