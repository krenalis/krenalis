CREATE TABLE `customers` (
  `id` INT(10) NOT NULL AUTO_INCREMENT ,
  `name` VARCHAR(45) NOT NULL DEFAULT '',
  `email` VARCHAR(120) NOT NULL DEFAULT '',
  `password` VARCHAR(60) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`));

CREATE TABLE `properties` (
  `id` CHAR(10) NOT NULL,
  `customer` INT(10) NOT NULL,
  PRIMARY KEY (`id`));

CREATE TABLE `domains` (
  `property` CHAR(10) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  PRIMARY KEY (`property`, `name`));

/* default customer and properties */
INSERT INTO `customers` (`name`,`email`,`password`) VALUES ('ACME inc','acme@open2b.com','foopass');
INSERT INTO `properties` VALUES ('1234567890',1);
