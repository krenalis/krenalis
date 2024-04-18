CREATE TABLE users (
    "_id" SERIAL,
    "__identity_keys__" int[],

    "email" varchar(300), -- TODO(Gianluca): see https://github.com/open2b/chichi/issues/628.

    PRIMARY KEY ("_id")
)