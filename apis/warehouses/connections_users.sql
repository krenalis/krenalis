
CREATE TABLE connections_users (
    connection integer NOT NULL,
    "user" varchar(45) NOT NULL DEFAULT '',
    data varchar(655359) NOT NULL,
    timestamps varchar(655359) NOT NULL DEFAULT '',
    golden_record integer NOT NULL DEFAULT 0,
    PRIMARY KEY (connection, "user")
);
