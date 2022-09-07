
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
