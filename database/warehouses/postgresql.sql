
CREATE TYPE gender AS ENUM ('male', 'female', 'other');
CREATE TYPE drink AS ENUM ('water', 'wine', 'beer', 'tea');
CREATE TYPE fruit AS ENUM ('apple', 'orange', 'mango', 'peach', 'lime');

CREATE TABLE users (
    id                      SERIAL,
    "FirstName"             varchar(300),
    "LastName"              varchar(300),
    "Email"                 varchar(300),
    "Gender"                gender,
    "FoodPreferences_Drink" drink,
    "FoodPreferences_Fruit" fruit,
    "PhoneNumbers"          varchar(300)[],
    PRIMARY KEY (id)
);

CREATE TYPE event_os_name AS ENUM ('Android', 'Windows', 'iOS', 'macOS', 'Linux', 'Chrome OS', 'Other');
CREATE TYPE event_browser_name AS ENUM ('Chrome', 'Safari', 'Edge', 'Firefox', 'Samsung Internet', 'Opera', 'Other');
CREATE TYPE event_device_type AS ENUM ('desktop', 'tablet', 'mobile');

CREATE TABLE events (
    source integer,
    anonymous_id uuid,
    user_id varchar,
    date date,
    timestamp timestamp(3),
    sent_at timestamp(3),
    received_at timestamp(3),
    ip inet,
    os_name event_os_name,
    os_version varchar,
    user_agent varchar,
    browser_name event_browser_name,
    browser_other varchar,
    browser_version varchar,
    location_city varchar,
    location_country_code char(2),
    location_country_name varchar,
    location_latitude double precision,
    location_longitude double precision,
    device_type event_device_type,
    event varchar,
    language char(2),
    page_path varchar,
    page_referrer varchar,
    page_title varchar,
    page_url varchar,
    page_search varchar,
    target varchar,
    text varchar
)
