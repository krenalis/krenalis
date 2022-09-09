CREATE TABLE `customers` (
  `id` INT NOT NULL AUTO_INCREMENT ,
  `name` VARCHAR(45) NOT NULL DEFAULT '',
  `email` VARCHAR(120) NOT NULL DEFAULT '',
  `password` VARCHAR(60) CHARACTER SET ascii NOT NULL DEFAULT '',
  `internalIPs` VARCHAR(160) CHARACTER SET ascii NOT NULL DEFAULT '',
  PRIMARY KEY (`id`));

CREATE TABLE `properties` (
  `id` CHAR(10) NOT NULL,
  `customer` INT NOT NULL,
  PRIMARY KEY (`id`));

CREATE TABLE `domains` (
  `property` CHAR(10) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  PRIMARY KEY (`property`, `name`));

/* default customer and properties */
INSERT INTO `customers` (`name`,`email`,`password`) VALUES ('ACME inc','acme@open2b.com','$2a$10$iMuokZyvwdAQOJJmJvG83eSGGWTV3DOjI2DRU6SjuLEuK.vknUJVC'); /* Password: foopass2 */
INSERT INTO `properties` VALUES ('1234567890',1);

CREATE TABLE `smart_events` (
  `property` CHAR(10) NOT NULL,
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL DEFAULT '',
  `event` VARCHAR(50) NOT NULL DEFAULT '',
  `pages` TEXT NOT NULL DEFAULT '',
  `buttons` TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
);

INSERT INTO `smart_events` VALUES ('ABCDEFGHIJ',1,'View Nissan Car','pageview','[{\"Field\":\"url\",\"Operator\":\"StartsWith\",\"Value\":\"/cars/Nissan/\"},{\"Field\":\"url\",\"Operator\":\"StartsWith\",\"Value\":\"/auto/Nissan/\"}]','null'),('1234567890',38,'View Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','null'),('1234567890',39,'Configure a Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','[{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configure your car\",\"Domain\":\"english.example.com\"},{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configura la tua auto\",\"Domain\":\"italian.example.com\"}]'),('1234567890',40,'Click on Login Button','click','null','[{\"Field\":\"text\",\"Operator\":\"Contains\",\"Value\":\"Log in\"}]');

CREATE TABLE `devices` (
  `property` char(10) CHARACTER SET ascii DEFAULT '',
  `id` char(28) CHARACTER SET ascii NOT NULL,
  `user` int unsigned DEFAULT NULL,
  PRIMARY KEY (`property`, `id`)
);

CREATE TABLE `users` (
  `property` char(10) NOT NULL,
  `id` int unsigned NOT NULL,
  `device` char(28) CHARACTER SET ascii DEFAULT NULL,
  PRIMARY KEY (`property`,`id`)
);
