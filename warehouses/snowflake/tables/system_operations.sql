-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS "MEERGO_SYSTEM_OPERATIONS" (
    "ID" VARCHAR NOT NULL,
    "OPERATION_TYPE" VARCHAR NOT NULL,
    "COMPLETED_AT" TIMESTAMP_NTZ,
    "ERROR" VARCHAR NOT NULL DEFAULT '',
    PRIMARY KEY ("ID")
);
