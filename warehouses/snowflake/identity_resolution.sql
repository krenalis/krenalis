DROP TABLE IF EXISTS "_EDGES";
CREATE TABLE "_EDGES" (
    "I1" int,
    "I2" int
);

DROP TABLE IF EXISTS "_CLUSTERS_TO_MERGE";
CREATE TABLE "_CLUSTERS_TO_MERGE"("C1" int, "C2" int);

CREATE OR REPLACE PROCEDURE RESOLVE_IDENTITIES()
RETURNS BOOLEAN
LANGUAGE SQL
AS $$
BEGIN

    -- Determine the edges of the identities graph.
    TRUNCATE "_EDGES";
    EXECUTE IMMEDIATE 'INSERT INTO
        "_EDGES"
    SELECT
        "I1"."__PK__",
        "I2"."__PK__"
    FROM
        "_USER_IDENTITIES" "I1"
            CROSS JOIN
        "_USER_IDENTITIES" "I2"
    WHERE
        "I1"."__PK__" < "I2"."__PK__" AND (
            ("I1"."__CONNECTION__" = "I2"."__CONNECTION__"
                AND "I1"."__IDENTITY_ID__" = "I2"."__IDENTITY_ID__"
                AND "I1"."__IS_ANONYMOUS__" = "I2"."__IS_ANONYMOUS__"
            )
            OR {{ same_user }} -- This placeholder will be replaced by Meergo.
        )';

    -- Reset the user identity clusters, as they may have been modified by a
    -- previous execution of Identity Resolution. If they are not reset, user
    -- identities that were previously considered to belong to the same cluster
    -- will continue to be regarded as such, even though they should not be
    -- according to the parameters of the current execution of identity
    -- resolution.
    DROP TABLE IF EXISTS "NUMBERED_USERS";
    CREATE TEMPORARY TABLE "NUMBERED_USERS" AS
        SELECT 
            "__PK__",
            ROW_NUMBER() OVER (ORDER BY "__PK__") AS "CLUSTER_VALUE"
        FROM 
            "_USER_IDENTITIES";
    UPDATE "_USER_IDENTITIES" 
    SET "__CLUSTER__" = "NUMBERED_USERS"."CLUSTER_VALUE"
    FROM "NUMBERED_USERS"
    WHERE "_USER_IDENTITIES"."__PK__" = "NUMBERED_USERS"."__PK__";
    DROP TABLE IF EXISTS "NUMBERED_USERS";

    -- Do the clustering.
    DECLARE
        has_clusters_to_merge boolean;
    BEGIN 

    -- The idea here is to keep iterating as long as there are two
    -- identities that are the same user but have different clusters.
    LOOP
    
        -- Determine the clusters to merge.
        TRUNCATE "_CLUSTERS_TO_MERGE";
        INSERT INTO
            "_CLUSTERS_TO_MERGE"
        SELECT
            "I1"."__CLUSTER__" "C1",
            "I2"."__CLUSTER__" "C2"
        FROM
            "_EDGES"
            JOIN "_USER_IDENTITIES" "I1" ON "_EDGES"."I1" = "I1"."__PK__"
            JOIN "_USER_IDENTITIES" "I2" ON "_EDGES"."I2" = "I2"."__PK__"
        WHERE
            "I1"."__CLUSTER__" <> "I2"."__CLUSTER__";

        -- Stop iterating when there are no more clusters to merge.
        SELECT count(*) > 0 INTO :has_clusters_to_merge FROM "_CLUSTERS_TO_MERGE";
        IF (NOT has_clusters_to_merge) THEN
            BREAK;
        END IF;

        -- Make the "_CLUSTERS_TO_MERGE" table symmetric.
        -- TODO(Gianluca): is this necessary?
        INSERT INTO "_CLUSTERS_TO_MERGE"
            SELECT "C2", "C1"
            FROM "_CLUSTERS_TO_MERGE";
        
        -- Update the clusters of the user identities.
        UPDATE
            "_USER_IDENTITIES" "IDENTITIES_A"
        SET
            "__CLUSTER__" = least("IDENTITIES_A"."__CLUSTER__", "TARGET")
        FROM
            "_USER_IDENTITIES" "IDENTITIES_B"
            JOIN (
                SELECT
                    "C1" "SOURCE",
                    min("C2") "TARGET"
                FROM
                    "_CLUSTERS_TO_MERGE"
                GROUP BY
                    "SOURCE"
            ) "NEW_CLUSTERS" ON "NEW_CLUSTERS"."SOURCE" = "IDENTITIES_B"."__CLUSTER__"
        WHERE
            "IDENTITIES_A"."__PK__" = "IDENTITIES_B"."__PK__";

    END LOOP;
    END;

    -- This placeholder will be replaced by Meergo:
    {{ merge_identities_in_users }};

    -- Update associations between identities and users by updating the GID of
    -- the identities.
    UPDATE "_USER_IDENTITIES" AS "UI"
    SET "__GID__" = "U"."__ID__"
    FROM {{ new_users_name }} AS "U"
    WHERE ARRAY_CONTAINS("UI"."__PK__", "U"."__IDENTITIES__");

    -- Update associations between events and users by updating the user ID of
    -- the events.
    UPDATE "EVENTS" SET "MUID" = null;
    UPDATE "EVENTS" SET "MUID" = "_USER_IDENTITIES"."__GID__"
    FROM "_USER_IDENTITIES" WHERE
       "EVENTS"."CONNECTION_ID" = "_USER_IDENTITIES"."__CONNECTION__"
           AND
       (
           ("EVENTS"."USER_ID" <> '' AND "EVENTS"."USER_ID" = "_USER_IDENTITIES"."__IDENTITY_ID__")
               OR
           ("EVENTS"."USER_ID" = '' AND ARRAY_CONTAINS("EVENTS"."ANONYMOUS_ID"::variant, "_USER_IDENTITIES"."__ANONYMOUS_IDS__"))
       );

    RETURN true;
END
$$;
