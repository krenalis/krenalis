CREATE TABLE _user_identities (
    "__pk__" serial,
    "__action__" integer NOT NULL,
    "__is_anonymous__" boolean NOT NULL DEFAULT FALSE,
    "__identity_id__" text NOT NULL,
    "__connection__" integer NOT NULL,
    "__anonymous_ids__" text [],
    "__last_change_time__" timestamp NOT NULL,
    "__gid__" uuid,
    "__cluster__" serial,
    "email" varchar(300),
    PRIMARY KEY ("__pk__")
);

CREATE VIEW "user_identities" AS
SELECT
    "__pk__",
    "__action__",
    "__is_anonymous__",
    "__identity_id__",
    "__connection__",
    "__anonymous_ids__",
    "__last_change_time__",
    "__gid__",
    "email"
FROM
    "_user_identities";