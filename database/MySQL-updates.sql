
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

CREATE TABLE `data_sources` (
  `account` INT NOT NULL,
  `connector` INT NOT NULL,
  `access_token` VARCHAR(500) NOT NULL DEFAULT '',
  `refresh_token` VARCHAR(500) NOT NULL DEFAULT '',
  `access_token_expiration_timestamp` TIMESTAMP NOT NULL,
  PRIMARY KEY (`account`, `connector`)
);

CREATE TABLE `data_sources_raw_users_data` (
  `account` int NOT NULL,
  `connector` int NOT NULL,
  `data` text NOT NULL,
  PRIMARY KEY (`account`,`connector`)
);

INSERT INTO `schemas` (`account`, `user_schema`, `group_schema`) VALUES ('1', '{\n    \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n    \"$id\": \"https://example.com/product.schema.json\",\n    \"description\": \"Schema di uno user\",\n    \"self\": {\n        \"vendor\": \"com.example\",\n        \"name\": \"schema_1\",\n        \"format\": \"jsonschema\",\n        \"version\": \"1-0-0\"\n    },\n    \"type\": \"object\",\n    \"properties\": {\n        \"FirstName\": {\n            \"title\": \"First name\",\n            \"description\": \"First of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n        \"LastName\": {\n          \"title\": \"Last name\",\n            \"description\": \"Last name of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n        \"Email\": {\n            \"title\": \"Email address\",\n            \"description\": \"Email address of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        }\n    },\n    \"additionalProperties\": false\n}', '{\n    \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n    \"$id\": \"https://example.com/product.schema.json\",\n    \"description\": \"Schema di un gruppo\",\n    \"self\": {\n        \"vendor\": \"com.example\",\n        \"name\": \"schema_1\",\n        \"format\": \"jsonschema\",\n        \"version\": \"1-0-0\"\n    },\n    \"type\": \"object\",\n    \"properties\": {\n        \"Name\": {\n            \"title\": \"Group name\",\n            \"description\": \"Name of the group\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n    },\n    \"additionalProperties\": false\n}');

ALTER TABLE `data_sources_raw_users_data`
	ADD COLUMN `user` VARCHAR(45) NOT NULL DEFAULT '' AFTER `account`,
	ADD COLUMN `internalUserID` INT NOT NULL AFTER `data`,
	CHANGE COLUMN `connector` `connector` INT NOT NULL FIRST,
	DROP PRIMARY KEY,
	ADD PRIMARY KEY (`connector`, `account`, `user`);

ALTER TABLE `data_sources` ADD COLUMN `transformation` TEXT NOT NULL DEFAULT '' AFTER `access_token_expiration_timestamp`;

ALTER TABLE `data_sources`
  ADD COLUMN `user_cursor` VARCHAR(500) NOT NULL DEFAULT '' AFTER `transformation`;

UPDATE `connectors` SET `name` = 'HubSpot' WHERE (`id` = '1');

CREATE TABLE `warehouse_users` (
  `Email` VARCHAR(500) NOT NULL DEFAULT '',
  `FirstName` VARCHAR(500) NOT NULL DEFAULT '',
  `LastName` VARCHAR(500) NOT NULL DEFAULT '',
  PRIMARY KEY (`Email`)
);

CREATE TABLE `data_sources_properties` (
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

RENAME TABLE `data_sources` TO `data_sources`;
RENAME TABLE `data_sources_properties` TO `data_sources_properties`;
RENAME TABLE `data_sources_raw_users_data` TO `data_sources_raw_users_data`;

ALTER TABLE `connectors`
    RENAME COLUMN `oauth_url` TO `oauthURL`,
    RENAME COLUMN `logo_url` TO `logoURL`,
    RENAME COLUMN `client_id` TO `clientID`,
    RENAME COLUMN `client_secret` TO `clientSecret`,
    RENAME COLUMN `token_endpoint` TO `tokenEndpoint`;

ALTER TABLE `schemas`
    RENAME COLUMN `user_schema` TO `userSchema`,
    RENAME COLUMN `group_schema` TO `groupSchema`,
    RENAME COLUMN `event_schema` TO `eventSchema`;

ALTER TABLE `data_sources`
    RENAME COLUMN `access_token` TO `accessToken`,
    RENAME COLUMN `refresh_token` TO `refreshToken`,
    RENAME COLUMN `access_token_expiration_timestamp` TO `accessTokenExpirationTimestamp`,
    RENAME COLUMN `user_cursor` TO `userCursor`;

CREATE TABLE `workspaces` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `account` INT NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  PRIMARY KEY (`id`)
);

INSERT INTO `workspaces` (`id`, `account`, `name`) VALUES (1, 1, 'Workspace');

ALTER TABLE `data_sources`
    CHANGE `account` `workspace` INT NOT NULL,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`workspace` , `connector`);

ALTER TABLE `data_sources_properties`
    CHANGE `account` `workspace` INT NOT NULL,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`workspace`, `connector`, `name`);

ALTER TABLE `data_sources_raw_users_data`
    CHANGE `account` `workspace` INT NOT NULL,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`connector`, `workspace`, `user`);

ALTER TABLE `schemas`
    CHANGE `account` `workspace` INT NOT NULL,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`workspace`);

CREATE TABLE `connectors_resources` (
  `connector` INT NOT NULL,
  `resource` VARCHAR(100) NOT NULL DEFAULT '',
  `accessToken` VARCHAR(500) NOT NULL DEFAULT '',
  `refreshToken` VARCHAR(500) NOT NULL DEFAULT '',
  `accessTokenExpirationTimestamp` TIMESTAMP NOT NULL,
  PRIMARY KEY (`connector`, `resource`),
);

ALTER TABLE `data_sources` (
    ADD COLUMN `resource` VARCHAR(100) NOT NULL DEFAULT '' AFTER `workspace`,
    DROP COLUMN `accessToken`,
    DROP COLUMN `refreshToken`,
    DROP COLUMN `accessTokenExpirationTimestamp`,
);

ALTER TABLE `data_sources_raw_users_data`
    CHANGE `connector` `connector` int NOT NULL AFTER `workspace`,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`workspace`, `connector`, `user`);

ALTER TABLE `data_sources_raw_users_data`
    ADD COLUMN `resource`  varchar(100) NOT NULL DEFAULT '' AFTER `connector`,
    DROP COLUMN `internalUserID`,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`workspace`, `connector`, `resource`, `user`);

RENAME TABLE `connectors_resources` TO `resources`;
RENAME TABLE `data_sources_properties` TO `resources_properties`;

ALTER TABLE `resources_properties`
    DROP PRIMARY KEY,
    DROP COLUMN `workspace`,
    ADD COLUMN `resource` VARCHAR(100) NOT NULL DEFAULT '' AFTER `connector`,
    ADD PRIMARY KEY (`connector`, `resource`, `name`);

ALTER TABLE `workspaces`
    ADD COLUMN `userSchema` TEXT NOT NULL AFTER `name`,
    ADD COLUMN `groupSchema` TEXT NOT NULL AFTER `userSchema`,
    ADD COLUMN `eventSchema` TEXT NOT NULL AFTER `groupSchema`;

UPDATE `workspaces` AS `w`, `schemas` AS `s`
SET `w`.`userSchema` = `s`.`userSchema`, `w`.`groupSchema` = `s`.`groupSchema`, `w`.`eventSchema` = `s`.`eventSchema`
WHERE `s`.`workspace` = `w`.`id`;

DROP TABLE `schemas`;

RENAME TABLE `data_sources_raw_users_data` TO `data_sources_users`;

ALTER TABLE `data_sources_users`
    DROP PRIMARY KEY,
    DROP COLUMN `resource`,
    ADD PRIMARY KEY (`workspace`, `connector`, `user`);

RENAME TABLE `resources_properties` TO `data_sources_properties`;

ALTER TABLE `data_sources_properties`
    DROP PRIMARY KEY,
    ADD COLUMN `workspace` INT NOT NULL FIRST,
    DROP COLUMN `resource`,
    ADD PRIMARY KEY (`workspace`, `connector`, `name`);

UPDATE `data_sources_properties` SET `workspace` = 1;

ALTER TABLE `connectors`
    ADD COLUMN `webhooksPer` ENUM('Connector', 'Resource', 'DataSource') NOT NULL DEFAULT 'Connector' AFTER `tokenEndpoint`;

ALTER TABLE `data_sources`
    ADD COLUMN `settings` VARCHAR(10000) NOT NULL DEFAULT '' AFTER `userCursor`;

ALTER TABLE `data_sources`
    ADD COLUMN `id` INT NOT NULL AUTO_INCREMENT FIRST,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`id`);

ALTER TABLE `data_sources_properties`
    DROP PRIMARY KEY,
    DROP COLUMN `connector`,
    DROP COLUMN `workspace`,
    ADD COLUMN `source` INT NOT NULL,
    ADD PRIMARY KEY (`source` , `name`);

ALTER TABLE `data_sources_users`
    DROP PRIMARY KEY,
    DROP COLUMN `connector`,
    DROP COLUMN `workspace`,
    ADD COLUMN `source` INT NOT NULL,
    ADD PRIMARY KEY (`source` , `user`);

ALTER TABLE `data_sources_users` 
    ADD COLUMN `timestamps` TEXT NOT NULL DEFAULT '' AFTER `data`;

TRUNCATE TABLE `connectors`;
INSERT INTO `connectors` VALUES (1,'HubSpot','https://app-eu1.hubspot.com/oauth/authorize?client_id=cef1005a-72be-4047-a301-ef6057588325&redirect_uri=https://localhost:9090/admin/oauth/authorize&scope=crm.objects.contacts.read%20crm.objects.contacts.write%20crm.schemas.contacts.read','https://cdn4.iconfinder.com/data/icons/logos-and-brands/512/168_Hubspot_logo_logos-512.png','cef1005a-72be-4047-a301-ef6057588325','136e50df-5b89-478f-bf01-4a71547fa668','https://api.hubapi.com/oauth/v1/token','Connector'),(2,'Dummy','https://app-eu1.hubspot.com/oauth/authorize?client_id=cef1005a-72be-4047-a301-ef6057588325&redirect_uri=https://localhost:9090/admin/oauth/authorize&scope=crm.objects.contacts.read%20crm.objects.contacts.write%20crm.schemas.contacts.read','','cef1005a-72be-4047-a301-ef6057588325','136e50df-5b89-478f-bf01-4a71547fa668','https://api.hubapi.com/oauth/v1/token','Connector');

CREATE TABLE `data_sources_stats` (
    `source` INT NOT NULL,
    `timeSlot` INT NOT NULL,
    `usersIn` INT NOT NULL,
    PRIMARY KEY (`source`, `timeSlot`)
);

ALTER TABLE `resources`
    DROP PRIMARY KEY,
    ADD COLUMN `id` INT NOT NULL AUTO_INCREMENT FIRST,
    RENAME COLUMN `resource` TO `code`,
    ADD KEY `connector` (`connector`),
    ADD PRIMARY KEY (`id`);

ALTER TABLE `data_sources`
    RENAME COLUMN `resource` TO `resourceCode`,
    ADD COLUMN `resource` INT NOT NULL AFTER `resourceCode`;

UPDATE `data_sources` AS `s`
INNER JOIN `resources` AS `r` ON `r`. `code` = `s`.`resourceCode`
SET `s`.`resource` = `r`.`id`;

ALTER TABLE `data_sources` DROP COLUMN `resourceCode`;

ALTER TABLE `data_sources`
    ADD COLUMN `properties` MEDIUMTEXT NOT NULL,
    ADD COLUMN `usedProperties` MEDIUMTEXT NOT NULL;

DROP TABLE `data_sources_properties`;

ALTER TABLE `connectors`
    ADD COLUMN `type` ENUM('App', 'Database') DEFAULT 'App' AFTER `name`,
    CHANGE COLUMN `webhooksPer` `webhooksPer` ENUM('None', 'Connector', 'Resource', 'DataSource') NOT NULL DEFAULT 'None';

ALTER TABLE `data_sources`
    ADD COLUMN `usersQuery` MEDIUMTEXT NOT NULL AFTER `usedProperties`;

INSERT INTO `connectors` (`name`, `type`, `logoURL`) VALUES
    ('MySQL','Database','https://cdn4.iconfinder.com/data/icons/logos-3/181/MySQL-512.png');

ALTER TABLE `connectors`
    CHANGE COLUMN `type` `type` ENUM('App', 'Database', 'Storage', 'File') DEFAULT 'App';

ALTER TABLE `connectors`
    CHANGE COLUMN `type` `type` ENUM('App', 'Database', 'File', 'Stream') DEFAULT 'App';

ALTER TABLE `data_sources`
    ADD COLUMN `stream` INT NOT NULL AFTER `connector`;

ALTER TABLE `data_sources`
    ADD COLUMN `type` ENUM('App', 'Database', 'FileStream') DEFAULT 'App' AFTER `workspace`;

UPDATE `data_sources` AS `s`, `connectors` AS `c`
SET `s`.`type` = IF(`s`.`stream` = 0, `c`.`type`, 'FileStream')
WHERE `c`.`id` = `s`.`connector`;

ALTER TABLE `data_sources`
    CHANGE COLUMN `settings` `settings` TEXT NOT NULL,
    ADD COLUMN `streamSettings` TEXT NOT NULL AFTER `settings`;

ALTER TABLE `warehouse_users` 
    ADD COLUMN `id` INT UNSIGNED NOT NULL AUTO_INCREMENT FIRST,
    DROP PRIMARY KEY,
    ADD PRIMARY KEY (`id`);

ALTER TABLE `data_sources_users`
    ADD COLUMN `goldenRecord` INT UNSIGNED NOT NULL;

INSERT INTO `connectors` (`name`, `type`, `oauthURL`, `logoURL`, `clientID`, `clientSecret`, `tokenEndpoint`, `webhooksPer`, `defaultTokenType`, `defaultExpiresIn`, `forcedExpiresIn`)
VALUES ('Mailchimp', 'App', 'https://login.mailchimp.com/oauth2/authorize?response_type=code&client_id=631597222767&redirect_uri=https://127.0.0.1:9090/admin/oauth/authorize', 'https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/mailchimp-512.png', '631597222767', '90c2d1a1383de35e5ecca5a73f0e2c19e751056d0e3cdd81ac', 'https://login.mailchimp.com/oauth2/token', 'DataSource', 'bearer', '0', 'never');

ALTER TABLE `connectors`
    ADD COLUMN `defaultTokenType` VARCHAR(10) NOT NULL DEFAULT 'bearer' AFTER `webhooksPer`,
    ADD COLUMN `defaultExpiresIn` INT NOT NULL DEFAULT '0' AFTER `defaultTokenType`,
    ADD COLUMN `forcedExpiresIn` VARCHAR(10) NOT NULL DEFAULT '' AFTER `defaultExpiresIn`;

ALTER TABLE `resources`
    CHANGE COLUMN `accessTokenExpirationTimestamp` `accessTokenExpirationTime` DATETIME NOT NULL DEFAULT '0000-00-00 00:00:00';

INSERT INTO `connectors` (`name`,`type`,`logoURL`) VALUES
    ('CSV','File','https://cdn3.iconfinder.com/data/icons/cad-database-presentation-spreadsheet-vector-fil-2/512/19-512.png'),
    ('SFTP','Stream','https://cdn2.iconfinder.com/data/icons/whcompare-servers-web-hosting/50/sftp-512.png');

INSERT INTO `connectors` (`name`,`type`,`logoURL`) VALUES
    ('HTTP','Stream','https://cdn4.iconfinder.com/data/icons/application-windows-3/32/HTTP-500.png');

UPDATE `connectors`
SET `logoURL` = 'https://cdn4.iconfinder.com/data/icons/application-windows-3/32/HTTP-512.png'
WHERE `name` = 'HTTP';

INSERT INTO `connectors` (`name`,`type`,`logoURL`) VALUES
    ('Excel','File','https://cdn0.iconfinder.com/data/icons/logos-microsoft-office-365/128/Microsoft_Office-02-512.png');

INSERT INTO `connectors` (`name`,`type`,`logoURL`) VALUES
    ('S3','Stream','https://cdn2.iconfinder.com/data/icons/amazon-aws-stencils/100/Storage__Content_Delivery_Amazon_S3-512.png');

ALTER TABLE `data_sources`
    ADD COLUMN `identityColumn` VARCHAR(100) NOT NULL DEFAULT '' AFTER `userCursor`,
    ADD COLUMN `timestampColumn` VARCHAR(100) NOT NULL DEFAULT '' AFTER `identityColumn`;

ALTER TABLE `data_sources` 
    DROP COLUMN `transformation`;

CREATE TABLE `transformations` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `goldenRecordName` VARCHAR(100) NOT NULL DEFAULT '',
  `sourceCode` TEXT NOT NULL,
  PRIMARY KEY (`id`)
);

CREATE TABLE `transformations_connections` (
  `dataSource` INT NOT NULL,
  `property` VARCHAR(50) NOT NULL DEFAULT '',
  `transformation` INT,
  PRIMARY KEY (`dataSource`, `property`, `transformation`)
);

INSERT INTO `connectors` (`name`, `type`, `logoURL`) VALUES
    ('PostgreSQL','Database','https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/postgresql-512.png');

INSERT INTO `connectors` (`name`, `type`) VALUES ('Parquet','File');
