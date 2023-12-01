
-- TODO: add the CREATE TABLE for "users_identities".

CREATE TABLE "users" (
    "id"                     NUMBER,
    "dummy_id"               VARCHAR,
    "anonymous_id"           VARCHAR,
    "android_id"             VARCHAR,
    "android_idfa"           VARCHAR,
    "android_push_token"     VARCHAR,
    "ios_id"                 VARCHAR,
    "ios_idfa"               VARCHAR,
    "ios_push_token"         VARCHAR,
    "first_name"             VARCHAR(300),
    "last_name"              VARCHAR(300),
    "email"                  VARCHAR(300),
    "gender"                 VARCHAR,
    "food_preferences_drink" VARCHAR,
    "food_preferences_fruit" VARCHAR,
    "phone_numbers"          ARRAY,
    "favorite_movie"         OBJECT,
    PRIMARY KEY ("id")
);

-- TODO: add the CREATE TABLE for "groups_identities".

CREATE TABLE "groups" (
    "id"            NUMBER,
    "creation_time" TIMESTAMPNTZ NOT NULL,
    "timestamp"     TIMESTAMPNTZ NOT NULL,
    PRIMARY KEY ("id")
);

-- Keep in sync with the apis.eventColumns and events.eventsMergeTable variables.

CREATE TABLE "events" (
    "gid" NUMBER NOT NULL,
    "anonymous_id" VARCHAR NOT NULL,
    "category" VARCHAR NOT NULL,
    "app_name" VARCHAR NOT NULL,
    "app_version" VARCHAR NOT NULL,
    "app_build" VARCHAR NOT NULL,
    "app_namespace" VARCHAR NOT NULL,
    "browser_name" VARCHAR NOT NULL, -- "Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"
    "browser_other" VARCHAR NOT NULL,
    "browser_version" VARCHAR NOT NULL,
    "campaign_name" VARCHAR NOT NULL,
    "campaign_source" VARCHAR NOT NULL,
    "campaign_medium" VARCHAR NOT NULL,
    "campaign_term" VARCHAR NOT NULL,
    "campaign_content" VARCHAR NOT NULL,
    "device_id" VARCHAR NOT NULL,
    "device_advertising_id" VARCHAR NOT NULL,
    "device_ad_tracking_enabled" BOOLEAN NOT NULL,
    "device_manufacturer" VARCHAR NOT NULL,
    "device_model" VARCHAR NOT NULL,
    "device_name" VARCHAR NOT NULL,
    "device_type" VARCHAR NOT NULL,
    "device_token" VARCHAR NOT NULL,
    "ip" VARCHAR NOT NULL,
    "library_name" VARCHAR NOT NULL,
    "library_version" VARCHAR NOT NULL,
    "locale" VARCHAR(5) NOT NULL,
    "location_city" VARCHAR NOT NULL,
    "location_country" VARCHAR NOT NULL,
    "location_latitude" FLOAT NOT NULL,
    "location_longitude" FLOAT NOT NULL,
    "location_speed" FLOAT NOT NULL,
    "network_bluetooth" BOOLEAN NOT NULL,
    "network_carrier" VARCHAR NOT NULL,
    "network_cellular" BOOLEAN NOT NULL,
    "network_wifi" BOOLEAN NOT NULL,
    "os_name" VARCHAR NOT NULL, -- "Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"
    "os_version" VARCHAR NOT NULL,
    "page_path" VARCHAR NOT NULL,
    "page_referrer" VARCHAR NOT NULL,
    "page_search" VARCHAR NOT NULL,
    "page_title" VARCHAR NOT NULL,
    "page_url" VARCHAR NOT NULL,
    "referrer_id" VARCHAR NOT NULL,
    "referrer_type" VARCHAR NOT NULL,
    "screen_width" SMALLINT NOT NULL,
    "screen_height" SMALLINT NOT NULL,
    "screen_density" NUMBER(3,2) NOT NULL,
    "session_id" BIGINT NOT NULL,
    "session_start" BOOLEAN NOT NULL,
    "timezone" VARCHAR NOT NULL,
    "user_agent" VARCHAR NOT NULL,
    "event" VARCHAR NOT NULL,
    "group_id" VARCHAR NOT NULL,
    "message_id" VARCHAR NOT NULL,
    "name" VARCHAR NOT NULL,
    "properties" OBJECT NOT NULL,
    "received_at" DATETIME NOT NULL,
    "sent_at" DATETIME(3) NOT NULL,
    "source" integer NOT NULL,
    "timestamp" DATETIME(3) NOT NULL,
    "traits" OBJECT NOT NULL,
    "type" VARCHAR NOT NULL, -- "alias", "identify", "group", "page", "screen", "track"
    "user_id" VARCHAR NOT NULL,
    PRIMARY KEY ("message_id")
)
