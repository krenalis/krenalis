
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

CREATE TABLE users (
    id                      SERIAL,
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

CREATE TYPE event_os_name AS ENUM ('Android', 'Windows', 'iOS', 'macOS', 'Linux', 'Chrome OS', 'Other');
CREATE TYPE event_browser_name AS ENUM ('Chrome', 'Safari', 'Edge', 'Firefox', 'Samsung Internet', 'Opera', 'Other');
CREATE TYPE event_device_type AS ENUM ('desktop', 'tablet', 'mobile');

CREATE TABLE events (
    source integer NOT NULL,
    anonymous_id uuid NOT NULL,
    user_id varchar NOT NULL,
    date date NOT NULL,
    timestamp timestamp(3) NOT NULL,
    sent_at timestamp(3) NOT NULL,
    received_at timestamp(3) NOT NULL,
    ip inet NOT NULL,
    os_name event_os_name NOT NULL,
    os_version varchar NOT NULL,
    user_agent varchar NOT NULL,
    screen_density smallint NOT NULL,
    screen_width smallint NOT NULL,
    screen_height smallint NOT NULL,
    browser_name event_browser_name NOT NULL,
    browser_other varchar NOT NULL,
    browser_version varchar NOT NULL,
    location_city varchar NOT NULL,
    location_country_code char(2) NOT NULL,
    location_country_name varchar NOT NULL,
    location_latitude double precision NOT NULL,
    location_longitude double precision NOT NULL,
    device_type event_device_type NOT NULL,
    event varchar NOT NULL,
    language char(2) NOT NULL,
    page_path varchar NOT NULL,
    page_referrer varchar NOT NULL,
    page_title varchar NOT NULL,
    page_url varchar NOT NULL,
    page_search varchar NOT NULL,
    utm_source varchar NOT NULL,
    utm_medium varchar NOT NULL,
    utm_campaign varchar NOT NULL,
    utm_term varchar NOT NULL,
    utm_content varchar NOT NULL,
    target varchar NOT NULL,
    text varchar NOT NULL,
    properties jsonb NOT NULL
)
