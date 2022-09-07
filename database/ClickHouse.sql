CREATE TABLE `events`
(
    `property` String,
    `timestamp` DateTime,
    `osName` Enum('Android', 'Windows', 'iOS', 'MacOS', 'Linux', 'ChromeOS', 'Other'),
    `osVersion` String,
    `browserName` String,
    `browserVersion` String,
    `deviceType` Enum('desktop', 'tablet', 'mobile'),
    `event` String,
    `language` String,
    `referrer` String,
    `target` String,
    `text` String,
    `title` String,
    `url` String,
    `user` String,
    `country` String,
    `city` String
)
ENGINE = MergeTree
PRIMARY KEY (timestamp, event, property)
ORDER BY (timestamp, event, property)
SETTINGS index_granularity = 8192;
