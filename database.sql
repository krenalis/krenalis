CREATE TABLE `events` (
  `timestamp` datetime,
  `browser` String,
  `event` String,
  `language` String,
  `referrer` String,
  `session` String,
  `target` String,
  `text` String,
  `title` String,
  `url` String,
  PRIMARY KEY (
    `timestamp`,
    `browser`,
    `event`,
    `language`,
    `referrer`,
    `session`,
    `target`,
    `text`,
    `title`,
    `url`
  )
) ENGINE = MergeTree;
