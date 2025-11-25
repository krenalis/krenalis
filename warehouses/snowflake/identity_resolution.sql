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
        "I1"."__PK__",
        "I2"."__PK__"
    FROM
        "_IDENTITIES" "I1"
            CROSS JOIN
        "_IDENTITIES" "I2"
    WHERE
        "I1"."__PK__" < "I2"."__PK__" AND (
            ("I1"."__CONNECTION__" = "I2"."__CONNECTION__"
                AND "I1"."__IDENTITY_ID__" = "I2"."__IDENTITY_ID__"
                AND "I1"."__IS_ANONYMOUS__" = "I2"."__IS_ANONYMOUS__"
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
            "__PK__",
            ROW_NUMBER() OVER (ORDER BY "__PK__") AS "CLUSTER_VALUE"
        FROM 
            "_IDENTITIES";
    UPDATE "_IDENTITIES"
    SET "__CLUSTER__" = "NUMBERED_PROFILES"."CLUSTER_VALUE"
    FROM "NUMBERED_PROFILES"
    WHERE "_IDENTITIES"."__PK__" = "NUMBERED_PROFILES"."__PK__";
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
            "I1"."__CLUSTER__" "C1",
            "I2"."__CLUSTER__" "C2"
        FROM
            "MEERGO_GRAPH_EDGES"
            JOIN "_IDENTITIES" "I1" ON "MEERGO_GRAPH_EDGES"."I1" = "I1"."__PK__"
            JOIN "_IDENTITIES" "I2" ON "MEERGO_GRAPH_EDGES"."I2" = "I2"."__PK__"
        WHERE
            "I1"."__CLUSTER__" <> "I2"."__CLUSTER__";

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
            "__CLUSTER__" = least("IDENTITIES_A"."__CLUSTER__", "TARGET")
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
            ) "NEW_CLUSTERS" ON "NEW_CLUSTERS"."SOURCE" = "IDENTITIES_B"."__CLUSTER__"
        WHERE
            "IDENTITIES_A"."__PK__" = "IDENTITIES_B"."__PK__";

    END LOOP;
    END;

    -- This placeholder will be replaced by Meergo:
    {{ merge_identities_in_profiles }};

    -- Update associations between identities and profiles by updating the MPID
    -- of the identities.
    UPDATE "_IDENTITIES" AS "I"
    SET "__mpid__" = "U"."__MPID__"
    FROM {{ new_profiles_name }} AS "U"
    WHERE ARRAY_CONTAINS("I"."__PK__", "U"."__IDENTITIES__");

    -- Update associations between events and profiles by updating the MPID of
    -- the events.
    UPDATE "MEERGO_EVENTS" SET "MPID" = null;
    UPDATE "MEERGO_EVENTS" SET "MPID" = "_IDENTITIES"."__mpid__"
    FROM "_IDENTITIES" WHERE
       "MEERGO_EVENTS"."CONNECTION_ID" = "_IDENTITIES"."__CONNECTION__"
           AND
       (
           ("MEERGO_EVENTS"."USER_ID" <> '' AND "MEERGO_EVENTS"."USER_ID" = "_IDENTITIES"."__IDENTITY_ID__")
               OR
           ("MEERGO_EVENTS"."USER_ID" = '' AND ARRAY_CONTAINS("MEERGO_EVENTS"."ANONYMOUS_ID"::variant, "_IDENTITIES"."__ANONYMOUS_IDS__"))
       );

    RETURN true;
END
$$;
