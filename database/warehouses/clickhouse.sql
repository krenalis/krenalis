-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

-- TODO: add the CREATE TABLE for "users_identities".

CREATE TABLE users
(
    `id` Int32,
    `dummy_id` String,
    `anonymous_id` String,
    "android_id" String,
    "android_idfa" String,
    "android_push_token" String,
    "ios_id" String,
    "ios_idfa" String,
    "ios_push_token" String,
    `first_name` String,
    `last_name` String,
    `email` String,
    `gender` Enum8('male' = 1, 'female' = 2, 'other' = 3),
    `food_preferences_drink` Enum8('water' = 1, 'wine' = 2, 'beer' = 3, 'tea' = 4),
    `food_preferences_fruit` Enum8('apple' = 1, 'orange' = 2, 'mango' = 3, 'peach' = 4, 'lime' = 5),
    `phone_numbers` Array(String)
)
ENGINE = MergeTree
PRIMARY KEY id
ORDER BY id
SETTINGS index_granularity = 8192;

-- TODO: add the CREATE TABLE for "groups_identities".

CREATE TABLE groups
(
    `id` Int32,
)
ENGINE = MergeTree
PRIMARY KEY id
ORDER BY id
SETTINGS index_granularity = 8192;

-- Keep in sync with the apis.eventColumns and events.batchEventsColumns variables.

CREATE TABLE events
(
    `gid` Int32,
    `anonymous_id` String,
    `category` String,
    `app_name` String,
    `app_version` String,
    `app_build` String,
    `app_namespace` String,
    `browser_name` Enum8('Chrome' = 1, 'Safari' = 2, 'Edge' = 3, 'Firefox' = 4, 'Samsung Internet' = 5, 'Opera' = 6, 'Other' = 7),
    `browser_other` String,
    `browser_version` String,
    `campaign_name` String,
    `campaign_source` String,
    `campaign_medium` String,
    `campaign_term` String,
    `campaign_content` String,
    `device_id` String,
    `device_advertising_id` String,
    `device_ad_tracking_enabled` Bool,
    `device_manufacturer` String,
    `device_model` String,
    `device_name` String,
    `device_type` String,
    `device_token` String,
    `ip` IPv6,
    `library_name` String,
    `library_version` String,
    `locale` String,
    `location_city` String,
    `location_country` String,
    `location_latitude` Float64,
    `location_longitude` Float64,
    `location_speed` Float64,
    `network_bluetooth` Bool,
    `network_carrier` String,
    `network_cellular` Bool,
    `network_wifi` Bool,
    `os_name` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'macOS' = 4, 'Linux' = 5, 'Chrome OS' = 6, 'Other' = 7),
    `os_version` String,
    `page_path` String,
    `page_referrer` String,
    `page_search` String,
    `page_title` String,
    `page_url` String,
    `referrer_id` String,
    `referrer_type` String,
    `screen_width` Int16,
    `screen_height` Int16,
    `screen_density` Decimal(3,2),
    `session_id` Int64,
    `session_start` Bool,
    `timezone` String,
    `user_agent` String,
    `event` String,
    `group_id` String,
    `message_id` String,
    `name` String,
    `properties` String,
    `received_at` DateTime,
    `sent_at` DateTime,
    `source` Int32,
    `timestamp` DateTime,
    `traits` String,
    `type` Enum8('alias' = 1, 'identify' = 2, 'group' = 3, 'page' = 4, 'screen' = 5, 'track' = 6),
    `user_id` String
)
ENGINE = MergeTree
PRIMARY KEY (source, date, user_id, timestamp)
ORDER BY (source, date, user_id, timestamp)
SETTINGS index_granularity = 8192;
