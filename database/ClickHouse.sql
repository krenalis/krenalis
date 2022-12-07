-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

CREATE TABLE events
(
    `source` Int32,
    `date` Date,
    `timestamp` DateTime,
    `os_name` Enum8('Other' = 0, 'Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'Chrome OS' = 6),
    `os_version` String,
    `browser` Enum8('Other' = 0, 'Chrome' = 1, 'Safari' = 2, 'Edge' = 3, 'Firefox' = 4, 'Samsung Internet' = 5, 'Opera' = 6),
    `browser_other` String,
    `browser_version` String,
    `device_type` Enum8('desktop' = 1, 'tablet' = 2, 'mobile' = 3),
    `event` Enum8('pageview' = 1, 'click' = 2),
    `language` FixedString(2),
    `referrer` String,
    `target` String,
    `text` String,
    `title` String,
    `domain` String,
    `path` String,
    `query_string` String,
    `user` UInt32,
    `country` FixedString(2),
    `city` String
)
ENGINE = MergeTree
PRIMARY KEY (`source`, `date`, `user`, `timestamp`)
SETTINGS index_granularity = 8192;
