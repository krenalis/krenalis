/* 
  Add property and user fields to events table.
  If events must be kept create a customer/property in the sql db and assign all
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
