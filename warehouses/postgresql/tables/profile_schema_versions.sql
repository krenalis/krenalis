-- Copyright 2026 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS krenalis_profile_schema_versions (
    version integer NOT NULL,
    operation uuid NOT NULL,          -- operation that created this version of the profile schema.
    timestamp timestamp(3) NOT NULL,  -- timestamp when this version was created.
    PRIMARY KEY ("version")
);
