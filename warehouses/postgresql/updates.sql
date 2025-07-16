--
-- SPDX-License-Identifier: Elastic-2.0
--
--
-- Copyright (c) 2025 Open2b
--
--
-- This file contains queries to run for updating the Meergo data warehouse
-- versions during development.

ALTER TABLE
    events
ADD
    COLUMN IF NOT EXISTS context_os_other VARCHAR NOT NULL DEFAULT '';

ALTER TABLE
    events
ALTER COLUMN
    context_os_other DROP DEFAULT;

ALTER TABLE events
    ALTER COLUMN "channel" DROP NOT NULL,
    ALTER COLUMN "category" DROP NOT NULL,
    ALTER COLUMN "context_app_name" DROP NOT NULL,
    ALTER COLUMN "context_app_version" DROP NOT NULL,
    ALTER COLUMN "context_app_build" DROP NOT NULL,
    ALTER COLUMN "context_app_namespace" DROP NOT NULL,
    ALTER COLUMN "context_browser_name" DROP NOT NULL,
    ALTER COLUMN "context_browser_other" DROP NOT NULL,
    ALTER COLUMN "context_browser_version" DROP NOT NULL,
    ALTER COLUMN "context_campaign_name" DROP NOT NULL,
    ALTER COLUMN "context_campaign_source" DROP NOT NULL,
    ALTER COLUMN "context_campaign_medium" DROP NOT NULL,
    ALTER COLUMN "context_campaign_term" DROP NOT NULL,
    ALTER COLUMN "context_campaign_content" DROP NOT NULL,
    ALTER COLUMN "context_device_id" DROP NOT NULL,
    ALTER COLUMN "context_device_advertising_id" DROP NOT NULL,
    ALTER COLUMN "context_device_ad_tracking_enabled" DROP NOT NULL,
    ALTER COLUMN "context_device_manufacturer" DROP NOT NULL,
    ALTER COLUMN "context_device_model" DROP NOT NULL,
    ALTER COLUMN "context_device_name" DROP NOT NULL,
    ALTER COLUMN "context_device_type" DROP NOT NULL,
    ALTER COLUMN "context_device_token" DROP NOT NULL,
    ALTER COLUMN "context_ip" DROP NOT NULL,
    ALTER COLUMN "context_library_name" DROP NOT NULL,
    ALTER COLUMN "context_library_version" DROP NOT NULL,
    ALTER COLUMN "context_locale" DROP NOT NULL,
    ALTER COLUMN "context_location_city" DROP NOT NULL,
    ALTER COLUMN "context_location_country" DROP NOT NULL,
    ALTER COLUMN "context_location_latitude" DROP NOT NULL,
    ALTER COLUMN "context_location_longitude" DROP NOT NULL,
    ALTER COLUMN "context_location_speed" DROP NOT NULL,
    ALTER COLUMN "context_network_bluetooth" DROP NOT NULL,
    ALTER COLUMN "context_network_carrier" DROP NOT NULL,
    ALTER COLUMN "context_network_cellular" DROP NOT NULL,
    ALTER COLUMN "context_network_wifi" DROP NOT NULL,
    ALTER COLUMN "context_os_name" DROP NOT NULL,
    ALTER COLUMN "context_os_other" DROP NOT NULL,
    ALTER COLUMN "context_os_version" DROP NOT NULL,
    ALTER COLUMN "context_page_path" DROP NOT NULL,
    ALTER COLUMN "context_page_referrer" DROP NOT NULL,
    ALTER COLUMN "context_page_search" DROP NOT NULL,
    ALTER COLUMN "context_page_title" DROP NOT NULL,
    ALTER COLUMN "context_page_url" DROP NOT NULL,
    ALTER COLUMN "context_referrer_id" DROP NOT NULL,
    ALTER COLUMN "context_referrer_type" DROP NOT NULL,
    ALTER COLUMN "context_screen_width" DROP NOT NULL,
    ALTER COLUMN "context_screen_height" DROP NOT NULL,
    ALTER COLUMN "context_screen_density" DROP NOT NULL,
    ALTER COLUMN "context_session_id" DROP NOT NULL,
    ALTER COLUMN "context_session_start" DROP NOT NULL,
    ALTER COLUMN "context_timezone" DROP NOT NULL,
    ALTER COLUMN "context_user_agent" DROP NOT NULL,
    ALTER COLUMN "event" DROP NOT NULL,
    ALTER COLUMN "group_id" DROP NOT NULL,
    ALTER COLUMN "name" DROP NOT NULL,
    ALTER COLUMN "properties" DROP NOT NULL,
    ALTER COLUMN "user_id" DROP NOT NULL;

ALTER TABLE
    events
ADD
    COLUMN IF NOT EXISTS "previous_id" character varying;
