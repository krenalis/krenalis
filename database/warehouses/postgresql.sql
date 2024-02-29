CREATE TYPE gender AS ENUM ('male', 'female', 'other');
CREATE TYPE drink AS ENUM ('water', 'wine', 'beer', 'tea');
CREATE TYPE fruit AS ENUM ('apple', 'orange', 'mango', 'peach', 'lime');
CREATE TYPE music_genre AS ENUM('pop', 'rock', 'blues', 'classical');

ALTER TABLE users_identities
    ADD COLUMN "dummy_id" text,
    ADD COLUMN "android_id" text,
    ADD COLUMN "android_idfa" text,
    ADD COLUMN "android_push_token" text,
    ADD COLUMN "ios_id" text,
    ADD COLUMN "ios_idfa" text,
    ADD COLUMN "ios_push_token" text,
    ADD COLUMN "first_name" varchar(300),
    ADD COLUMN "last_name" varchar(300),
    ADD COLUMN "gender" gender,    
    ADD COLUMN "food_preferences_drink" drink,
    ADD COLUMN "food_preferences_fruit" fruit,
    ADD COLUMN "phone_numbers" varchar(300)[],
    ADD COLUMN "favorite_movie_title" text,
    ADD COLUMN "favorite_movie_length" double precision,
    ADD COLUMN "favorite_movie_soundtrack_title" text,
    ADD COLUMN "favorite_movie_soundtrack_author" text,
    ADD COLUMN "favorite_movie_soundtrack_length" double precision,
    ADD COLUMN "favorite_movie_soundtrack_genre" music_genre;

ALTER TABLE users
    ADD COLUMN "dummy_id" text,
    ADD COLUMN "android_id" text,
    ADD COLUMN "android_idfa" text,
    ADD COLUMN "android_push_token" text,
    ADD COLUMN "ios_id" text,
    ADD COLUMN "ios_idfa" text,
    ADD COLUMN "ios_push_token" text,
    ADD COLUMN "first_name" varchar(300),
    ADD COLUMN "last_name" varchar(300),
    ADD COLUMN "gender" gender,    
    ADD COLUMN "food_preferences_drink" drink,
    ADD COLUMN "food_preferences_fruit" fruit,
    ADD COLUMN "phone_numbers" varchar(300)[],
    ADD COLUMN "favorite_movie_title" text,
    ADD COLUMN "favorite_movie_length" double precision,
    ADD COLUMN "favorite_movie_soundtrack_title" text,
    ADD COLUMN "favorite_movie_soundtrack_author" text,
    ADD COLUMN "favorite_movie_soundtrack_length" double precision,
    ADD COLUMN "favorite_movie_soundtrack_genre" music_genre;