-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS meergo_destination_profiles (
    _pipeline integer NOT NULL,
    _external_id text NOT NULL DEFAULT '',
    _out_matching_value text NOT NULL,
    PRIMARY KEY (_pipeline, _external_id)
);
