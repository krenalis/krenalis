/* 
  Add property and user fields to events table.
  If events must be kept create a account/property in the sql db and assign all
  the events to the property using the following SQL.
  Or just drop the old table and create the new one.
*/
RENAME TABLE `events` TO `events_old`;
CREATE TABLE `events` (
  `property` String,
  `timestamp` datetime,
  `browser` String,
  `event` String,
  `language` String,
  `referrer` String,
  `target` String,
  `text` String,
  `title` String,
  `url` String,
  `user` String,
  PRIMARY KEY (
    `property`,
    `timestamp`,
    `browser`,
    `event`,
    `language`,
    `referrer`,
    `target`,
    `text`,
    `title`,
    `url`,
    `user`
  )
) ENGINE = MergeTree;

INSERT INTO `events`
    SELECT '1234567890', `timestamp`, `browser`, `event`, `language`, `referrer`, `target`, `text`, `title`, `url`, `session`
    FROM `events_old`;

ALTER TABLE `events` ADD COLUMN `country` String;
ALTER TABLE `events` ADD COLUMN `city` String;

DROP TABLE `events`;
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

ALTER TABLE `events` ADD COLUMN `domain` String AFTER `url`;
ALTER TABLE `events` ADD COLUMN `path` String AFTER `domain`;
ALTER TABLE `events` ADD COLUMN `queryString` String AFTER `path`;
ALTER TABLE `events` DROP COLUMN `url`;

ALTER TABLE `events` UPDATE `user` = '0' WHERE TRUE;
ALTER TABLE `events` MODIFY COLUMN `user` UInt32;

ALTER TABLE `events` ADD COLUMN `date` Date AFTER `property`;
ALTER TABLE `events` UPDATE `date` = toDate(`timestamp`) WHERE TRUE;

DROP TABLE `events`;
CREATE TABLE events
(
    `property` String,
    `date` Date,
    `timestamp` DateTime,
    `osName` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'ChromeOS' = 6, 'Other' = 7),
    `osVersion` String,
    `browserName` String,
    `browserVersion` String,
    `deviceType` Enum8('desktop' = 1, 'tablet' = 2, 'mobile' = 3),
    `event` Enum8('pageview' = 1, 'click' = 2),
    `language` String,
    `referrer` String,
    `target` String,
    `text` String,
    `title` String,
    `domain` String,
    `path` String,
    `queryString` String,
    `user` UInt32,
    `country` String,
    `city` String
)
ENGINE = MergeTree
PRIMARY KEY (timestamp, event, property)
ORDER BY (timestamp, event, property)
SETTINGS index_granularity = 8192;

DROP TABLE `events`;
CREATE TABLE events
(
    `property` String,
    `date` Date,
    `timestamp` DateTime,
    `osName` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'ChromeOS' = 6, 'Other' = 7),
    `osVersion` String,
    `browserName` String,
    `browserVersion` String,
    `deviceType` Enum8('desktop' = 1, 'tablet' = 2, 'mobile' = 3),
    `event` Enum8('pageview' = 1, 'click' = 2),
    `language` String,
    `referrer` String,
    `target` String,
    `text` String,
    `title` String,
    `domain` String,
    `path` String,
    `queryString` String,
    `user` UInt32,
    `country` String,
    `city` String
)
ENGINE = MergeTree
PRIMARY KEY (`property`, `date`, `user`, `timestamp`)
SETTINGS index_granularity = 8192;

DROP TABLE `events`;
CREATE TABLE events
(
    `property` UInt32,
    `date` Date,
    `timestamp` DateTime,
    `osName` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'ChromeOS' = 6, 'Other' = 7),
    `osVersion` String,
    `browserName` String,
    `browserVersion` String,
    `deviceType` Enum8('desktop' = 1, 'tablet' = 2, 'mobile' = 3),
    `event` Enum8('pageview' = 1, 'click' = 2),
    `language` String,
    `referrer` String,
    `target` String,
    `text` String,
    `title` String,
    `domain` String,
    `path` String,
    `queryString` String,
    `user` UInt32,
    `country` String,
    `city` String
)
ENGINE = MergeTree
PRIMARY KEY (`property`, `date`, `user`, `timestamp`)
SETTINGS index_granularity = 8192;

ALTER TABLE `events` MODIFY COLUMN `language` FixedString(2);
ALTER TABLE `events` MODIFY COLUMN `country` FixedString(2);

ALTER TABLE `events` MODIFY COLUMN `osName` Enum8('Other' = 0, 'Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'ChromeOS' = 6);

DROP TABLE `events`;
CREATE TABLE events
(
    `property` UInt32,
    `date` Date,
    `timestamp` DateTime,
    `osName` Enum8('Other' = 0, 'Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'ChromeOS' = 6),
    `osVersion` String,
    `browser` Enum8('Other' = 0, 'Chrome' = 1, 'Safari' = 2, 'Edge' = 3, 'Firefox' = 4, 'Samsung Internet' = 5, 'Opera' = 6),
    `browserOther` String,
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

ALTER TABLE `events` MODIFY COLUMN `osName` Enum8('Other' = 0, 'Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'Chrome OS' = 6);

DROP TABLE `events`;
CREATE TABLE events
(
    `source` Int32,
    `date` Date,
    `timestamp` DateTime,
    `osName` Enum8('Other' = 0, 'Android' = 1, 'Windows' = 2, 'iOS' = 3, 'MacOS' = 4, 'Linux' = 5, 'Chrome OS' = 6),
    `osVersion` String,
    `browser` Enum8('Other' = 0, 'Chrome' = 1, 'Safari' = 2, 'Edge' = 3, 'Firefox' = 4, 'Samsung Internet' = 5, 'Opera' = 6),
    `browserOther` String,
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
PRIMARY KEY (`source`, `date`, `user`, `timestamp`)
SETTINGS index_granularity = 8192;

DROP TABLE `events`;
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

DROP TABLE `events`;
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
    `event` UInt8,
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
