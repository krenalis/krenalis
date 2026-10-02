-- Copyright 2026 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS "KRENALIS_PROFILE_SCHEMA_VERSIONS" (
    "VERSION" INTEGER NOT NULL,
    "OPERATION" VARCHAR NOT NULL,        -- operation that created this version of the profile schema.
    "TIMESTAMP" TIMESTAMP_NTZ NOT NULL,  -- timestamp when this version was created.
    PRIMARY KEY ("VERSION")
);
