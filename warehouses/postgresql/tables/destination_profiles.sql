-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

CREATE TABLE IF NOT EXISTS meergo_destination_profiles (
    __action__ integer NOT NULL,
    __external_id__ text NOT NULL DEFAULT '',
    __out_matching_value__ text NOT NULL,
    PRIMARY KEY (__action__, __external_id__)
);
