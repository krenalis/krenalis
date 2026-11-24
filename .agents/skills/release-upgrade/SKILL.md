---
name: release-upgrade
description: Record in core/internal/initdb/RELEASE-UPGRADE.md how a change alters Krenalis's PostgreSQL database, so that the migration and the finalization can be written when the next release is prepared; also write them for a release, increment the schema version, and empty them afterwards.
---

# Release upgrade

Krenalis's PostgreSQL database has a schema version: an integer, unrelated to the Krenalis version, recorded in the single row of the `schema_version` table together with whether it has been finalized. A database without that table, that is, a database of `v0.48.0`, is at schema version 1, finalized. The constant `schemaVersion` in `core/internal/initdb/upgrade.go` is the schema version of the current code; it is incremented only by the releases that change the database, so a release that does not change it needs no migration.

A release that changes the database upgrades it in two steps, both permanent flags of Krenalis:

1. `-migrate-db` (`core.MigrateDB`, `initdb.Migrate`) runs `migrate` in `upgrade.go`. It migrates a finalized database of the previous schema version and records the new schema version, not finalized. It runs while the Krenalis instances of the previous release are still running, so `migrate` must not break them (see "Compatibility with the previous release").
2. `-finalize-db` (`core.FinalizeDB`, `initdb.Finalize`) runs `finalize` in `upgrade.go`, once no instance of the previous release is running: it does not check it, and otherwise the behavior is undefined and may be destructive. It removes what only those instances needed, such as the old columns and the triggers that keep them in sync, adds the constraints that they could violate, and records the schema version as finalized. After it, the previous release can no longer run on the database.

Both run in a single transaction, under the advisory lock shared with the initialization and with `lock_timeout` set to 5 seconds, and check the recorded schema version, so they do nothing when run again. At startup, `initdb.CheckSchemaVersion` lets Krenalis run only on its own schema version, or on the next one not yet finalized. The check compares schema versions, not releases, so it also lets an older release at the same schema version as the previous release run; only the previous release, however, is required to keep working (see "Compatibility with the previous release"). A database at schema version N must be finalized before the release with schema version N+1 can migrate it, so the releases that change the database are deployed one at a time, in order.

`migrate` and `finalize` hold the upgrade of a single release. Together with their helpers in `upgrade_release.go`, their test in `upgrade_release_test.go`, and the SQL files of the base release in `core/internal/initdb/testdata`, they exist on `main` only while a release is prepared; otherwise both do nothing. Between releases, changes record what the upgrade must do in `core/internal/initdb/RELEASE-UPGRADE.md`, and `migrate` and `finalize` are written from those notes when the next release is prepared. So a commit of `main` that changes the database cannot be deployed on an existing database.

## Base release

The notes and the upgrade start from the base release: the last release published and used in production, which is the one with the **Latest** badge on GitHub. Pre-releases and aborted releases are not base releases, even when their tag is more recent, so the base release cannot be determined from the local tags.

Never guess the base release. When it is needed and the notes do not state it, ask the user for its tag, then verify that the tag exists and is an ancestor of `HEAD`:

```bash
git rev-parse --verify --quiet "refs/tags/<tag>"
git merge-base --is-ancestor <tag> HEAD
```

If a check fails, stop and report it without choosing another tag.

## Compatibility with the previous release

During a rolling deployment, the instances of the previous release keep running on a migrated database until they are replaced, and they run on it again after a rollback to the previous release. Meanwhile they keep reading and writing the database, and they exchange notifications with the instances of the new release.

This is a requirement: the upgrade to a release must never break the previous one. For the database, a single release is enough, if it follows the rules below.

The only exception is the upgrade from `v0.48.0`, which has neither `-migrate-db` nor `initdb.CheckSchemaVersion`: since Krenalis is not in production yet, it may break `v0.48.0`, so neither a rolling deployment nor a rollback to `v0.48.0` is supported. Remove this paragraph after that release.

### The database

Until the finalization, the database holds both the representation that the previous release uses and the one that the new release uses, kept in sync:

- `migrate` adds what the new release needs without changing or removing what the previous release uses: tables, nullable columns or columns with a default, populated from the existing data. When the two releases must see the same data in different formats, `migrate` puts the new format in a new column or table, and adds triggers that keep the two in sync in both directions: what the previous release writes in the old representation is copied to the new one, and vice versa. Each release reads and writes only its own representation. The triggers also fill the new columns of the rows that the previous release inserts, when their default is not the right value.
- `finalize` removes what only the previous release used: the triggers, the old columns and tables. It also adds the constraints that the previous release could violate. It converts nothing, because the triggers have kept the new representation up to date.
- The new release reads and writes only the new representation, and must work both on the migrated database and on the finalized one, because the finalization may come long after the deployment.

A trigger cannot show the same column in two formats to two readers, so never change a format in place, by converting the values of a column that the previous release reads: put the new format in a new column. If the new format carries information that the old one cannot express, the trigger from the new to the old representation cannot be written: ask the user how to proceed.

### Notifications and NATS

The two releases also exchange notifications and NATS messages, which no trigger can translate. How to keep them compatible is still open, and a policy must be agreed in https://github.com/krenalis/krenalis/issues/2796. Until then, ask the user before changing a notification or the format of the events published on NATS.

## Recording a change

Record a change when it requires an existing database to be upgraded: changes to `schema.sql` and to the other SQL files of `core/internal/initdb`, as well as changes to the format of data stored in the database, such as JSON in a column, that leave the schema unchanged.

1. If `migrate` or `finalize` in `upgrade.go` does something, a release is being prepared, or the upgrade of the previous release was not removed: stop and ask the user how to proceed.
2. If `RELEASE-UPGRADE.md` does not exist, ask the user for the base release and create the file:

   ```markdown
   # Release upgrade

   Base release: `<tag>`

   What the migration and the finalization of the next release must do to upgrade a database of the base release. See `.agents/skills/release-upgrade/SKILL.md`.

   ## Migration

   ## Finalization
   ```

3. Add an entry, as a list item, under the section of the step that must do it, in the same change that alters the database. When a change needs both steps, as a removed column that the previous release still reads, add an entry to each.

An entry describes what the upgrade must do, including what the diff of `schema.sql` cannot show:

- a rename, which the diff shows as a removal and an addition;
- the value that existing rows take in a new column, when it differs from the column's default;
- the conversion of existing data;
- the order that the steps require;
- how it keeps the previous release working, when it is not obvious (see "Compatibility with the previous release").

For example:

```markdown
## Migration

- Add `connections.health`; existing rows take `'Healthy'`, while the default for new rows is `'Unknown'`.

## Finalization

- Drop `pipelines.legacy_filter`, which the current code no longer reads.
```

The notes describe the net change since the base release, not the history of `main`. When a change alters or reverts what an entry describes, rewrite or remove that entry instead of adding a new one, and delete the file when no entries remain.

## Writing the upgrade for a release

1. Read the base release from `RELEASE-UPGRADE.md` and confirm it with the user, because it must still be the latest release in production. If the file does not exist, ask the user for the base release.
2. Compare the notes with `git diff <base> HEAD -- core/internal/initdb/`. Every difference must be covered by an entry, and every entry by the diff or by a change in the format of stored data; ask the user about any mismatch. If the database has not changed, write nothing and leave `schemaVersion` unchanged.
3. Check that `schemaVersion` is the schema version of the base release: `git show <base>:core/internal/initdb/upgrade.go`, or 1 if the base release has no `schemaVersion`. Then increment it by one.
4. Write `migrate` from the entries under "Migration", and `finalize` from those under "Finalization", each within the transaction it receives, with the helpers they need in `upgrade_release.go`:
   - Use SQL for DDL, triggers, and backfills that a statement can express. Use Go for conversions that are recursive, as of JSON values, or that depend on other data; in Go, work on the decoded JSON, such as `map[string]any` decoded with `UseNumber`, not on the types of the current code, which already describe the new format and may reject the old one.
   - When `migrate` adds triggers that convert between two formats, the backfill must produce the same values as the triggers. Prefer writing the conversion once, in a SQL function that both the triggers and the backfill use.
   - They need not be idempotent, since the recorded schema version prevents running them twice.
   - Every statement is subject to the 5-second `lock_timeout`, which keeps them from waiting for the running instances. The opposite is not prevented: the rows they update stay locked until the commit, and the running instances that write them wait. So update only the rows that change, filtering them in SQL or comparing the values in Go.
   - Tables of configuration, such as `workspaces` and `pipelines`, are small and can be updated in the transaction. Ask the user before updating a table that grows with the data, such as one of events, identities or metrics, and before a statement that cannot run in a transaction, such as `CREATE INDEX CONCURRENTLY`.
5. Save in `core/internal/initdb/testdata/<base>/`, with `git show <base>:core/internal/initdb/<file>`, every SQL file that the `initialize` function of the base release executes: `schema.sql` and the others, such as `rate_limiter_leases.sql`. `newBaseTestDatabase`, in `upgrade_helpers_test.go`, creates the base database as `initialize` does now: `schema.sql` one statement at a time, then the other files whole, in lexical order. If the `initialize` of the base release differs, adapt the helper first.
6. Write `upgrade_release_test.go` with the helpers of `upgrade_helpers_test.go`, which do not depend on the release:
   - Create the base database with `newBaseTestDatabase` and insert test data that covers every step that transforms existing data, and rows that must not change.
   - Run `Migrate`, then `Finalize`. After each, check the recorded schema version, and that `schemaSnapshot` matches the one of a database created with `newInitializedTestDatabase`.
   - After each, check the converted values with `assertQueryJSON`, and load the state with `loadTestState`, which runs the code that Krenalis runs at startup on the upgraded data, such as the parsing of schemas and filters. Connections and pipelines load only if their connectors are imported by the test.
   - If `migrate` adds triggers, test them between `Migrate` and `Finalize`: write rows in the old representation, as the previous release does, and check the new one, and vice versa.
7. Delete `RELEASE-UPGRADE.md` in the same change.

## Removing the upgrade after a release

Once the release is published, empty the bodies of `migrate` and `finalize` so that they return `nil`, and delete `upgrade_release.go`, `upgrade_release_test.go`, and `testdata/<base>`. Keep `schemaVersion` at its new value, and keep `upgrade_helpers_test.go`, `-migrate-db`, `-finalize-db`, `core.MigrateDB`, `core.FinalizeDB`, `initdb.Migrate`, `initdb.Finalize`, and `initdb.CheckSchemaVersion`.
