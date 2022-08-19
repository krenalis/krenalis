CREATE TABLE `events` (
  `id` int NOT NULL AUTO_INCREMENT,
  `timestamp` datetime NOT NULL,
  `language` varchar(10) DEFAULT NULL,
  `browser` varchar(100) DEFAULT NULL,
  `url` varchar(250) DEFAULT NULL,
  `referrer` varchar(250) DEFAULT NULL,
  `target` varchar(250) DEFAULT NULL,
  `event` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
