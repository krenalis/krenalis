
CREATE TABLE IF NOT EXISTS _destinations_users (
    __action__ integer NOT NULL,
    __user__ text NOT NULL DEFAULT '',
    __property__ text NOT NULL,
    PRIMARY KEY (__action__, __user__)
);
