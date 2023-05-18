
CREATE TABLE destinations_users (
    connection integer NOT NULL,
    "user" varchar(45) NOT NULL DEFAULT '',
    property varchar(500) NOT NULL,
    PRIMARY KEY (connection, "user")
);
