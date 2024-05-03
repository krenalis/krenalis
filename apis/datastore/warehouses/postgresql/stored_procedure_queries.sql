CREATE OR REPLACE FUNCTION matching_func(VARIADIC identifiers text[])
RETURNS boolean
AS $$
    DECLARE
        i INT;
    BEGIN
        FOR i IN 1..array_length(identifiers, 1) BY 2 LOOP
            IF identifiers[i] IS NOT NULL AND identifiers[i+1] IS NOT NULL THEN -- TODO(Gianluca) is "IS NOT NULL" correct? See the issue https://github.com/open2b/chichi/issues/657.
                RETURN identifiers[i] = identifiers[i+1]; -- TODO(Gianluca): is the "=" comparison correct? See the issue https://github.com/open2b/chichi/issues/657.
            END IF;
        END loop;
        RETURN false;
    END;
$$ LANGUAGE plpgsql;

DROP TABLE IF EXISTS "matchings";
CREATE TABLE matchings (
    i1 int,
    i2 int,
    match boolean not null
);

DROP TABLE IF EXISTS clusters_to_merge;
CREATE TABLE clusters_to_merge(c1 int, c2 int);

CREATE OR REPLACE PROCEDURE resolve_sync_users()
LANGUAGE sql
AS $$

    -- Determine the matchings.
    TRUNCATE matchings;
    INSERT INTO
        matchings(i1, i2, match)
    SELECT
        i1.__identity_key__,
        i2.__identity_key__,
        
        -- This placeholder will be replaced by Chichi:
        {{ matching_expr }} as match
    FROM
        users_identities i1
            CROSS JOIN
        users_identities i2
    WHERE
        i1.__identity_key__ < i2.__identity_key__;
    
    -- Do the clustering.
    DO $clustering$
        DECLARE
            has_clusters_to_merge boolean;
        BEGIN 
    
        LOOP
        
            -- Determine the clusters to merge.
            TRUNCATE clusters_to_merge;
            INSERT INTO
                clusters_to_merge(c1, c2)
            SELECT
                i1.__cluster__ c1,
                i2.__cluster__ c2
            FROM
                matchings m
                JOIN users_identities i1 ON m.i1 = i1.__identity_key__
                JOIN users_identities i2 ON m.i2 = i2.__identity_key__
            WHERE
                m.match
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
                users_identities identities_a
            SET
                __cluster__ = least(identities_a.__cluster__, target)
            FROM
                users_identities identities_b
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
                identities_a.__identity_key__ = identities_b.__identity_key__;

        END LOOP;

    END $clustering$;

    -- This placeholder will be replaced by Chichi:
    {{ users_sync_queries }};

    -- Update the GID of the user identities.
    UPDATE "users_identities" SET "__gid__" = (
        SELECT "__id__"
        FROM "users"
        WHERE
            "users_identities"."__identity_key__" = ANY ("users"."__identity_keys__")
        LIMIT 1
    )
    FROM "users"
    WHERE
        "users_identities"."__identity_key__" = ANY ("users"."__identity_keys__");

    -- Update the GID of the events.
    UPDATE "events" SET "gid" = 0;
    UPDATE "events" SET "gid" = "users_identities"."__gid__"
    FROM "users_identities" WHERE
        "events"."source" = "users_identities"."__connection__"
            AND
        (
            ("events"."user_id" <> '' AND "events"."user_id" = "users_identities"."__identity_id__")
                OR
            ("events"."user_id" = '' AND "events"."anonymous_id" = ANY ("users_identities"."__anonymous_ids__"))
        );

$$;
