-- Copyright 2026 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

-- Add the queries to update the PostgreSQL warehouse.
--
-- NEVER EMPTY THIS FILE unless absolutely necessary, otherwise it becomes
-- difficult to perform updates.

-- Rename warehouse tables from meergo_* to krenalis_*.
ALTER TABLE IF EXISTS "meergo_identities"              RENAME TO "krenalis_identities";
ALTER TABLE IF EXISTS "meergo_destination_profiles"    RENAME TO "krenalis_destination_profiles";
ALTER TABLE IF EXISTS "meergo_events"                  RENAME TO "krenalis_events";
ALTER TABLE IF EXISTS "meergo_profile_schema_versions" RENAME TO "krenalis_profile_schema_versions";
ALTER TABLE IF EXISTS "meergo_system_operations"       RENAME TO "krenalis_system_operations";

-- Rename all versioned profiles tables (e.g. meergo_profiles_0, meergo_profiles_1, ...).
DO $$
DECLARE
    tbl text;
    new_name text;
BEGIN
    FOR tbl IN
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name LIKE 'meergo\_profiles\_%'
          AND table_type = 'BASE TABLE'
        ORDER BY table_name
    LOOP
        new_name := 'krenalis' || substring(tbl FROM length('meergo') + 1);
        EXECUTE format('ALTER TABLE %I RENAME TO %I', tbl, new_name);
    END LOOP;
END;
$$;

-- Rename identity resolution working tables if they exist (created only when
-- identity resolution has been run at least once).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'meergo_graph_edges'
    ) THEN
        ALTER TABLE "meergo_graph_edges" RENAME TO "krenalis_graph_edges";
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'meergo_graph_merge_clusters'
    ) THEN
        ALTER TABLE "meergo_graph_merge_clusters" RENAME TO "krenalis_graph_merge_clusters";
    END IF;
END;
$$;

-- Recreate the events view pointing to the renamed table.
-- (PostgreSQL resolves view references by OID so the view would already work
-- after the rename above, but we recreate it explicitly for clarity).
CREATE OR REPLACE VIEW "events" AS SELECT * FROM "krenalis_events";

-- Drop the resolve_identities procedure so it is cleanly recreated with the
-- new table names the next time identity resolution runs.
DROP PROCEDURE IF EXISTS resolve_identities();

-- === Rename "mpid" to "kpid" in all relevant tables and views. ===

-- Rename column in krenalis_events.
ALTER TABLE IF EXISTS "krenalis_events" RENAME COLUMN "mpid" TO "kpid";

-- Rename column in krenalis_identities.
ALTER TABLE IF EXISTS "krenalis_identities" RENAME COLUMN "_mpid" TO "_kpid";

-- Rename column in all versioned profiles tables (e.g. krenalis_profiles_0,
-- krenalis_profiles_1, ...).
DO $$
DECLARE
    tbl text;
BEGIN
    FOR tbl IN
        SELECT table_name
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name LIKE 'krenalis\_profiles\_%'
          AND table_type = 'BASE TABLE'
        ORDER BY table_name
    LOOP
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = current_schema()
              AND table_name = tbl
              AND column_name = '_mpid'
        ) THEN
            EXECUTE format('ALTER TABLE %I RENAME COLUMN "_mpid" TO "_kpid"', tbl);
        END IF;
    END LOOP;
END;
$$;
