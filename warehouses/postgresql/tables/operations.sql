-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

DO $$
    BEGIN
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = '_operation') THEN
        CREATE TYPE _operation AS ENUM ('IdentityResolution', 'AlterUserSchema');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS _operations (
    id uuid NOT NULL,
    operation_type _operation,
    completed_at timestamp(3),
    error text NOT NULL DEFAULT '',
    PRIMARY KEY ("id")
);
