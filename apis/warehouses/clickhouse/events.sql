--
-- SPDX-License-Identifier: Elastic-2.0
--
--
-- Copyright (c) 2023 Open2b
--

-- Keep in sync with the schema at "apis/events/schema.go".

CREATE TABLE events
(
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
    `screen_density` Int16,
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
    `type` String,
    `user_id` String
)
ENGINE = MergeTree
PRIMARY KEY (source, date, user_id, timestamp)
ORDER BY (source, date, user_id, timestamp)
SETTINGS index_granularity = 8192;
