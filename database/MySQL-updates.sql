
ALTER TABLE `customers` CHANGE COLUMN `password` `password` VARCHAR(60) NOT NULL DEFAULT '';
ALTER TABLE `properties` CHANGE COLUMN `code` `id` CHAR(10) NOT NULL;
RENAME TABLE `properties_domains` TO `domains`;
ALTER TABLE `domains` CHANGE COLUMN `domain` `name` VARCHAR(255) NOT NULL;
