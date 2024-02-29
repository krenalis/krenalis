CREATE TABLE users (
    "_id" SERIAL,
    "__identity_ids__" int[],

    "email" varchar(300), -- TODO(Gianluca): keep this until https://github.com/open2b/chichi/issues/572 is resolved.

    PRIMARY KEY ("_id")
)