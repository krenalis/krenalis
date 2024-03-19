CREATE TABLE users_identities (
    "_identity_id"   serial,
    "_connection"    int NOT NULL DEFAULT 0,
    "_external_id"   text NOT NULL DEFAULT '',
    "_anonymous_ids" text[],
    "_updated_at"    timestamp NOT NULL,
    "_gid"           int NOT NULL DEFAULT 0,
    "_business_id"   varchar(40) NOT NULL DEFAULT '',
    "__cluster__"       serial,

    "email" varchar(300), -- TODO(Gianluca): TODO(Gianluca): keep this until https://github.com/open2b/chichi/issues/572 is resolved.

    PRIMARY KEY ("_identity_id")
);