-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS "MEERGO_DESTINATION_PROFILES" (
    "_pipeline" INT NOT NULL,
    "_external_id" VARCHAR NOT NULL DEFAULT '',
    "_out_matching_value" VARCHAR NOT NULL,
    PRIMARY KEY ("_pipeline", "_external_id")
);
