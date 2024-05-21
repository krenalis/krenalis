CREATE TABLE _users (
    "__id__" uuid,
    "__identity_keys__" int [],
    
    "email" varchar(300)
);

CREATE VIEW "users" AS SELECT "__id__", "email" FROM "_users";