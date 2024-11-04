CREATE TABLE IF NOT EXISTS "_user_identities" (
    "__pk__" INT AUTOINCREMENT START 0 INCREMENT 1 ORDER,
    "__action__" INT NOT NULL,
    "__is_anonymous__" BOOLEAN NOT NULL DEFAULT FALSE,
    "__identity_id__" VARCHAR NOT NULL,
    "__connection__" INT NOT NULL,
    "__anonymous_ids__" ARRAY,
    "__last_change_time__" TIMESTAMP NOT NULL,
    "__execution__" INT,
    "__gid__" VARCHAR,
    "__cluster__" INT AUTOINCREMENT START 0 INCREMENT 1 ORDER,
    "email" VARCHAR(300),
    PRIMARY KEY ("__pk__")
);
