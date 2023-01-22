-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

CREATE TABLE users
(
    `id` Int32,
    `FirstName` String,
    `LastName` String,
    `Email` String,
    `Gender` Enum8('male' = 1, 'female' = 2, 'other' = 3),
    `FoodPreferences_Drink` Enum8('water' = 1, 'wine' = 2, 'beer' = 3, 'tea' = 4),
    `FoodPreferences_Fruit` Enum8('apple' = 1, 'orange' = 2, 'mango' = 3, 'peach' = 4, 'lime' = 5),
    `PhoneNumbers` Array(String)
)
ENGINE = MergeTree
PRIMARY KEY id
ORDER BY id
SETTINGS index_granularity = 8192;

CREATE TABLE events
(
    `source` Int32,
    `anonymous_id` UUID,
    `user_id` String,
    `date` Date,
    `timestamp` DateTime,
    `sent_at` DateTime,
    `received_at` DateTime,
    `ip` IPv6,
    `os_name` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'macOS' = 4, 'Linux' = 5, 'Chrome OS' = 6, 'Other' = 7),
    `os_version` String,
    `user_agent` String,
    `browser_name` Enum8('Chrome' = 1, 'Safari' = 2, 'Edge' = 3, 'Firefox' = 4, 'Samsung Internet' = 5, 'Opera' = 6, 'Other' = 7),
    `browser_other` String,
    `browser_version` String,
    `location_city` String,
    `location_country_code` String,
    `location_country_name` String,
    `location_latitude` Float64,
    `location_longitude` Float64,
    `device_type` Enum8('desktop' = 1, 'tablet' = 2, 'mobile' = 3),
    `event` String,
    `language` FixedString(2),
    `page_path` String,
    `page_referrer` String,
    `page_title` String,
    `page_url` String,
    `page_search` String,
    `target` String,
    `text` String
)
ENGINE = MergeTree
PRIMARY KEY (source, date, user_id, timestamp)
ORDER BY (source, date, user_id, timestamp)
SETTINGS index_granularity = 8192;
