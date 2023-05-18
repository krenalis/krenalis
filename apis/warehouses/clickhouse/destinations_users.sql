
CREATE TABLE destinations_users (
    connection Int32,
    user String,
    property String
)
ENGINE = MergeTree()
PRIMARY KEY (connection, user)
ORDER BY (connection, user);
