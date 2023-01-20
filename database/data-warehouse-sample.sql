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
    PRIMARY KEY (id)
);