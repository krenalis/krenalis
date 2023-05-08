-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

CREATE TABLE users
(
    `id` Int32,
    `FirstName` String,
    `LastName` String,
    `Email` String,
    `Gender` Enum8('male' = 1, 'female' = 2, 'other' = 3),
    `FoodPreferences_Drink` Enum8('water' = 1, 'wine' = 2, 'beer' = 3, 'tea' = 4),
    `FoodPreferences_Fruit` Enum8('apple' = 1, 'orange' = 2, 'mango' = 3, 'peach' = 4, 'lime' = 5),
    `PhoneNumbers` Array(String),
    `timestamp` DateTime DEFAULT now()
)
ENGINE = MergeTree
PRIMARY KEY id
ORDER BY id
SETTINGS index_granularity = 8192;

