CREATE TABLE users_identities (
    "_identity_id" serial,
    "_connection" int NOT NULL DEFAULT 0,
    "_external_id" text NOT NULL DEFAULT '',
    "_displayed_property" varchar(40) NOT NULL DEFAULT '',
    "_anonymous_ids" text[],
    "_updated_at" timestamp NOT NULL,
    "_gid" int NOT NULL DEFAULT 0,
    
    "__cluster__" serial,

    "email" varchar(300), -- TODO(Gianluca): see https://github.com/open2b/chichi/issues/628.

    PRIMARY KEY ("_identity_id")
);