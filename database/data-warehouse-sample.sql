
CREATE TYPE gender AS ENUM ('male', 'female', 'other');

CREATE TABLE users (
    id          SERIAL,
    "FirstName" varchar(300),
    "LastName"  varchar(300),
    "Email"     varchar(300),
    "Gender"    gender,
    PRIMARY KEY (id)
);
