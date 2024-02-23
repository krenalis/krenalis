CREATE TABLE groups_identities (
    id              SERIAL,
    "creation_time" timestamp NOT NULL DEFAULT now(),
    "timestamp"     timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);