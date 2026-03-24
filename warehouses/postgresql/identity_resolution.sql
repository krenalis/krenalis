-- Copyright 2026 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

DROP TABLE IF EXISTS "krenalis_graph_edges";
CREATE TABLE krenalis_graph_edges (
    "i1" int,
    "i2" int
);

DROP TABLE IF EXISTS "krenalis_graph_merge_clusters";
CREATE TABLE krenalis_graph_merge_clusters("c1" int, "c2" int);

CREATE OR REPLACE PROCEDURE resolve_identities()
LANGUAGE sql
AS $$

    -- Determine the edges of the identities graph.
    TRUNCATE "krenalis_graph_edges";
    INSERT INTO
        "krenalis_graph_edges"
    SELECT
        "i1"."_pk",
        "i2"."_pk"
    FROM
        "krenalis_identities" "i1"
            CROSS JOIN
        "krenalis_identities" "i2"
    WHERE
        "i1"."_pk" < "i2"."_pk" AND (
            ("i1"."_connection" = "i2"."_connection"
                AND "i1"."_identity_id" = "i2"."_identity_id"
                AND "i1"."_is_anonymous" = "i2"."_is_anonymous"
            )
            OR {{ same_profile }} -- This placeholder will be replaced by Krenalis.
        );

    -- Reset the identity clusters, as they may have been modified by a previous
    -- execution of Identity Resolution. If they are not reset, identities that
    -- were previously considered to belong to the same cluster will continue to
    -- be regarded as such, even though they should not be according to the
    -- parameters of the current execution of identity resolution.
    WITH "numbered_profiles" AS (
        SELECT 
            "_pk",
            ROW_NUMBER() OVER (ORDER BY "_pk") AS "cluster_value"
        FROM 
            "krenalis_identities"
    )
    UPDATE "krenalis_identities"
    SET "_cluster" = "numbered_profiles"."cluster_value"
    FROM "numbered_profiles"
    WHERE "krenalis_identities"."_pk" = "numbered_profiles"."_pk";

    -- Do the clustering.
    DO $clustering$
        DECLARE
            has_clusters_to_merge boolean;
        BEGIN 

        -- The idea here is to keep iterating as long as there are two
        -- identities that are the same profile but have different clusters.
        LOOP
        
            -- Determine the clusters to merge.
            TRUNCATE "krenalis_graph_merge_clusters";
            INSERT INTO
                "krenalis_graph_merge_clusters"
            SELECT
                "i1"."_cluster" "c1",
                "i2"."_cluster" "c2"
            FROM
                "krenalis_graph_edges"
                JOIN "krenalis_identities" "i1" ON "krenalis_graph_edges"."i1" = "i1"."_pk"
                JOIN "krenalis_identities" "i2" ON "krenalis_graph_edges"."i2" = "i2"."_pk"
            WHERE
                "i1"."_cluster" <> "i2"."_cluster";

            -- Stop iterating when there are no more clusters to merge.
            SELECT count(*) > 0 INTO has_clusters_to_merge FROM "krenalis_graph_merge_clusters";
            EXIT WHEN NOT has_clusters_to_merge;

            -- Make the "krenalis_graph_merge_clusters" table symmetric.
            -- TODO(Gianluca): is this necessary?
            INSERT INTO "krenalis_graph_merge_clusters"
                SELECT "c2", "c1"
                FROM "krenalis_graph_merge_clusters";
            
            -- Update the clusters of the identities.
            UPDATE
                "krenalis_identities" "identities_a"
            SET
                "_cluster" = least("identities_a"."_cluster", "target")
            FROM
                "krenalis_identities" "identities_b"
                JOIN (
                    SELECT
                        "c1" "source",
                        min("c2") "target"
                    FROM
                        "krenalis_graph_merge_clusters"
                    GROUP BY
                        "source"
                ) "new_clusters" ON "new_clusters"."source" = "identities_b"."_cluster"
            WHERE
                "identities_a"."_pk" = "identities_b"."_pk";

        END LOOP;

    END $clustering$;

    -- This placeholder will be replaced by Krenalis:
    {{ merge_identities_in_profiles }};

    -- Update associations between identities and profiles by updating the MPID
    -- of the identities.
    UPDATE "krenalis_identities" AS "i"
    SET "_mpid" = "u"."_mpid"
    FROM {{ new_profiles_name }} AS "u"
    WHERE "i"."_pk" = ANY ("u"."_identities");

    -- Update associations between events and profiles by updating the MPID of
    -- the events.
    UPDATE "krenalis_events" SET "mpid" = null;
    UPDATE "krenalis_events" SET "mpid" = "krenalis_identities"."_mpid"
    FROM "krenalis_identities" WHERE
        "krenalis_events"."connection_id" = "krenalis_identities"."_connection"
            AND
        (
            ("krenalis_events"."user_id" <> '' AND "krenalis_events"."user_id" = "krenalis_identities"."_identity_id")
                OR
            ("krenalis_events"."user_id" = '' AND "krenalis_events"."anonymous_id" = ANY ("krenalis_identities"."_anonymous_ids"))
        );

$$;
