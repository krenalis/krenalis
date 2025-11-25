-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

DO $$
    BEGIN
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = 'system_operation_type') THEN
        CREATE TYPE system_operation_type AS ENUM ('IdentityResolution', 'AlterProfileSchema');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS meergo_system_operations (
    id uuid NOT NULL,
    operation_type system_operation_type,
    completed_at timestamp(3),
    error text NOT NULL DEFAULT '',
    PRIMARY KEY ("id")
);
