
ALTER TABLE `customers` CHANGE COLUMN `password` `password` VARCHAR(60) NOT NULL DEFAULT '';
ALTER TABLE `properties` CHANGE COLUMN `code` `id` CHAR(10) NOT NULL;
RENAME TABLE `properties_domains` TO `domains`;
ALTER TABLE `domains` CHANGE COLUMN `domain` `name` VARCHAR(255) NOT NULL;
UPDATE `customers` SET `password` = '$2a$10$rlHZ0RUyMMeMQxDlAK6S2.sL7Y8Z8IafRsagpdYyadZrKpLJWH94K' WHERE `password` = 'foopass';
ALTER TABLE `customers` CHANGE COLUMN `password` `password` CHAR(60) CHARACTER SET ascii NOT NULL DEFAULT '';
ALTER TABLE `customers` ADD COLUMN `internalIPs` VARCHAR(160) CHARACTER SET ascii NOT NULL DEFAULT '';
UPDATE `customers` SET `password` = '$2a$10$iMuokZyvwdAQOJJmJvG83eSGGWTV3DOjI2DRU6SjuLEuK.vknUJVC' WHERE `password` = '$2a$10$rlHZ0RUyMMeMQxDlAK6S2.sL7Y8Z8IafRsagpdYyadZrKpLJWH94K';
CREATE TABLE `smart_events` (
  `property` CHAR(10) NOT NULL,
  `id` INT(10) NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL DEFAULT '',
  `event` VARCHAR(50) NOT NULL DEFAULT '',
  `pages` TEXT NOT NULL DEFAULT '',
  `buttons` TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
);
INSERT INTO `smart_events` VALUES ('ABCDEFGHIJ',1,'View Nissan Car','view','[{\"Field\":\"url\",\"Operator\":\"StartsWith\",\"Value\":\"/cars/Nissan/\"},{\"Field\":\"url\",\"Operator\":\"StartsWith\",\"Value\":\"/auto/Nissan/\"}]','null'),('1234567890',38,'View Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','null'),('1234567890',39,'Configure a Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','[{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configure your car\",\"Domain\":\"english.example.com\"},{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configura la tua auto\",\"Domain\":\"italian.example.com\"}]'),('1234567890',40,'Click on Login Button','click','null','[{\"Field\":\"text\",\"Operator\":\"Contains\",\"Value\":\"Log in\"}]');

CREATE TABLE `devices` (
  `property` char(10) CHARACTER SET ascii DEFAULT '',
  `id` char(28) CHARACTER SET ascii NOT NULL,
  `user` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`property`, `id`)
);

CREATE TABLE `users` (
  `property` char(10) NOT NULL,
  `id` int(10) unsigned NOT NULL,
  `device` char(28) CHARACTER SET ascii DEFAULT NULL,
  PRIMARY KEY (`property`,`id`)
);

ALTER TABLE `customers` MODIFY COLUMN `id` INT NOT NULL AUTO_INCREMENT;
ALTER TABLE `properties` MODIFY COLUMN `customer` INT NOT NULL;
ALTER TABLE `smart_events` MODIFY COLUMN `id` INT NOT NULL AUTO_INCREMENT;
ALTER TABLE `devices` MODIFY COLUMN `user` int unsigned DEFAULT NULL;
ALTER TABLE `users` MODIFY COLUMN `id` int unsigned NOT NULL;
