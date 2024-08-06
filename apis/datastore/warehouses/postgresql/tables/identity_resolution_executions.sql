CREATE TABLE _identity_resolution_executions (
    id serial,
    start_time timestamp without time zone DEFAULT NULL,
    end_time timestamp without time zone DEFAULT NULL,
    PRIMARY KEY ("id")
);