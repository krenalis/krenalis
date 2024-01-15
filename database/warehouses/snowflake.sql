
-- TODO: add the CREATE TABLE for "users_identities".

CREATE TABLE "users" (
    "_id"                    NUMBER,
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
    PRIMARY KEY ("_id")
);

-- TODO: add the CREATE TABLE for "groups_identities".

CREATE TABLE "groups" (
    "id"            NUMBER,
    "creation_time" TIMESTAMPNTZ NOT NULL,
    "timestamp"     TIMESTAMPNTZ NOT NULL,
    PRIMARY KEY ("id")
);

-- Keep in sync with the events.eventsMergeTable variable.

CREATE TABLE "events" (
    "gid" NUMBER NOT NULL,
    "anonymousId" VARCHAR NOT NULL,
    "category" VARCHAR NOT NULL,
    "context_app_name" VARCHAR NOT NULL,
    "context_app_version" VARCHAR NOT NULL,
    "context_app_build" VARCHAR NOT NULL,
    "context_app_namespace" VARCHAR NOT NULL,
    "context_browser_name" VARCHAR NOT NULL, -- "Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"
    "context_browser_other" VARCHAR NOT NULL,
    "context_browser_version" VARCHAR NOT NULL,
    "context_campaign_name" VARCHAR NOT NULL,
    "context_campaign_source" VARCHAR NOT NULL,
    "context_campaign_medium" VARCHAR NOT NULL,
    "context_campaign_term" VARCHAR NOT NULL,
    "context_campaign_content" VARCHAR NOT NULL,
    "context_device_id" VARCHAR NOT NULL,
    "context_device_advertisingId" VARCHAR NOT NULL,
    "context_device_adTrackingEnabled" BOOLEAN NOT NULL,
    "context_device_manufacturer" VARCHAR NOT NULL,
    "context_device_model" VARCHAR NOT NULL,
    "context_device_name" VARCHAR NOT NULL,
    "context_device_type" VARCHAR NOT NULL,
    "context_device_token" VARCHAR NOT NULL,
    "context_ip" VARCHAR NOT NULL,
    "context_library_name" VARCHAR NOT NULL,
    "context_library_version" VARCHAR NOT NULL,
    "context_locale" VARCHAR(5) NOT NULL,
    "context_location_city" VARCHAR NOT NULL,
    "context_location_country" VARCHAR NOT NULL,
    "context_location_latitude" FLOAT NOT NULL,
    "context_location_longitude" FLOAT NOT NULL,
    "context_location_speed" FLOAT NOT NULL,
    "context_network_bluetooth" BOOLEAN NOT NULL,
    "context_network_carrier" VARCHAR NOT NULL,
    "context_network_cellular" BOOLEAN NOT NULL,
    "context_network_wifi" BOOLEAN NOT NULL,
    "context_os_name" VARCHAR NOT NULL, -- "Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"
    "context_os_version" VARCHAR NOT NULL,
    "context_page_path" VARCHAR NOT NULL,
    "context_page_referrer" VARCHAR NOT NULL,
    "context_page_search" VARCHAR NOT NULL,
    "context_page_title" VARCHAR NOT NULL,
    "context_page_url" VARCHAR NOT NULL,
    "context_referrer_id" VARCHAR NOT NULL,
    "context_referrer_type" VARCHAR NOT NULL,
    "context_screen_width" SMALLINT NOT NULL,
    "context_screen_height" SMALLINT NOT NULL,
    "context_screen_density" NUMBER(3,2) NOT NULL,
    "context_session_id" BIGINT NOT NULL,
    "context_session_start" BOOLEAN NOT NULL,
    "context_timezone" VARCHAR NOT NULL,
    "context_userAgent" VARCHAR NOT NULL,
    "event" VARCHAR NOT NULL,
    "groupId" VARCHAR NOT NULL,
    "messageId" VARCHAR NOT NULL,
    "name" VARCHAR NOT NULL,
    "properties" OBJECT NOT NULL,
    "receivedAt" DATETIME NOT NULL,
    "sentAt" DATETIME(3) NOT NULL,
    "source" integer NOT NULL,
    "timestamp" DATETIME(3) NOT NULL,
    "traits" OBJECT NOT NULL,
    "type" VARCHAR NOT NULL, -- "alias", "identify", "group", "page", "screen", "track"
    "userId" VARCHAR NOT NULL,
    PRIMARY KEY ("messageId")
)
