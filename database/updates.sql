-- TODO(Gianluca): fill this file with update queries.

ALTER TABLE user_schema_primary_sources RENAME TO primary_sources;

BEGIN;
ALTER TYPE notification_name
    ADD VALUE IF NOT EXISTS 'AddMember' BEFORE 'CreateAccessKey';
ALTER TYPE notification_name
    ADD VALUE IF NOT EXISTS 'DeleteMember' AFTER 'DeleteEventWriteKey';
COMMIT;
