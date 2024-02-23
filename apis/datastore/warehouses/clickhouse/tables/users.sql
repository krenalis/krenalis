CREATE TABLE users
(
    `_id` Int32,
)
ENGINE = MergeTree
PRIMARY KEY `_id`
ORDER BY `_id`
SETTINGS index_granularity = 8192;