CREATE TYPE _operation AS ENUM ('IdentityResolution', 'AlterSchema');

CREATE TABLE _operations (
    id serial,
    operation _operation NOT NULL,
    start_time timestamp without time zone DEFAULT NULL,
    end_time timestamp without time zone DEFAULT NULL,
    users_version integer DEFAULT NULL,
    PRIMARY KEY ("id")
);