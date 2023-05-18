
CREATE TABLE destinations_users (
    connection integer NOT NULL,
    "user" text NOT NULL DEFAULT '',
    property text NOT NULL,
    PRIMARY KEY (connection, "user")
);
