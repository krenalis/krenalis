CREATE TABLE _users (
    "__id__" SERIAL,
    "__identity_keys__" int[],

    "email" varchar(300),

    PRIMARY KEY ("__id__")
);

CREATE VIEW "users" AS SELECT "__id__", "email" FROM "_users";