CREATE TABLE users_identities (
    "__identity_key__" serial,
    "__connection__" int NOT NULL DEFAULT 0,
    "__identity_id__" text NOT NULL DEFAULT '',
    "__displayed_property__" varchar(40) NOT NULL DEFAULT '',
    "__anonymous_ids__" text[],
    "__last_change_time__" timestamp NOT NULL,
    "__gid__" int NOT NULL DEFAULT 0,
    
    "__cluster__" serial,

    "email" varchar(300), -- TODO(Gianluca): see https://github.com/open2b/chichi/issues/628.

    PRIMARY KEY ("__identity_key__")
);