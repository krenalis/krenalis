--
-- SPDX-License-Identifier: Elastic-2.0
--
--
-- Copyright (c) 2023 Open2b
--

-- Keep in sync with the schema at "apis/events/schema.go".

CREATE TYPE event_os_name AS ENUM ('Android', 'Windows', 'iOS', 'macOS', 'Linux', 'Chrome OS', 'Other');
CREATE TYPE event_browser_name AS ENUM ('Chrome', 'Safari', 'Edge', 'Firefox', 'Samsung Internet', 'Opera', 'Other');

CREATE TABLE events (
    source integer NOT NULL,
    event varchar NOT NULL,
    message_id varchar NOT NULL,
    anonymous_id uuid NOT NULL,
    user_id varchar NOT NULL,
    date date NOT NULL,
    timestamp timestamp(3) NOT NULL,
    sent_at timestamp(3) NOT NULL,
    received_at timestamp(3) NOT NULL,
    ip inet NOT NULL,
    network_cellular boolean NOT NULL,
    network_wifi boolean NOT NULL,
    network_bluetooth boolean NOT NULL,
    network_carrier varchar NOT NULL,
    os_name event_os_name NOT NULL,
    os_version varchar NOT NULL,
    app_name varchar NOT NULL,
    app_version varchar NOT NULL,
    app_build varchar NOT NULL,
    app_namespace varchar NOT NULL,
    screen_density smallint NOT NULL,
    screen_width smallint NOT NULL,
    screen_height smallint NOT NULL,
    user_agent varchar NOT NULL,
    browser_name event_browser_name NOT NULL,
    browser_other varchar NOT NULL,
    browser_version varchar NOT NULL,
    device_id varchar NOT NULL,
    device_name varchar NOT NULL,
    device_manufacturer varchar NOT NULL,
    device_model varchar NOT NULL,
    device_type varchar NOT NULL,
    device_version varchar NOT NULL,
    device_advertising_id varchar NOT NULL,
    location_city varchar NOT NULL,
    location_country varchar NOT NULL,
    location_region varchar NOT NULL,
    location_latitude double precision NOT NULL,
    location_longitude double precision NOT NULL,
    location_speed double precision NOT NULL,
    locale varchar(5) NOT NULL,
    timezone varchar NOT NULL,
    page_url varchar NOT NULL,
    page_path varchar NOT NULL,
    page_search varchar NOT NULL,
    page_hash varchar NOT NULL,
    page_title varchar NOT NULL,
    page_referrer varchar NOT NULL,
    referrer_type varchar NOT NULL,
    referrer_name varchar NOT NULL,
    referrer_url varchar NOT NULL,
    referrer_link varchar NOT NULL,
    campaign_name varchar NOT NULL,
    campaign_source varchar NOT NULL,
    campaign_medium varchar NOT NULL,
    campaign_term varchar NOT NULL,
    campaign_content varchar NOT NULL,
    library_name varchar NOT NULL,
    library_version varchar NOT NULL,
    properties jsonb NOT NULL
)
