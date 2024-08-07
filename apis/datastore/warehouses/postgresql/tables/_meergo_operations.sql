CREATE TYPE meergo_operation AS ENUM ('IdentityResolution', 'AlterSchema');

CREATE TABLE _meergo_operations (
    id serial,
    operation meergo_operation NOT NULL,
    start_time timestamp without time zone DEFAULT NULL,
    end_time timestamp without time zone DEFAULT NULL,
    PRIMARY KEY ("id")
);