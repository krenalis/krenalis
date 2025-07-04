--
-- SPDX-License-Identifier: Elastic-2.0
--
--
-- Copyright (c) 2025 Open2b
--
--
-- This file contains queries to run for updating the Meergo data warehouse
-- versions during development.

ALTER TABLE
    events
ADD
    COLUMN IF NOT EXISTS context_os_other VARCHAR NOT NULL DEFAULT '';

ALTER TABLE
    events
ALTER COLUMN
    context_os_other DROP DEFAULT;
