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


ALTER TYPE action_target RENAME TO pipeline_target;
ALTER TABLE workspaces RENAME COLUMN actions_to_purge TO pipelines_to_purge;
ALTER TABLE actions RENAME TO pipelines;
ALTER TABLE actions_executions RENAME TO pipelines_executions;
ALTER TABLE pipelines_executions RENAME COLUMN action TO pipeline;
ALTER TABLE actions_errors RENAME TO pipelines_errors;
ALTER TABLE pipelines_errors RENAME COLUMN action TO pipeline;
ALTER TABLE actions_metrics RENAME TO pipelines_metrics;
ALTER TABLE pipelines_metrics RENAME COLUMN action TO pipeline;

BEGIN;

-- 1) Create the new ENUM type with the corrected values, already sorted alphabetically
CREATE TYPE notification_name_new AS ENUM (
    'AddMember',
    'CreateAccessKey',
    'CreateConnection',
    'CreateEventWriteKey',
    'CreatePipeline',
    'CreateWorkspace',
    'DeleteAccessKey',
    'DeleteConnection',
    'DeleteEventWriteKey',
    'DeleteMember',
    'DeletePipeline',
    'DeleteWorkspace',
    'EndAlterProfileSchema',
    'EndIdentityResolution',
    'EndPipelineExecution',
    'ExecutePipeline',
    'LinkConnection',
    'PurgePipelines',
    'RenameConnection',
    'RenameWorkspace',
    'SetAccount',
    'SetConnectionSettings',
    'SetPipelineFormatSettings',
    'SetPipelineSchedulePeriod',
    'SetPipelineStatus',
    'StartAlterProfileSchema',
    'StartIdentityResolution',
    'UnlinkConnection',
    'UpdateConnection',
    'UpdateIdentityPropertiesToUnset',
    'UpdateIdentityResolutionSettings',
    'UpdatePipeline',
    'UpdateWarehouse',
    'UpdateWarehouseMode',
    'UpdateWorkspace'
    );

-- 2) Migrate notifications.name from the old type to the new one
--    Replacing "Action" with "Pipeline" in existing values
ALTER TABLE notifications
    ALTER COLUMN name TYPE notification_name_new
        USING (
        REPLACE(name::text, 'Action', 'Pipeline')::notification_name_new
        );

-- 3) Drop the old type
DROP TYPE notification_name;

-- 4) Rename the new type to the original name
ALTER TYPE notification_name_new RENAME TO notification_name;

COMMIT;




