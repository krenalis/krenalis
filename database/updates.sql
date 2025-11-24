-- TODO(Gianluca): fill this file with update queries.

ALTER TABLE workspaces RENAME COLUMN alter_user_schema_id TO alter_profile_schema_id;
ALTER TABLE workspaces RENAME COLUMN alter_user_schema_schema TO alter_profile_schema_schema;
ALTER TABLE workspaces RENAME COLUMN alter_user_schema_primary_sources TO alter_profile_schema_primary_sources;
ALTER TABLE workspaces RENAME COLUMN alter_user_schema_operations TO alter_profile_schema_operations;
ALTER TABLE workspaces RENAME COLUMN alter_user_schema_start_time TO alter_profile_schema_start_time;
ALTER TABLE workspaces RENAME COLUMN alter_user_schema_end_time TO alter_profile_schema_end_time;
ALTER TABLE workspaces RENAME COLUMN alter_user_schema_error TO alter_profile_schema_error;
ALTER TABLE workspaces RENAME COLUMN user_schema TO profile_schema;
ALTER TABLE workspaces RENAME COLUMN ui_user_profile_image TO ui_profile_image;
ALTER TABLE workspaces RENAME COLUMN ui_user_profile_first_name TO ui_profile_first_name;
ALTER TABLE workspaces RENAME COLUMN ui_user_profile_last_name TO ui_profile_last_name;
ALTER TABLE workspaces RENAME COLUMN ui_user_profile_extra TO ui_profile_extra;

ALTER TYPE notification_name RENAME VALUE 'StartAlterUserSchema' TO 'StartAlterProfileSchema';
ALTER TYPE notification_name RENAME VALUE 'EndAlterUserSchema' TO 'EndAlterProfileSchema';
