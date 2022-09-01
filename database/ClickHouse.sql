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
    `url`
    `user`
  )
) ENGINE = MergeTree;
