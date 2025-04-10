CREATE TABLE IF NOT EXISTS _user_schema_versions (
    version integer NOT NULL,
    operation uuid NOT NULL,          -- useful for logging purposes.
    timestamp timestamp(3) NOT NULL,  -- useful for logging purposes.
    PRIMARY KEY ("version")
);
