
CREATE TYPE gender AS ENUM ('male', 'female', 'other');
CREATE TYPE drink AS ENUM ('water', 'wine', 'beer', 'tea');
CREATE TYPE fruit AS ENUM ('apple', 'orange', 'mango', 'peach', 'lime');

CREATE TYPE music_genre AS ENUM('pop', 'rock', 'blues', 'classical');

CREATE TYPE music AS (
    "title" text,
    "author" text,
    "length" double precision,
    "genre" music_genre
);

CREATE TYPE movie AS (
    "title" text,
    "length" double precision,
    "soundtrack" music
);

CREATE TABLE users_identities (

    "__identity_id__"   serial,
    "__action__"        int,
    "__external_id__"   text,
    "__anonymous_ids__" text[],
    "__cluster__"       serial,
    "__timestamp__"     timestamp NOT NULL DEFAULT now(),

    -- User properties.
    "dummy_id"              text,
    "anonymous_id"          text,
    "android_id"            text,
    "android_idfa"          text,
    "android_push_token"    text,
    "ios_id"                text,
    "ios_idfa"              text,
    "ios_push_token"        text,
    "FirstName"             varchar(300),
    "LastName"              varchar(300),
    "Email"                 varchar(300),
    "Gender"                gender,
    "FoodPreferences_Drink" drink,
    "FoodPreferences_Fruit" fruit,
    "PhoneNumbers"          varchar(300)[],
    "FavouriteMovie"        movie,
    
    PRIMARY KEY ("__identity_id__")
);

CREATE TABLE users (
    id                      SERIAL,
    "dummy_id"              text,
    "anonymous_id"          text,
    "android_id"            text,
    "android_idfa"          text,
    "android_push_token"    text,
    "ios_id"                text,
    "ios_idfa"              text,
    "ios_push_token"        text,
    "FirstName"             varchar(300),
    "LastName"              varchar(300),
    "Email"                 varchar(300),
    "Gender"                gender,
    "FoodPreferences_Drink" drink,
    "FoodPreferences_Fruit" fruit,
    "PhoneNumbers"          varchar(300)[],
    "FavouriteMovie"        movie,
    PRIMARY KEY (id)
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

-- Keep in sync with the apis.eventColumns and events.batchEventsColumns variables.

CREATE TYPE event_os_name AS ENUM ('Android', 'Windows', 'iOS', 'macOS', 'Linux', 'Chrome OS', 'Other');
CREATE TYPE event_browser_name AS ENUM ('Chrome', 'Safari', 'Edge', 'Firefox', 'Samsung Internet', 'Opera', 'Other');
CREATE TYPE event_type AS ENUM ('alias', 'identify', 'group', 'page', 'screen', 'track');

CREATE TABLE events (
    gid integer NOT NULL,
    anonymous_id varchar NOT NULL,
    category varchar NOT NULL,
    app_name varchar NOT NULL,
    app_version varchar NOT NULL,
    app_build varchar NOT NULL,
    app_namespace varchar NOT NULL,
    browser_name event_browser_name NOT NULL,
    browser_other varchar NOT NULL,
    browser_version varchar NOT NULL,
    campaign_name varchar NOT NULL,
    campaign_source varchar NOT NULL,
    campaign_medium varchar NOT NULL,
    campaign_term varchar NOT NULL,
    campaign_content varchar NOT NULL,
    device_id varchar NOT NULL,
    device_advertising_id varchar NOT NULL,
    device_ad_tracking_enabled boolean NOT NULL,
    device_manufacturer varchar NOT NULL,
    device_model varchar NOT NULL,
    device_name varchar NOT NULL,
    device_type varchar NOT NULL,
    device_token varchar NOT NULL,
    ip inet NOT NULL,
    library_name varchar NOT NULL,
    library_version varchar NOT NULL,
    locale varchar(5) NOT NULL,
    location_city varchar NOT NULL,
    location_country varchar NOT NULL,
    location_latitude double precision NOT NULL,
    location_longitude double precision NOT NULL,
    location_speed double precision NOT NULL,
    network_bluetooth boolean NOT NULL,
    network_carrier varchar NOT NULL,
    network_cellular boolean NOT NULL,
    network_wifi boolean NOT NULL,
    os_name event_os_name NOT NULL,
    os_version varchar NOT NULL,
    page_path varchar NOT NULL,
    page_referrer varchar NOT NULL,
    page_search varchar NOT NULL,
    page_title varchar NOT NULL,
    page_url varchar NOT NULL,
    referrer_id varchar NOT NULL,
    referrer_type varchar NOT NULL,
    screen_width smallint NOT NULL,
    screen_height smallint NOT NULL,
    screen_density smallint NOT NULL,
    session_id bigint NOT NULL,
    session_start boolean NOT NULL,
    timezone varchar NOT NULL,
    user_agent varchar NOT NULL,
    event varchar NOT NULL,
    group_id varchar NOT NULL,
    message_id varchar NOT NULL,
    name varchar NOT NULL,
    properties jsonb NOT NULL,
    received_at timestamp(3) NOT NULL,
    sent_at timestamp(3) NOT NULL,
    source integer NOT NULL,
    timestamp timestamp(3) NOT NULL,
    traits jsonb NOT NULL,
    type event_type NOT NULL,
    user_id varchar NOT NULL,
    PRIMARY KEY (message_id)
)
