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
            i1.__identity_id__,
            i2.__identity_id__,
            
            -- This placeholder will be replaced by Chichi:
            {{ matching_expr }} as match
        FROM
            users_identities i1
                CROSS JOIN
            users_identities i2
        WHERE
            i1.__identity_id__ < i2.__identity_id__;
        
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
                    JOIN users_identities i1 ON m.i1 = i1.__identity_id__
                    JOIN users_identities i2 ON m.i2 = i2.__identity_id__
                WHERE
                    m.match
                    AND i1.__cluster__ <> i2.__cluster__;

                -- Stop iterating when there are no more clusters to merge.
                SELECT count(*) > 0 INTO has_clusters_to_merge FROM clusters_to_merge;
                EXIT WHEN NOT has_clusters_to_merge;

                -- Make the "clusters_to_merge" table simmetric.
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
                    identities_a.__identity_id__ = identities_b.__identity_id__;

            END LOOP;

        END $clustering$;

        -- This placeholder will be replaced by Chichi:
        {{ users_sync_queries }}

    $$;
