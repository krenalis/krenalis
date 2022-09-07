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
