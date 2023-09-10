

CREATE TABLE "users" (
    "id"                    NUMBER,
    "creation_time"         TIMESTAMPNTZ NOT NULL,
    "timestamp"             TIMESTAMPNTZ NOT NULL,
    "dummy_id"              VARCHAR,
    "anonymous_id"          VARCHAR,
    "android_id"            VARCHAR,
    "android_idfa"          VARCHAR,
    "android_push_token"    VARCHAR,
    "ios_id"                VARCHAR,
    "ios_idfa"              VARCHAR,
    "ios_push_token"        VARCHAR,
    "FirstName"             VARCHAR(300),
    "LastName"              VARCHAR(300),
    "Email"                 VARCHAR(300),
    "Gender"                VARCHAR,
    "FoodPreferences_Drink" VARCHAR,
    "FoodPreferences_Fruit" VARCHAR,
    "PhoneNumbers"          ARRAY,
    "FavouriteMovie"        OBJECT,
    PRIMARY KEY ("id")
);

CREATE TABLE "groups" (
    "id"            NUMBER,
    "creation_time" TIMESTAMPNTZ NOT NULL,
    "timestamp"     TIMESTAMPNTZ NOT NULL,
    PRIMARY KEY ("id")
);
