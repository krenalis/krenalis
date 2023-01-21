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

CREATE TYPE event_os_name AS ENUM ('Android', 'Windows', 'iOS', 'MacOS', 'Linux', 'ChromeOS', 'Other');
CREATE TYPE event_browser_name AS ENUM ('Chrome', 'Safari', 'Edge', 'Firefox', 'SamsungInternet', 'Opera', 'Other');
CREATE TYPE event_device_type AS ENUM ('desktop', 'tablet', 'mobile');

CREATE TABLE events (
    source integer,
    date date,
    timestamp timestamp,
    os_name event_os_name,
    os_version varchar,
    browser_name event_browser_name,
    browser_other varchar,
    browser_version varchar,
    device_type event_device_type,
    event varchar,
    language char(2),
    referrer varchar,
    target varchar,
    text varchar,
    title varchar,
    domain varchar,
    path varchar,
    query_string varchar,
    "user" varchar,
    country char(2),
    city varchar
)