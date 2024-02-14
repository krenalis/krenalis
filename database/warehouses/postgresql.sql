
CREATE TYPE gender AS ENUM ('male', 'female', 'other');
CREATE TYPE drink AS ENUM ('water', 'wine', 'beer', 'tea');
CREATE TYPE fruit AS ENUM ('apple', 'orange', 'mango', 'peach', 'lime');

CREATE TYPE music_genre AS ENUM('pop', 'rock', 'blues', 'classical');

CREATE TABLE users_identities (

    "_identity_id"   serial,
    "_connection"    int NOT NULL DEFAULT 0,
    "_external_id"   text NOT NULL DEFAULT '',
    "_anonymous_ids" text[],
    "_updated_at"    timestamp NOT NULL,
    "_gid"           int NOT NULL DEFAULT 0,

    "_business_id_value"  varchar(40) NOT NULL DEFAULT '',
    "_business_id_label"  varchar(16) NOT NULL DEFAULT '',
    
    "__cluster__"       serial,

    "dummy_id"                          text,
    "anonymous_id"                      text,
    "android_id"                        text,
    "android_idfa"                      text,
    "android_push_token"                text,
    "ios_id"                            text,
    "ios_idfa"                          text,
    "ios_push_token"                    text,
    "first_name"                        varchar(300),
    "last_name"                         varchar(300),
    "email"                             varchar(300),
    "gender"                            gender,    
    "food_preferences_drink"            drink,
    "food_preferences_fruit"            fruit,
    "phone_numbers"                     varchar(300)[],
    "favorite_movie_title"              text,
    "favorite_movie_length"             double precision,
    "favorite_movie_soundtrack_title"   text,
    "favorite_movie_soundtrack_author"  text,
    "favorite_movie_soundtrack_length"  double precision,
    "favorite_movie_soundtrack_genre"   music_genre,
    
    PRIMARY KEY ("_identity_id")
);

CREATE TABLE users (

    "_id" SERIAL,
    
    "__identity_ids__" int[],

    "dummy_id"                          text,
    "anonymous_id"                      text,
    "android_id"                        text,
    "android_idfa"                      text,
    "android_push_token"                text,
    "ios_id"                            text,
    "ios_idfa"                          text,
    "ios_push_token"                    text,
    "first_name"                        varchar(300),
    "last_name"                         varchar(300),
    "email"                             varchar(300),
    "gender"                            gender,    
    "food_preferences_drink"            drink,
    "food_preferences_fruit"            fruit,
    "phone_numbers"                     varchar(300)[],
    "favorite_movie_title"              text,
    "favorite_movie_length"             double precision,
    "favorite_movie_soundtrack_title"   text,
    "favorite_movie_soundtrack_author"  text,
    "favorite_movie_soundtrack_length"  double precision,
    "favorite_movie_soundtrack_genre"   music_genre,
    
    PRIMARY KEY ("_id")
);

CREATE TABLE groups_identities (
    id              SERIAL,
    "creation_time" timestamp NOT NULL DEFAULT now(),
    "timestamp"     timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);

CREATE TABLE groups (
    id SERIAL,
    PRIMARY KEY (id)
);

-- Keep in sync with the events.eventsMergeTable variable.

CREATE TYPE event_os_name AS ENUM ('None', 'Android', 'Windows', 'iOS', 'macOS', 'Linux', 'Chrome OS', 'Other');
CREATE TYPE event_browser_name AS ENUM ('None', 'Chrome', 'Safari', 'Edge', 'Firefox', 'Samsung Internet', 'Opera', 'Other');
CREATE TYPE event_type AS ENUM ('alias', 'anonymize', 'identify', 'group', 'page', 'screen', 'track');

CREATE TABLE events (
    "gid" integer NOT NULL DEFAULT 0,
    "anonymous_id" varchar NOT NULL,
    "category" varchar NOT NULL,
    "context_app_name" varchar NOT NULL,
    "context_app_version" varchar NOT NULL,
    "context_app_build" varchar NOT NULL,
    "context_app_namespace" varchar NOT NULL,
    "context_browser_name" event_browser_name NOT NULL,
    "context_browser_other" varchar NOT NULL,
    "context_browser_version" varchar NOT NULL,
    "context_campaign_name" varchar NOT NULL,
    "context_campaign_source" varchar NOT NULL,
    "context_campaign_medium" varchar NOT NULL,
    "context_campaign_term" varchar NOT NULL,
    "context_campaign_content" varchar NOT NULL,
    "context_device_id" varchar NOT NULL,
    "context_device_advertising_id" varchar NOT NULL,
    "context_device_ad_tracking_enabled" boolean NOT NULL,
    "context_device_manufacturer" varchar NOT NULL,
    "context_device_model" varchar NOT NULL,
    "context_device_name" varchar NOT NULL,
    "context_device_type" varchar NOT NULL,
    "context_device_token" varchar NOT NULL,
    "context_ip" inet NOT NULL,
    "context_library_name" varchar NOT NULL,
    "context_library_version" varchar NOT NULL,
    "context_locale" varchar(5) NOT NULL,
    "context_location_city" varchar NOT NULL,
    "context_location_country" varchar NOT NULL,
    "context_location_latitude" double precision NOT NULL,
    "context_location_longitude" double precision NOT NULL,
    "context_location_speed" double precision NOT NULL,
    "context_network_bluetooth" boolean NOT NULL,
    "context_network_carrier" varchar NOT NULL,
    "context_network_cellular" boolean NOT NULL,
    "context_network_wifi" boolean NOT NULL,
    "context_os_name" event_os_name NOT NULL,
    "context_os_version" varchar NOT NULL,
    "context_page_path" varchar NOT NULL,
    "context_page_referrer" varchar NOT NULL,
    "context_page_search" varchar NOT NULL,
    "context_page_title" varchar NOT NULL,
    "context_page_url" varchar NOT NULL,
    "context_referrer_id" varchar NOT NULL,
    "context_referrer_type" varchar NOT NULL,
    "context_screen_width" smallint NOT NULL,
    "context_screen_height" smallint NOT NULL,
    "context_screen_density" NUMERIC(3,2) NOT NULL,
    "context_session_id" bigint NOT NULL,
    "context_session_start" boolean NOT NULL,
    "context_timezone" varchar NOT NULL,
    "context_user_agent" varchar NOT NULL,
    "event" varchar NOT NULL,
    "group_id" varchar NOT NULL,
    "message_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "properties" jsonb NOT NULL,
    "received_at" timestamp(3) NOT NULL,
    "sent_at" timestamp(3) NOT NULL,
    "source" integer NOT NULL,
    "timestamp" timestamp(3) NOT NULL,
    "traits" jsonb NOT NULL,
    "type" event_type NOT NULL,
    "user_id" varchar NOT NULL,
    PRIMARY KEY ("message_id")
)
