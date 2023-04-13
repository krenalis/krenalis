
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
    -- TODO(Gianluca): "PhoneNumbers" has been put between
    -- "FoodPreferences_Drink" and "FoodPreferences_Fruit" to avoid grouping
    -- because the method "Workspace.Users" does not handle nested properties
    -- correctly.
    "PhoneNumbers"          varchar(300)[],
    "FoodPreferences_Fruit" fruit,
    "FavouriteMovie"        movie,
    PRIMARY KEY (id)
);
