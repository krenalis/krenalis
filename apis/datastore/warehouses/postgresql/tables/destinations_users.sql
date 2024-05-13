
CREATE TABLE _destinations_users (
    action integer NOT NULL,
    "user" text NOT NULL DEFAULT '',
    property text NOT NULL,
    PRIMARY KEY (action, "user")
);
