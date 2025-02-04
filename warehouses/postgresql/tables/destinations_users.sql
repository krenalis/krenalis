CREATE TABLE IF NOT EXISTS _destinations_users (
    __action__ integer NOT NULL,
    __external_id__ text NOT NULL DEFAULT '',
    __out_matching_value__ text NOT NULL,
    PRIMARY KEY (__action__, __external_id__)
);
