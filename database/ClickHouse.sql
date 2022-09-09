-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

CREATE TABLE events
(
    `property` UInt32,
    `date` Date,
    `timestamp` DateTime,
    `osName` Enum8('Other' = 0, 'Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'ChromeOS' = 6),
    `osVersion` String,
    `browserName` String,
    `browserVersion` String,
    `deviceType` Enum8('desktop' = 1, 'tablet' = 2, 'mobile' = 3),
    `event` Enum8('pageview' = 1, 'click' = 2),
    `language` FixedString(2),
    `referrer` String,
    `target` String,
    `text` String,
    `title` String,
    `domain` String,
    `path` String,
    `queryString` String,
    `user` UInt32,
    `country` FixedString(2),
    `city` String
)
ENGINE = MergeTree
PRIMARY KEY (`property`, `date`, `user`, `timestamp`)
SETTINGS index_granularity = 8192;
