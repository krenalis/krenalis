-- Keep in sync with the events.eventsMergeTable variable.

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = 'event_os_name') THEN
        CREATE TYPE event_os_name AS ENUM ('None', 'Android', 'Windows', 'iOS', 'macOS', 'Linux', 'Chrome OS', 'Other');
    END IF;
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = 'event_browser_name') THEN
        CREATE TYPE event_browser_name AS ENUM ('None', 'Chrome', 'Safari', 'Edge', 'Firefox', 'Samsung Internet', 'Opera', 'Other');
    END IF;
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = 'event_type') THEN
        CREATE TYPE event_type AS ENUM ('alias', 'identify', 'group', 'page', 'screen', 'track');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS events (
    "id" UUID NOT NULL,
    "user" UUID,
    "connection" integer NOT NULL,
    "anonymous_id" character varying NOT NULL,
    "channel" character varying NOT NULL,
    "category" character varying NOT NULL,
    "context_app_name" character varying NOT NULL,
    "context_app_version" character varying NOT NULL,
    "context_app_build" character varying NOT NULL,
    "context_app_namespace" character varying NOT NULL,
    "context_browser_name" event_browser_name NOT NULL,
    "context_browser_other" character varying NOT NULL,
    "context_browser_version" character varying NOT NULL,
    "context_campaign_name" character varying NOT NULL,
    "context_campaign_source" character varying NOT NULL,
    "context_campaign_medium" character varying NOT NULL,
    "context_campaign_term" character varying NOT NULL,
    "context_campaign_content" character varying NOT NULL,
    "context_device_id" character varying NOT NULL,
    "context_device_advertising_id" character varying NOT NULL,
    "context_device_ad_tracking_enabled" boolean NOT NULL,
    "context_device_manufacturer" character varying NOT NULL,
    "context_device_model" character varying NOT NULL,
    "context_device_name" character varying NOT NULL,
    "context_device_type" character varying NOT NULL,
    "context_device_token" character varying NOT NULL,
    "context_ip" inet NOT NULL,
    "context_library_name" character varying NOT NULL,
    "context_library_version" character varying NOT NULL,
    "context_locale" character varying(5) NOT NULL,
    "context_location_city" character varying NOT NULL,
    "context_location_country" character varying NOT NULL,
    "context_location_latitude" double precision NOT NULL,
    "context_location_longitude" double precision NOT NULL,
    "context_location_speed" double precision NOT NULL,
    "context_network_bluetooth" boolean NOT NULL,
    "context_network_carrier" character varying NOT NULL,
    "context_network_cellular" boolean NOT NULL,
    "context_network_wifi" boolean NOT NULL,
    "context_os_name" event_os_name NOT NULL,
    "context_os_version" character varying NOT NULL,
    "context_page_path" character varying NOT NULL,
    "context_page_referrer" character varying NOT NULL,
    "context_page_search" character varying NOT NULL,
    "context_page_title" character varying NOT NULL,
    "context_page_url" character varying NOT NULL,
    "context_referrer_id" character varying NOT NULL,
    "context_referrer_type" character varying NOT NULL,
    "context_screen_width" smallint NOT NULL,
    "context_screen_height" smallint NOT NULL,
    "context_screen_density" NUMERIC(3,2) NOT NULL,
    "context_session_id" bigint NOT NULL,
    "context_session_start" boolean NOT NULL,
    "context_timezone" character varying NOT NULL,
    "context_user_agent" character varying NOT NULL,
    "event" character varying NOT NULL,
    "group_id" character varying NOT NULL,
    "message_id" character varying NOT NULL,
    "name" character varying NOT NULL,
    "properties" jsonb NOT NULL,
    "received_at" timestamp(3) NOT NULL,
    "sent_at" timestamp(3) NOT NULL,
    "timestamp" timestamp(3) NOT NULL,
    "traits" jsonb NOT NULL,
    "type" event_type NOT NULL,
    "user_id" character varying NOT NULL,
    PRIMARY KEY ("id")
)