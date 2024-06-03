CREATE TABLE _user_identities (
    "__pk__" serial,
    "__connection__" int NOT NULL,
    "__identity_id__" text NOT NULL,
    "__is_anonymous__" boolean NOT NULL DEFAULT FALSE,
    "__anonymous_ids__" text [],
    "__displayed_property__" varchar(40) NOT NULL DEFAULT '',
    "__last_change_time__" timestamp NOT NULL,
    "__gid__" uuid,
    "__cluster__" serial,
    "email" varchar(300),
    PRIMARY KEY ("__pk__")
);

CREATE VIEW "user_identities" AS
SELECT
    "__pk__",
    "__connection__",
    "__identity_id__",
    "__is_anonymous__",
    "__anonymous_ids__",
    "__displayed_property__",
    "__last_change_time__",
    "__gid__",
    "email"
FROM
    "_user_identities";