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
    `screen_density` UInt16,
    `screen_width` UInt16,
    `screen_height` UInt16,
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
    `utm_source` String,
    `utm_medium` String,
    `utm_campaign` String,
    `utm_term` String,
    `utm_content` String,
    `target` String,
    `text` String,
    `properties` String
)
ENGINE = MergeTree
PRIMARY KEY (source, date, user_id, timestamp)
ORDER BY (source, date, user_id, timestamp)
SETTINGS index_granularity = 8192;
