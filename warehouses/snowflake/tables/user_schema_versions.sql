CREATE TABLE IF NOT EXISTS "_USER_SCHEMA_VERSIONS" (
    "VERSION" INTEGER NOT NULL,
    "OPERATION" VARCHAR NOT NULL,        -- useful for logging purposes.
    "TIMESTAMP" TIMESTAMP_NTZ NOT NULL,  -- useful for logging purposes.
    PRIMARY KEY ("VERSION")
);
