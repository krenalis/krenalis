
CREATE TABLE connections_users (
    connection Int32,
    user String,
    data String,
    timestamps String,
    golden_record Int32
)
ENGINE = MergeTree()
PRIMARY KEY (connection, user)
ORDER BY (connection, user);
