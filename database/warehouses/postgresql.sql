
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
    "creation_time"         timestamp NOT NULL DEFAULT now(),
    "timestamp"             timestamp NOT NULL DEFAULT now(),
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

CREATE TABLE groups (
    id                      SERIAL,
    "creation_time"         timestamp NOT NULL DEFAULT now(),
    "timestamp"             timestamp NOT NULL DEFAULT now(),
    PRIMARY KEY (id)
);
