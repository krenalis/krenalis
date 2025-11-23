-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

DROP TABLE IF EXISTS "_edges";
CREATE TABLE _edges (
    "i1" int,
    "i2" int
);

DROP TABLE IF EXISTS "_clusters_to_merge";
CREATE TABLE _clusters_to_merge("c1" int, "c2" int);

CREATE OR REPLACE PROCEDURE resolve_identities()
LANGUAGE sql
AS $$

    -- Determine the edges of the identities graph.
    TRUNCATE "_edges";
    INSERT INTO
        "_edges"
    SELECT
        "i1"."__pk__",
        "i2"."__pk__"
    FROM
        "_identities" "i1"
            CROSS JOIN
        "_identities" "i2"
    WHERE
        "i1"."__pk__" < "i2"."__pk__" AND (
            ("i1"."__connection__" = "i2"."__connection__"
                AND "i1"."__identity_id__" = "i2"."__identity_id__"
                AND "i1"."__is_anonymous__" = "i2"."__is_anonymous__"
            )
            OR {{ same_profile }} -- This placeholder will be replaced by Meergo.
        );

    -- Reset the identity clusters, as they may have been modified by a previous
    -- execution of Identity Resolution. If they are not reset, identities that
    -- were previously considered to belong to the same cluster will continue to
    -- be regarded as such, even though they should not be according to the
    -- parameters of the current execution of identity resolution.
    WITH "numbered_profiles" AS (
        SELECT 
            "__pk__",
            ROW_NUMBER() OVER (ORDER BY "__pk__") AS "cluster_value"
        FROM 
            "_identities"
    )
    UPDATE "_identities"
    SET "__cluster__" = "numbered_profiles"."cluster_value"
    FROM "numbered_profiles"
    WHERE "_identities"."__pk__" = "numbered_profiles"."__pk__";

    -- Do the clustering.
    DO $clustering$
        DECLARE
            has_clusters_to_merge boolean;
        BEGIN 

        -- The idea here is to keep iterating as long as there are two
        -- identities that are the same profile but have different clusters.
        LOOP
        
            -- Determine the clusters to merge.
            TRUNCATE "_clusters_to_merge";
            INSERT INTO
                "_clusters_to_merge"
            SELECT
                "i1"."__cluster__" "c1",
                "i2"."__cluster__" "c2"
            FROM
                "_edges"
                JOIN "_identities" "i1" ON "_edges"."i1" = "i1"."__pk__"
                JOIN "_identities" "i2" ON "_edges"."i2" = "i2"."__pk__"
            WHERE
                "i1"."__cluster__" <> "i2"."__cluster__";

            -- Stop iterating when there are no more clusters to merge.
            SELECT count(*) > 0 INTO has_clusters_to_merge FROM "_clusters_to_merge";
            EXIT WHEN NOT has_clusters_to_merge;

            -- Make the "_clusters_to_merge" table symmetric.
            -- TODO(Gianluca): is this necessary?
            INSERT INTO "_clusters_to_merge"
                SELECT "c2", "c1"
                FROM "_clusters_to_merge";
            
            -- Update the clusters of the identities.
            UPDATE
                "_identities" "identities_a"
            SET
                "__cluster__" = least("identities_a"."__cluster__", "target")
            FROM
                "_identities" "identities_b"
                JOIN (
                    SELECT
                        "c1" "source",
                        min("c2") "target"
                    FROM
                        "_clusters_to_merge"
                    GROUP BY
                        "source"
                ) "new_clusters" ON "new_clusters"."source" = "identities_b"."__cluster__"
            WHERE
                "identities_a"."__pk__" = "identities_b"."__pk__";

        END LOOP;

    END $clustering$;

    -- This placeholder will be replaced by Meergo:
    {{ merge_identities_in_profiles }};

    -- Update associations between identities and profiles by updating the MPID
    -- of the identities.
    UPDATE "_identities" AS "ui"
    SET "__mpid__" = "u"."__mpid__"
    FROM {{ new_profiles_name }} AS "u"
    WHERE "ui"."__pk__" = ANY ("u"."__identities__");

    -- Update associations between events and profiles by updating the MPID of
    -- the events.
    UPDATE "events" SET "mpid" = null;
    UPDATE "events" SET "mpid" = "_identities"."__mpid__"
    FROM "_identities" WHERE
        "events"."connection_id" = "_identities"."__connection__"
            AND
        (
            ("events"."user_id" <> '' AND "events"."user_id" = "_identities"."__identity_id__")
                OR
            ("events"."user_id" = '' AND "events"."anonymous_id" = ANY ("_identities"."__anonymous_ids__"))
        );

$$;
