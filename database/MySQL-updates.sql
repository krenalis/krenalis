
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

TRUNCATE TABLE `smart_events`;
INSERT INTO `smart_events` VALUES ('ABCDEFGHIJ',1,'View Nissan Car','pageview','[{\"Field\":\"url\",\"Operator\":\"StartsWith\",\"Value\":\"/cars/Nissan/\"},{\"Field\":\"url\",\"Operator\":\"StartsWith\",\"Value\":\"/auto/Nissan/\"}]','null'),('1234567890',38,'View Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','null'),('1234567890',39,'Configure a Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','[{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configure your car\",\"Domain\":\"english.example.com\"},{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configura la tua auto\",\"Domain\":\"italian.example.com\"}]'),('1234567890',40,'Click on Login Button','click','null','[{\"Field\":\"text\",\"Operator\":\"Contains\",\"Value\":\"Log in\"}]');

TRUNCATE TABLE `smart_events`;
INSERT INTO `smart_events` VALUES ('1234567890',50,'View Nissan Car','pageview','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','null'),('1234567890',51,'Configure a Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','[{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configure your car\",\"Domain\":\"english.example.com\"},{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configura la tua auto\",\"Domain\":\"italian.example.com\"}]'),('1234567890',52,'Click on Login Button','click','null','[{\"Field\":\"text\",\"Operator\":\"Contains\",\"Value\":\"Log in\"}]');

ALTER TABLE `properties`
  DROP PRIMARY KEY,
  CHANGE COLUMN `id` `code` CHAR(10) CHARACTER SET ascii NOT NULL,
  ADD COLUMN `id` INT unsigned NOT NULL AUTO_INCREMENT FIRST,
  ADD UNIQUE KEY `code` (`code`),
  ADD KEY `customer` (`customer`),
  ADD PRIMARY KEY (`id`);

TRUNCATE `domains`;
ALTER TABLE `domains` MODIFY COLUMN `property` INT UNSIGNED NOT NULL;

TRUNCATE TABLE `smart_events`;
ALTER TABLE `smart_events` MODIFY COLUMN `property` INT unsigned NOT NULL;
INSERT INTO `smart_events` VALUES (1,50,'View Nissan Car','pageview','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','null'),(1,51,'Configure a Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','[{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configure your car\",\"Domain\":\"english.example.com\"},{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configura la tua auto\",\"Domain\":\"italian.example.com\"}]'),(1,52,'Click on Login Button','click','null','[{\"Field\":\"text\",\"Operator\":\"Contains\",\"Value\":\"Log in\"}]');

TRUNCATE TABLE `devices`;
ALTER TABLE `devices` MODIFY COLUMN `property` INT unsigned NOT NULL;

TRUNCATE TABLE `users`;
ALTER TABLE `users` MODIFY COLUMN `property` INT unsigned NOT NULL;

CREATE TABLE `schemas` (
  `account` INT NOT NULL,
  `user_schema` TEXT NOT NULL DEFAULT '',
  `group_schema` TEXT NOT NULL DEFAULT '',
  `event_schema` TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (`account`)
);

CREATE TABLE `connectors` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(200) NOT NULL DEFAULT '',
  `oauth_url` VARCHAR(500) NOT NULL DEFAULT '',
  `logo_url` VARCHAR(500) NOT NULL DEFAULT '',
  `client_id` VARCHAR(500) NOT NULL DEFAULT '',
  `client_secret` VARCHAR(500) NOT NULL DEFAULT '',
  `token_endpoint` VARCHAR(500) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
);

INSERT INTO `connectors` (`id`, `name`, `oauth_url`, `logo_url`, `client_id`, `client_secret`, `token_endpoint`) VALUES ('1', 'Hubspot', 'https://app-eu1.hubspot.com/oauth/authorize?client_id=cef1005a-72be-4047-a301-ef6057588325&redirect_uri=https://localhost:9090/admin/oauth/authorize&scope=crm.objects.contacts.read%20crm.objects.contacts.write%20crm.schemas.contacts.read', 'https://cdn4.iconfinder.com/data/icons/logos-and-brands/512/168_Hubspot_logo_logos-512.png', 'cef1005a-72be-4047-a301-ef6057588325', '136e50df-5b89-478f-bf01-4a71547fa668', 'https://api.hubapi.com/oauth/v1/token');

CREATE TABLE `account_connectors` (
  `account` INT NOT NULL,
  `connector` INT NOT NULL,
  `access_token` VARCHAR(500) NOT NULL DEFAULT '',
  `refresh_token` VARCHAR(500) NOT NULL DEFAULT '',
  `access_token_expiration_timestamp` TIMESTAMP NOT NULL,
  PRIMARY KEY (`account`, `connector`)
);

CREATE TABLE `connectors_raw_users_data` (
  `account` int NOT NULL,
  `connector` int NOT NULL,
  `data` text NOT NULL,
  PRIMARY KEY (`account`,`connector`)
);

INSERT INTO `schemas` (`account`, `user_schema`, `group_schema`) VALUES ('1', '{\n    \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n    \"$id\": \"https://example.com/product.schema.json\",\n    \"description\": \"Schema di uno user\",\n    \"self\": {\n        \"vendor\": \"com.example\",\n        \"name\": \"schema_1\",\n        \"format\": \"jsonschema\",\n        \"version\": \"1-0-0\"\n    },\n    \"type\": \"object\",\n    \"properties\": {\n        \"FirstName\": {\n            \"title\": \"First name\",\n            \"description\": \"First of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n        \"LastName\": {\n          \"title\": \"Last name\",\n            \"description\": \"Last name of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n        \"Email\": {\n            \"title\": \"Email address\",\n            \"description\": \"Email address of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        }\n    },\n    \"additionalProperties\": false\n}', '{\n    \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n    \"$id\": \"https://example.com/product.schema.json\",\n    \"description\": \"Schema di un gruppo\",\n    \"self\": {\n        \"vendor\": \"com.example\",\n        \"name\": \"schema_1\",\n        \"format\": \"jsonschema\",\n        \"version\": \"1-0-0\"\n    },\n    \"type\": \"object\",\n    \"properties\": {\n        \"Name\": {\n            \"title\": \"Group name\",\n            \"description\": \"Name of the group\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n    },\n    \"additionalProperties\": false\n}');

ALTER TABLE `connectors_raw_users_data` 
	ADD COLUMN `user` VARCHAR(45) NOT NULL DEFAULT '' AFTER `account`,
	ADD COLUMN `internalUserID` INT NOT NULL AFTER `data`,
	CHANGE COLUMN `connector` `connector` INT NOT NULL FIRST,
	DROP PRIMARY KEY,
	ADD PRIMARY KEY (`connector`, `account`, `user`);

ALTER TABLE `account_connectors` ADD COLUMN `transformation` TEXT NOT NULL DEFAULT '' AFTER `access_token_expiration_timestamp`;

ALTER TABLE `account_connectors` 
  ADD COLUMN `user_cursor` VARCHAR(500) NOT NULL DEFAULT '' AFTER `transformation`;

UPDATE `connectors` SET `name` = 'HubSpot' WHERE (`id` = '1');

CREATE TABLE `warehouse_users` (
  `Email` VARCHAR(500) NOT NULL DEFAULT '',
  `FirstName` VARCHAR(500) NOT NULL DEFAULT '',
  `LastName` VARCHAR(500) NOT NULL DEFAULT '',
  PRIMARY KEY (`Email`)
);

CREATE TABLE `connectors_properties` (
  `account` INT NOT NULL,
  `connector` INT NOT NULL,
  `name` VARCHAR(100) NOT NULL DEFAULT '',
  `type` VARCHAR(100) NOT NULL DEFAULT '',
  `label` VARCHAR(100) NOT NULL DEFAULT '',
  `options` TEXT NOT NULL DEFAULT '',
  `position` INT NOT NULL,
  PRIMARY KEY (`account`, `connector`, `name`)
);

ALTER TABLE `properties`
    CHANGE COLUMN `customer` `account` INT NOT NULL,
    RENAME KEY `customer` TO `account`;

RENAME TABLE `customers` TO `accounts`;
