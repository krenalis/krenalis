
CREATE TABLE destinations_users (
    action Int32,
    user String,
    property String
)
ENGINE = MergeTree()
PRIMARY KEY (action, user)
ORDER BY (action, user);
