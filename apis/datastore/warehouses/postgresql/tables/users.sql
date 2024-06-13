CREATE TABLE _users (
    "__id__" uuid,
    "__identities__" int [],
    "__last_change_time__" timestamp NOT NULL,
    
    "email" varchar(300)
);

CREATE VIEW "users" AS SELECT "__id__", "email" FROM "_users";