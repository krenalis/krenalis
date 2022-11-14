
CREATE TABLE `accounts` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(45) NOT NULL DEFAULT '',
  `email` VARCHAR(120) NOT NULL DEFAULT '',
  `password` VARCHAR(60) CHARACTER SET ascii NOT NULL DEFAULT '',
  `internalIPs` VARCHAR(160) CHARACTER SET ascii NOT NULL DEFAULT '',
  PRIMARY KEY (`id`));

INSERT INTO `accounts` (`name`,`email`,`password`) VALUES ('ACME inc','acme@open2b.com','$2a$10$iMuokZyvwdAQOJJmJvG83eSGGWTV3DOjI2DRU6SjuLEuK.vknUJVC'); /* Password: foopass2 */

CREATE TABLE `connectors` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(200) NOT NULL DEFAULT '',
  `type` ENUM('App', 'Database', 'File', 'Mobile', 'Server', 'Storage', 'Website') NOT NULL DEFAULT 'App',
  `logoURL` VARCHAR(500) NOT NULL DEFAULT '',
  `webhooksPer` ENUM('None', 'Connector', 'Resource', 'Source') NOT NULL DEFAULT 'None',
  `oAuthURL` VARCHAR(500) NOT NULL DEFAULT '',
  `oAuthClientID` VARCHAR(500) NOT NULL DEFAULT '',
  `oAuthClientSecret` VARCHAR(500) NOT NULL DEFAULT '',
  `oAuthTokenEndpoint` VARCHAR(500) NOT NULL DEFAULT '',
  `oAuthDefaultTokenType` VARCHAR(10) NOT NULL DEFAULT '',
  `oAuthDefaultExpiresIn` INT NOT NULL DEFAULT '0',
  `oAuthForcedExpiresIn` VARCHAR(10) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
);

INSERT INTO `connectors` VALUES
    ('1','HubSpot','App','https://cdn4.iconfinder.com/data/icons/logos-and-brands/512/168_Hubspot_logo_logos-512.png','Connector','https://app-eu1.hubspot.com/oauth/authorize?client_id=cef1005a-72be-4047-a301-ef6057588325&redirect_uri=https://localhost:9090/admin/oauth/authorize&scope=crm.objects.contacts.read%20crm.objects.contacts.write%20crm.schemas.contacts.read','cef1005a-72be-4047-a301-ef6057588325','136e50df-5b89-478f-bf01-4a71547fa668','https://api.hubapi.com/oauth/v1/token','','0',''),
    ('2','MySQL','Database','https://cdn4.iconfinder.com/data/icons/logos-3/181/MySQL-512.png','None','','','','','','0',''),
    ('3','Dummy','App','','Connector','https://app-eu1.hubspot.com/oauth/authorize?client_id=cef1005a-72be-4047-a301-ef6057588325&redirect_uri=https://localhost:9090/admin/oauth/authorize&scope=crm.objects.contacts.read%20crm.objects.contacts.write%20crm.schemas.contacts.read','cef1005a-72be-4047-a301-ef6057588325','136e50df-5b89-478f-bf01-4a71547fa668','https://api.hubapi.com/oauth/v1/token','','0',''),
    ('4','Mailchimp','App','https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/mailchimp-512.png','Source','https://login.mailchimp.com/oauth2/authorize?response_type=code&client_id=631597222767&redirect_uri=https://127.0.0.1:9090/admin/oauth/authorize','631597222767','90c2d1a1383de35e5ecca5a73f0e2c19e751056d0e3cdd81ac','https://login.mailchimp.com/oauth2/token','','0','never'),
    ('5','CSV','File','https://cdn3.iconfinder.com/data/icons/cad-database-presentation-spreadsheet-vector-fil-2/512/19-512.png','None','','','','','','0',''),
    ('6','SFTP','Storage','https://cdn2.iconfinder.com/data/icons/whcompare-servers-web-hosting/50/sftp-512.png','None','','','','','','0',''),
    ('7','HTTP','Storage','https://cdn4.iconfinder.com/data/icons/application-windows-3/32/HTTP-512.png','None','','','','','','0',''),
    ('8','Excel','File','https://cdn0.iconfinder.com/data/icons/logos-microsoft-office-365/128/Microsoft_Office-02-512.png','None','','','','','','0',''),
    ('9','S3','Storage','https://cdn2.iconfinder.com/data/icons/amazon-aws-stencils/100/Storage__Content_Delivery_Amazon_S3-512.png','None','','','','','','0',''),
    ('10','PostgreSQL','Database','https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/postgresql-512.png','None','','','','','','0',''),
    ('11','Parquet','File','','None','','','','','','0',''),
    ('12','Website','Website','https://cdn2.iconfinder.com/data/icons/free-simple-line-mix/48/22-Website-512.png','None','','','','','','0','');

CREATE TABLE `connections` (
  `id` INT NOT NULL,
  `workspace` INT NOT NULL,
  `type` ENUM('App', 'Database', 'File', 'Mobile', 'Server', 'Storage', 'Website') NOT NULL DEFAULT 'App',
  `role` ENUM('Source', 'Destination') NOT NULL DEFAULT 'Source',
  `connector` INT NOT NULL,
  `storage` INT NOT NULL,
  `resource` INT NOT NULL,
  `websiteHost` varchar(261) CHARACTER SET ascii NOT NULL DEFAULT '',
  `userCursor` VARCHAR(500) NOT NULL DEFAULT '',
  `identityColumn` VARCHAR(100) NOT NULL DEFAULT '',
  `timestampColumn` VARCHAR(100) NOT NULL DEFAULT '',
  `settings` TEXT NOT NULL,
  `properties` MEDIUMTEXT NOT NULL,
  `usedProperties` MEDIUMTEXT NOT NULL,
  `usersQuery` MEDIUMTEXT NOT NULL,
  PRIMARY KEY (`id`)
);

CREATE TABLE `connections_imports` (
    `id` INT NOT NULL AUTO_INCREMENT,
    `connection` INT NOT NULL,
    `storage` INT NOT NULL,
    `startTime` DATETIME NOT NULL,
    `endTime` DATETIME NOT NULL,
    `error` VARCHAR(1000) NOT NULL DEFAULT '',
    PRIMARY KEY(`id`)
);

CREATE TABLE `connections_keys` (
    `connection` INT NOT NULL,
    `position` TINYINT UNSIGNED NOT NULL,
    `key` CHAR(32) CHARACTER SET ascii NOT NULL,
    PRIMARY KEY(`connection`, `position`)
);

CREATE TABLE `transformations` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `goldenRecordName` VARCHAR(100) NOT NULL DEFAULT '',
  `sourceCode` TEXT NOT NULL,
  PRIMARY KEY (`id`)
);

CREATE TABLE `transformations_connections` (
  `connection` INT NOT NULL,
  `property` VARCHAR(50) NOT NULL DEFAULT '',
  `transformation` INT,
  PRIMARY KEY (`connection`, `property`, `transformation`)
);

CREATE TABLE `connections_stats` (
    `connection` INT NOT NULL,
    `timeSlot` INT NOT NULL,
    `usersIn` INT NOT NULL,
    PRIMARY KEY (`connection`, `timeSlot`)
);

CREATE TABLE `connections_users` (
  `connection` int NOT NULL,
  `user` varchar(45) NOT NULL DEFAULT '',
  `data` text NOT NULL,
  `timestamps` text NOT NULL DEFAULT '',
  `goldenRecord` int unsigned NOT NULL,
  PRIMARY KEY (`connection`, `user`)
);

CREATE TABLE `devices` (
  `property` INT unsigned NOT NULL,
  `id` char(28) CHARACTER SET ascii NOT NULL,
  `user` int unsigned DEFAULT NULL,
  PRIMARY KEY (`property`, `id`)
);

CREATE TABLE `domains` (
  `property` INT unsigned NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  PRIMARY KEY (`property`, `name`));

CREATE TABLE `properties` (
  `id` INT unsigned NOT NULL AUTO_INCREMENT,
  `code` CHAR(10) CHARACTER SET ascii NOT NULL,
  `account` INT NOT NULL,
  UNIQUE KEY `code` (`code`),
  KEY `account` (`account`),
  PRIMARY KEY (`id`));

INSERT INTO `properties` VALUES (1,'1234567890',1);

CREATE TABLE `resources` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `connector` INT NOT NULL,
  `code` VARCHAR(100) NOT NULL DEFAULT '',
  `oAuthAccessToken` VARCHAR(500) NOT NULL DEFAULT '',
  `oAuthRefreshToken` VARCHAR(500) NOT NULL DEFAULT '',
  `oAuthExpiresIn` DATETIME NOT NULL DEFAULT '0000-00-00 00:00:00',
  KEY `connector` (`connector`),
  PRIMARY KEY (`id`)
);

CREATE TABLE `smart_events` (
  `property` INT unsigned NOT NULL,
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL DEFAULT '',
  `event` VARCHAR(50) NOT NULL DEFAULT '',
  `pages` TEXT NOT NULL,
  `buttons` TEXT NOT NULL,
  PRIMARY KEY (`id`)
);

INSERT INTO `smart_events` VALUES (1,50,'View Nissan Car','pageview','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','null'),(1,51,'Configure a Nissan Car','click','[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]','[{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configure your car\",\"Domain\":\"english.example.com\"},{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configura la tua auto\",\"Domain\":\"italian.example.com\"}]'),(1,52,'Click on Login Button','click','null','[{\"Field\":\"text\",\"Operator\":\"Contains\",\"Value\":\"Log in\"}]');

CREATE TABLE `users` (
  `property` INT unsigned NOT NULL,
  `id` int unsigned NOT NULL,
  `device` char(28) CHARACTER SET ascii DEFAULT NULL,
  PRIMARY KEY (`property`,`id`)
);

CREATE TABLE `warehouse_users` (
  `id` INT unsigned NOT NULL AUTO_INCREMENT,
  `Email` VARCHAR(500) NOT NULL DEFAULT '',
  `FirstName` VARCHAR(500) NOT NULL DEFAULT '',
  `LastName` VARCHAR(500) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
);

CREATE TABLE `workspaces` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `account` INT NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  `userSchema` TEXT NOT NULL,
  `groupSchema` TEXT NOT NULL,
  `eventSchema` TEXT NOT NULL,
  PRIMARY KEY (`id`)
);

INSERT INTO `workspaces` (`id`, `account`, `name`, `userSchema`, `groupSchema`) VALUES ('1', '1', 'Workspace', '{\n    \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n    \"$id\": \"https://example.com/product.schema.json\",\n    \"description\": \"Schema di uno user\",\n    \"self\": {\n        \"vendor\": \"com.example\",\n        \"name\": \"schema_1\",\n        \"format\": \"jsonschema\",\n        \"version\": \"1-0-0\"\n    },\n    \"type\": \"object\",\n    \"properties\": {\n        \"FirstName\": {\n            \"title\": \"First name\",\n            \"description\": \"First of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n        \"LastName\": {\n          \"title\": \"Last name\",\n            \"description\": \"Last name of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n        \"Email\": {\n            \"title\": \"Email address\",\n            \"description\": \"Email address of the user\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        }\n    },\n    \"additionalProperties\": false\n}', '{\n    \"$schema\": \"https://json-schema.org/draft/2020-12/schema\",\n    \"$id\": \"https://example.com/product.schema.json\",\n    \"description\": \"Schema di un gruppo\",\n    \"self\": {\n        \"vendor\": \"com.example\",\n        \"name\": \"schema_1\",\n        \"format\": \"jsonschema\",\n        \"version\": \"1-0-0\"\n    },\n    \"type\": \"object\",\n    \"properties\": {\n        \"Name\": {\n            \"title\": \"Group name\",\n            \"description\": \"Name of the group\",\n            \"type\": [\n                \"string\",\n                \"null\"\n            ],\n            \"maxLength\": 300\n        },\n    },\n    \"additionalProperties\": false\n}');
