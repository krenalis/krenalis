CREATE TABLE _users_identities (
    "__identity_key__" serial,
    "__connection__" int NOT NULL DEFAULT 0,
    "__identity_id__" text NOT NULL DEFAULT '',
    "__displayed_property__" varchar(40) NOT NULL DEFAULT '',
    "__anonymous_ids__" text[],
    "__last_change_time__" timestamp NOT NULL,
    "__gid__" int NOT NULL DEFAULT 0,
    
    "__cluster__" serial,

    "email" varchar(300),

    PRIMARY KEY ("__identity_key__")
);

CREATE VIEW "users_identities" AS
SELECT
    "__identity_key__",
    "__connection__",
    "__identity_id__",
    "__displayed_property__",
    "__anonymous_ids__",
    "__last_change_time__",
    "__gid__",
    "email"
FROM "_users_identities";