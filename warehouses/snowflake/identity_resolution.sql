-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

DROP TABLE IF EXISTS "MEERGO_GRAPH_EDGES";
CREATE TABLE "MEERGO_GRAPH_EDGES" (
    "I1" int,
    "I2" int
);

DROP TABLE IF EXISTS "MEERGO_GRAPH_MERGE_CLUSTERS";
CREATE TABLE "MEERGO_GRAPH_MERGE_CLUSTERS"("C1" int, "C2" int);

CREATE OR REPLACE PROCEDURE RESOLVE_IDENTITIES()
RETURNS BOOLEAN
LANGUAGE SQL
AS $$
BEGIN

    -- Determine the edges of the identities graph.
    TRUNCATE "MEERGO_GRAPH_EDGES";
    EXECUTE IMMEDIATE 'INSERT INTO
        "MEERGO_GRAPH_EDGES"
    SELECT
        "I1"."_pk",
        "I2"."_pk"
    FROM
        "_IDENTITIES" "I1"
            CROSS JOIN
        "_IDENTITIES" "I2"
    WHERE
        "I1"."_pk" < "I2"."_pk" AND (
            ("I1"."_connection" = "I2"."_connection"
                AND "I1"."_identity_id" = "I2"."_identity_id"
                AND "I1"."_is_anonymous" = "I2"."_is_anonymous"
            )
            OR {{ same_profile }} -- This placeholder will be replaced by Meergo.
        )';

    -- Reset the identity clusters, as they may have been modified by a previous
    -- execution of Identity Resolution. If they are not reset, identities that
    -- were previously considered to belong to the same cluster will continue to
    -- be regarded as such, even though they should not be according to the
    -- parameters of the current execution of identity resolution.
    DROP TABLE IF EXISTS "NUMBERED_PROFILES";
    CREATE TEMPORARY TABLE "NUMBERED_PROFILES" AS
        SELECT 
            "_pk",
            ROW_NUMBER() OVER (ORDER BY "_pk") AS "CLUSTER_VALUE"
        FROM 
            "_IDENTITIES";
    UPDATE "_IDENTITIES"
    SET "_cluster" = "NUMBERED_PROFILES"."CLUSTER_VALUE"
    FROM "NUMBERED_PROFILES"
    WHERE "_IDENTITIES"."_pk" = "NUMBERED_PROFILES"."_pk";
    DROP TABLE IF EXISTS "NUMBERED_PROFILES";

    -- Do the clustering.
    DECLARE
        has_clusters_to_merge boolean;
    BEGIN 

    -- The idea here is to keep iterating as long as there are two
    -- identities that are the same profile but have different clusters.
    LOOP
    
        -- Determine the clusters to merge.
        TRUNCATE "MEERGO_GRAPH_MERGE_CLUSTERS";
        INSERT INTO
            "MEERGO_GRAPH_MERGE_CLUSTERS"
        SELECT
            "I1"."_cluster" "C1",
            "I2"."_cluster" "C2"
        FROM
            "MEERGO_GRAPH_EDGES"
            JOIN "_IDENTITIES" "I1" ON "MEERGO_GRAPH_EDGES"."I1" = "I1"."_pk"
            JOIN "_IDENTITIES" "I2" ON "MEERGO_GRAPH_EDGES"."I2" = "I2"."_pk"
        WHERE
            "I1"."_cluster" <> "I2"."_cluster";

        -- Stop iterating when there are no more clusters to merge.
        SELECT count(*) > 0 INTO :has_clusters_to_merge FROM "MEERGO_GRAPH_MERGE_CLUSTERS";
        IF (NOT has_clusters_to_merge) THEN
            BREAK;
        END IF;

        -- Make the "MEERGO_GRAPH_MERGE_CLUSTERS" table symmetric.
        -- TODO(Gianluca): is this necessary?
        INSERT INTO "MEERGO_GRAPH_MERGE_CLUSTERS"
            SELECT "C2", "C1"
            FROM "MEERGO_GRAPH_MERGE_CLUSTERS";
        
        -- Update the clusters of the identities.
        UPDATE
            "_IDENTITIES" "IDENTITIES_A"
        SET
            "_cluster" = least("IDENTITIES_A"."_cluster", "TARGET")
        FROM
            "_IDENTITIES" "IDENTITIES_B"
            JOIN (
                SELECT
                    "C1" "SOURCE",
                    min("C2") "TARGET"
                FROM
                    "MEERGO_GRAPH_MERGE_CLUSTERS"
                GROUP BY
                    "SOURCE"
            ) "NEW_CLUSTERS" ON "NEW_CLUSTERS"."SOURCE" = "IDENTITIES_B"."_cluster"
        WHERE
            "IDENTITIES_A"."_pk" = "IDENTITIES_B"."_pk";

    END LOOP;
    END;

    -- This placeholder will be replaced by Meergo:
    {{ merge_identities_in_profiles }};

    -- Update associations between identities and profiles by updating the MPID
    -- of the identities.
    UPDATE "_IDENTITIES" AS "I"
    SET "_mpid" = "U"."_mpid"
    FROM {{ new_profiles_name }} AS "U"
    WHERE ARRAY_CONTAINS("I"."_pk", "U"."_identities");

    -- Update associations between events and profiles by updating the MPID of
    -- the events.
    UPDATE "MEERGO_EVENTS" SET "MPID" = null;
    UPDATE "MEERGO_EVENTS" SET "MPID" = "_IDENTITIES"."_mpid"
    FROM "_IDENTITIES" WHERE
       "MEERGO_EVENTS"."CONNECTION_ID" = "_IDENTITIES"."_connection"
           AND
       (
           ("MEERGO_EVENTS"."USER_ID" <> '' AND "MEERGO_EVENTS"."USER_ID" = "_IDENTITIES"."_identity_id")
               OR
           ("MEERGO_EVENTS"."USER_ID" = '' AND ARRAY_CONTAINS("MEERGO_EVENTS"."ANONYMOUS_ID"::variant, "_IDENTITIES"."_anonymous_ids"))
       );

    RETURN true;
END
$$;
