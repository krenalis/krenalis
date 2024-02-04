-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

-- TODO: add the CREATE TABLE for "users_identities".

CREATE TABLE users
(
    `_id` Int32,
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
PRIMARY KEY `_id`
ORDER BY `_id`
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

-- Keep in sync with the events.eventsMergeTable variable.

CREATE TABLE events
(
    `gid` Int32 DEFAULT 0,
    `anonymousId` String,
    `category` String,
    `context_app_name` String,
    `context_app_version` String,
    `context_app_build` String,
    `context_app_namespace` String,
    `context_browser_name` Enum8('Chrome' = 1, 'Safari' = 2, 'Edge' = 3, 'Firefox' = 4, 'Samsung Internet' = 5, 'Opera' = 6, 'Other' = 7),
    `context_browser_other` String,
    `context_browser_version` String,
    `context_campaign_name` String,
    `context_campaign_source` String,
    `context_campaign_medium` String,
    `context_campaign_term` String,
    `context_campaign_content` String,
    `context_device_id` String,
    `context_device_advertisingId` String,
    `context_device_adTrackingEnabled` Bool,
    `context_device_manufacturer` String,
    `context_device_model` String,
    `context_device_name` String,
    `context_device_type` String,
    `context_device_token` String,
    `context_ip` IPv6,
    `context_library_name` String,
    `context_library_version` String,
    `context_locale` String,
    `context_location_city` String,
    `context_location_country` String,
    `context_location_latitude` Float64,
    `context_location_longitude` Float64,
    `context_location_speed` Float64,
    `context_network_bluetooth` Bool,
    `context_network_carrier` String,
    `context_network_cellular` Bool,
    `context_network_wifi` Bool,
    `context_os_name` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'macOS' = 4, 'Linux' = 5, 'Chrome OS' = 6, 'Other' = 7),
    `context_os_version` String,
    `context_page_path` String,
    `context_page_referrer` String,
    `context_page_search` String,
    `context_page_title` String,
    `context_page_url` String,
    `context_referrer_id` String,
    `context_referrer_type` String,
    `context_screen_width` Int16,
    `context_screen_height` Int16,
    `context_screen_density` Decimal(3,2),
    `context_session_id` Int64,
    `context_session_start` Bool,
    `context_timezone` String,
    `context_userAgent` String,
    `event` String,
    `groupId` String,
    `messageId` String,
    `name` String,
    `properties` String,
    `receivedAt` DateTime,
    `sentAt` DateTime,
    `source` Int32,
    `timestamp` DateTime,
    `traits` String,
    `type` Enum8('alias' = 1, 'anonymize' = 2, 'identify' = 3, 'group' = 4, 'page' = 5, 'screen' = 6, 'track' = 7),
    `userId` String
)
ENGINE = MergeTree
PRIMARY KEY (source, date, user_id, timestamp)
ORDER BY (source, date, user_id, timestamp)
SETTINGS index_granularity = 8192;
