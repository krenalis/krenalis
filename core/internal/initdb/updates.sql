-- Add the queries to update Krenalis's PostgreSQL database here.
--
-- NEVER EMPTY THIS FILE unless absolutely necessary, otherwise it becomes
-- difficult to perform updates.

ALTER TABLE pipelines
    RENAME COLUMN identity_column TO user_id_column;
