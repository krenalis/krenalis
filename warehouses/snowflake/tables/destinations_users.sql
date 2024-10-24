CREATE TABLE IF NOT EXISTS "_destinations_users" (
    "__action__" INT NOT NULL,
    "__user__" VARCHAR NOT NULL DEFAULT '',
    "__property__" VARCHAR NOT NULL,
    PRIMARY KEY ("__action__", "__user__")
);
