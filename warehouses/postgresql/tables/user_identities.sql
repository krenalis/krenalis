CREATE TABLE IF NOT EXISTS _user_identities (
    "__pk__" serial,
    "__action__" integer NOT NULL,
    "__is_anonymous__" boolean NOT NULL DEFAULT FALSE,
    "__identity_id__" text NOT NULL,
    "__connection__" integer NOT NULL,
    "__anonymous_ids__" text [],
    "__last_change_time__" timestamp NOT NULL,
    "__execution__" integer,
    "__gid__" uuid,
    "__cluster__" serial,
    "email" character varying(300),
    PRIMARY KEY ("__pk__")
);
