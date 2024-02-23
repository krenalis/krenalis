CREATE TABLE groups
(
    `id` Int32,
)
ENGINE = MergeTree
PRIMARY KEY id
ORDER BY id
SETTINGS index_granularity = 8192;
