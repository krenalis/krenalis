-- Copyright 2026 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS "MEERGO_DESTINATION_PROFILES" (
    "_PIPELINE" INT NOT NULL,
    "_EXTERNAL_ID" VARCHAR NOT NULL DEFAULT '',
    "_OUT_MATCHING_VALUE" VARCHAR NOT NULL,
    PRIMARY KEY ("_PIPELINE", "_EXTERNAL_ID")
);
