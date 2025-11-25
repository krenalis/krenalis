-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS "MEERGO_DESTINATION_PROFILES" (
    "__ACTION__" INT NOT NULL,
    "__EXTERNAL_ID__" VARCHAR NOT NULL DEFAULT '',
    "__OUT_MATCHING_VALUE__" VARCHAR NOT NULL,
    PRIMARY KEY ("__ACTION__", "__EXTERNAL_ID__")
);
