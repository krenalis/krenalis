CREATE TABLE users
(
    `__id__` Int32,
)
ENGINE = MergeTree
PRIMARY KEY `__id__`
ORDER BY `__id__`
SETTINGS index_granularity = 8192;