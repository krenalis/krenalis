CREATE TABLE users
(
    `__id__` UUID,
)
ENGINE = MergeTree
ORDER BY `__id__`
SETTINGS index_granularity = 8192;