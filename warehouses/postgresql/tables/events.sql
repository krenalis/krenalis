-- Keep in sync with the events.eventsMergeTable variable.

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = 'event_os_name') THEN
        CREATE TYPE event_os_name AS ENUM ('Android', 'Windows', 'iOS', 'macOS', 'Linux', 'Chrome OS', 'Other');
    END IF;
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = 'event_browser_name') THEN
        CREATE TYPE event_browser_name AS ENUM ('Chrome', 'Safari', 'Edge', 'Firefox', 'Samsung Internet', 'Opera', 'Other');
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
    "channel" character varying,
    "category" character varying,
    "context_app_name" character varying,
    "context_app_version" character varying,
    "context_app_build" character varying,
    "context_app_namespace" character varying,
    "context_browser_name" event_browser_name,
    "context_browser_other" character varying,
    "context_browser_version" character varying,
    "context_campaign_name" character varying,
    "context_campaign_source" character varying,
    "context_campaign_medium" character varying,
    "context_campaign_term" character varying,
    "context_campaign_content" character varying,
    "context_device_id" character varying,
    "context_device_advertising_id" character varying,
    "context_device_ad_tracking_enabled" boolean,
    "context_device_manufacturer" character varying,
    "context_device_model" character varying,
    "context_device_name" character varying,
    "context_device_type" character varying,
    "context_device_token" character varying,
    "context_ip" inet,
    "context_library_name" character varying,
    "context_library_version" character varying,
    "context_locale" character varying(5),
    "context_location_city" character varying,
    "context_location_country" character varying,
    "context_location_latitude" double precision,
    "context_location_longitude" double precision,
    "context_location_speed" double precision,
    "context_network_bluetooth" boolean,
    "context_network_carrier" character varying,
    "context_network_cellular" boolean,
    "context_network_wifi" boolean,
    "context_os_name" event_os_name,
    "context_os_other" character varying,
    "context_os_version" character varying,
    "context_page_path" character varying,
    "context_page_referrer" character varying,
    "context_page_search" character varying,
    "context_page_title" character varying,
    "context_page_url" character varying,
    "context_referrer_id" character varying,
    "context_referrer_type" character varying,
    "context_screen_width" smallint,
    "context_screen_height" smallint,
    "context_screen_density" NUMERIC(3,2),
    "context_session_id" bigint,
    "context_session_start" boolean,
    "context_timezone" character varying,
    "context_user_agent" character varying,
    "event" character varying,
    "group_id" character varying,
    "message_id" character varying NOT NULL,
    "name" character varying,
    "properties" jsonb,
    "received_at" timestamp(3) NOT NULL,
    "sent_at" timestamp(3) NOT NULL,
    "timestamp" timestamp(3) NOT NULL,
    "traits" jsonb NOT NULL,
    "type" event_type NOT NULL,
    "user_id" character varying,
    PRIMARY KEY ("id")
)
