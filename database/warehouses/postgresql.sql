
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
