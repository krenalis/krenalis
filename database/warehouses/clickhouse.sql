-- The "CREATE TABLE" statements in this file must have the same style of the
-- queries obtained by running "SHOW CREATE TABLE" on ClickHouse (except for the
-- database name on the query, which should be omitted in this file).

-- TODO: add the CREATE TABLE for "users_identities".

CREATE TABLE users
(
    `_id` Int32,
    `dummy_id` String,
    `anonymous_id` String,
    "android_id" String,
    "android_idfa" String,
    "android_push_token" String,
    "ios_id" String,
    "ios_idfa" String,
    "ios_push_token" String,
    `first_name` String,
    `last_name` String,
    `email` String,
    `gender` Enum8('male' = 1, 'female' = 2, 'other' = 3),
    `food_preferences_drink` Enum8('water' = 1, 'wine' = 2, 'beer' = 3, 'tea' = 4),
    `food_preferences_fruit` Enum8('apple' = 1, 'orange' = 2, 'mango' = 3, 'peach' = 4, 'lime' = 5),
    `phone_numbers` Array(String)
)
ENGINE = MergeTree
PRIMARY KEY `_id`
ORDER BY `_id`
SETTINGS index_granularity = 8192;

-- TODO: add the CREATE TABLE for "groups_identities".

CREATE TABLE groups
(
    `id` Int32,
)
ENGINE = MergeTree
PRIMARY KEY id
ORDER BY id
SETTINGS index_granularity = 8192;
