-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

CREATE TABLE events
(
    `property` String,
    `timestamp` DateTime,
    `osName` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'ChromeOS' = 6, 'Other' = 7),
    `osVersion` String,
    `browserName` String,
    `browserVersion` String,
    `deviceType` Enum8('desktop' = 1, 'tablet' = 2, 'mobile' = 3),
    `event` String,
    `language` String,
    `referrer` String,
    `target` String,
    `text` String,
    `title` String,
    `domain` String,
    `path` String,
    `queryString` String,
    `user` String,
    `country` String,
    `city` String
)
ENGINE = MergeTree
PRIMARY KEY (timestamp, event, property)
ORDER BY (timestamp, event, property)
SETTINGS index_granularity = 8192;
