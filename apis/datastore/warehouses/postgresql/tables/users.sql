CREATE TABLE _users_0 (
    "__id__" uuid,
    "__identities__" int [],
    "__last_change_time__" timestamp NOT NULL,
    "email" varchar(300)
);

CREATE VIEW "users" AS
SELECT
    "__id__",
    "__last_change_time__",
    "email"
FROM
    "_users_0";