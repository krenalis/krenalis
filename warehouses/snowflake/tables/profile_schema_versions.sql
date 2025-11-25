-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS "_MEERGO_PROFILE_SCHEMA_VERSIONS" (
    "VERSION" INTEGER NOT NULL,
    "OPERATION" VARCHAR NOT NULL,        -- useful for logging purposes.
    "TIMESTAMP" TIMESTAMP_NTZ NOT NULL,  -- useful for logging purposes.
    PRIMARY KEY ("VERSION")
);
