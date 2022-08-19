CREATE TABLE `events` (
  `id` int NOT NULL AUTO_INCREMENT,
  `browser` varchar(100) DEFAULT NULL,
  `event` varchar(45) DEFAULT NULL,
  `language` varchar(10) DEFAULT NULL,
  `referrer` varchar(250) DEFAULT NULL,
  `session` varchar(300) DEFAULT NULL,
  `target` varchar(250) DEFAULT NULL,
  `text` varchar(300) DEFAULT NULL,
  `timestamp` datetime NOT NULL,
  `title` varchar(300) DEFAULT NULL,
  `url` varchar(250) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
