DROP TABLE IF EXISTS "edges";
CREATE TABLE edges (
    i1 int,
    i2 int,
    same_user boolean not null -- TODO(Gianluca): maybe this can be removed.
);

DROP TABLE IF EXISTS clusters_to_merge;
CREATE TABLE clusters_to_merge(c1 int, c2 int);

CREATE OR REPLACE PROCEDURE resolve_identities()
LANGUAGE sql
AS $$

    -- Determine the edges of the identities graph.
    TRUNCATE edges;
    INSERT INTO
        edges
    SELECT
        i1.__pk__,
        i2.__pk__,
        (
            (i1.__connection__ = i2.__connection__
                AND i1.__identity_id__ = i2.__identity_id__
                AND i1.__is_anonymous__ = i2.__is_anonymous__
            )
            OR {{ same_user }} -- This placeholder will be replaced by Meergo.
        ) as same_user
    FROM
        _user_identities i1
            CROSS JOIN
        _user_identities i2
    WHERE
        i1.__pk__ < i2.__pk__;
    
    -- Do the clustering.
    DO $clustering$
        DECLARE
            has_clusters_to_merge boolean;
        BEGIN 

        -- The idea here is to keep iterating as long as there are two
        -- identities that are the same user but have different clusters.
        LOOP
        
            -- Determine the clusters to merge.
            TRUNCATE clusters_to_merge;
            INSERT INTO
                clusters_to_merge
            SELECT
                i1.__cluster__ c1,
                i2.__cluster__ c2
            FROM
                edges
                JOIN _user_identities i1 ON edges.i1 = i1.__pk__
                JOIN _user_identities i2 ON edges.i2 = i2.__pk__
            WHERE
                edges.same_user
                AND i1.__cluster__ <> i2.__cluster__;

            -- Stop iterating when there are no more clusters to merge.
            SELECT count(*) > 0 INTO has_clusters_to_merge FROM clusters_to_merge;
            EXIT WHEN NOT has_clusters_to_merge;

            -- Make the "clusters_to_merge" table symmetric.
            -- TODO(Gianluca): is this necessary?
            INSERT INTO clusters_to_merge
                SELECT c2, c1
                FROM clusters_to_merge;
            
            -- Update the clusters of the user identities.
            UPDATE
                _user_identities identities_a
            SET
                __cluster__ = least(identities_a.__cluster__, target)
            FROM
                _user_identities identities_b
                JOIN (
                    SELECT
                        c1 source,
                        min(c2) target
                    FROM
                        clusters_to_merge
                    GROUP BY
                        source
                ) new_clusters ON new_clusters.source = identities_b.__cluster__
            WHERE
                identities_a.__pk__ = identities_b.__pk__;

        END LOOP;

    END $clustering$;

    -- This placeholder will be replaced by Meergo:
    {{ merge_identities_in_users }};

    -- Update associations between identities and users by updating the GID of
    -- the identities.
    UPDATE "_user_identities" SET "__gid__" = (
        SELECT "__id__"
        FROM {{ new_users_name }}
        WHERE
            "_user_identities"."__pk__" = ANY ({{ new_users_name }}."__identities__")
        LIMIT 1
    )
    FROM {{ new_users_name }}
    WHERE
        "_user_identities"."__pk__" = ANY ({{ new_users_name }}."__identities__");

    -- Update associations between events and users by updating the user ID of
    -- the events.
    UPDATE "events" SET "user" = null;
    UPDATE "events" SET "user" = "_user_identities"."__gid__"
    FROM "_user_identities" WHERE
        "events"."source" = "_user_identities"."__connection__"
            AND
        (
            ("events"."user_id" <> '' AND "events"."user_id" = "_user_identities"."__identity_id__")
                OR
            ("events"."user_id" = '' AND "events"."anonymous_id" = ANY ("_user_identities"."__anonymous_ids__"))
        );

$$;
