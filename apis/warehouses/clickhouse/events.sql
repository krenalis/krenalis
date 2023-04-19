--
-- SPDX-License-Identifier: Elastic-2.0
--
--
-- Copyright (c) 2023 Open2b
--

-- Keep in sync with the schema at "apis/events/schema.go".

CREATE TABLE events
(
    `source` Int32,
    `event` String,
    `message_id` String,
    `anonymous_id` UUID,
    `user_id` String,
    `date` Date,
    `timestamp` DateTime,
    `sent_at` DateTime,
    `received_at` DateTime,
    `ip` IPv6,
    `network_cellular` Bool,
    `network_wifi` Bool,
    `network_bluetooth` Bool,
    `network_carrier` String,
    `os_name` Enum8('Android' = 1, 'Windows' = 2, 'iOS' = 3, 'macOS' = 4, 'Linux' = 5, 'Chrome OS' = 6, 'Other' = 7),
    `os_version` String,
    `app_name` String,
    `app_version` String,
    `app_build` String,
    `app_namespace` String,
    `screen_density` Int16,
    `screen_width` Int16,
    `screen_height` Int16,
    `user_agent` String,
    `browser_name` Enum8('Chrome' = 1, 'Safari' = 2, 'Edge' = 3, 'Firefox' = 4, 'Samsung Internet' = 5, 'Opera' = 6, 'Other' = 7),
    `browser_other` String,
    `browser_version` String,
    `device_id` String,
    `device_name` String,
    `device_manufacturer` String,
    `device_model` String,
    `device_type` String,
    `device_version` String,
    `device_advertising_id` String,
    `location_city` String,
    `location_country` String,
    `location_region` String,
    `location_latitude` Float64,
    `location_longitude` Float64,
    `location_speed` Float64,
    `locale` String,
    `timezone` String,
    `page_url` String,
    `page_path` String,
    `page_search` String,
    `page_hash` String,
    `page_title` String,
    `page_referrer` String,
    `referrer_type` String,
    `referrer_name` String,
    `referrer_url` String,
    `referrer_link` String,
    `campaign_name` String,
    `campaign_source` String,
    `campaign_medium` String,
    `campaign_term` String,
    `campaign_content` String,
    `library_name` String,
    `library_version` String,
    `properties` String
)
ENGINE = MergeTree
PRIMARY KEY (source, date, user_id, timestamp)
ORDER BY (source, date, user_id, timestamp)
SETTINGS index_granularity = 8192;
