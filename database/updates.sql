
-- 1
CREATE TYPE sending_mode_new AS ENUM ('Client', 'Server', 'ClientAndServer');

-- 2
ALTER TABLE connections
ADD COLUMN sending_mode_tmp sending_mode_new;

-- 3
UPDATE connections
SET sending_mode_tmp =
    CASE sending_mode
        WHEN 'Device' THEN 'Client'
        WHEN 'Cloud' THEN 'Server'
        WHEN 'Combined' THEN 'ClientAndServer'
        ELSE NULL
    END;

-- 4
ALTER TABLE connections
DROP COLUMN sending_mode;

-- 5
ALTER TABLE connections
RENAME COLUMN sending_mode_tmp TO sending_mode;

-- 6
DROP TYPE sending_mode;

-- 7
ALTER TYPE sending_mode_new RENAME TO sending_mode;

-- 8
ALTER TABLE workspaces ADD column warehouse_mcp_settings varchar(65535) NOT NULL DEFAULT 'null'::jsonb;

-- 9
CREATE TYPE access_key_type AS ENUM ('API', 'MCP');

-- 10
ALTER TABLE api_keys
    RENAME TO access_keys;

-- 11
ALTER TABLE access_keys
    ADD COLUMN type access_key_type;

-- 12
UPDATE access_keys
SET type = 'API';

-- 13
ALTER TABLE access_keys
    ALTER COLUMN type SET NOT NULL;

-- 14
ALTER TYPE notification_name ADD VALUE 'CreateAccessKey';
ALTER TYPE notification_name ADD VALUE 'DeleteAccessKey';

-- 15
UPDATE notifications
SET name = 'CreateAccessKey'
WHERE name = 'CreateAPIKey';

-- 16
UPDATE notifications
SET name = 'DeleteAccessKey'
WHERE name = 'DeleteAPIKey';

-- 17
CREATE TYPE notification_name_new AS ENUM (
    'CreateAccessKey',
    'CreateAction',
    'CreateConnection',
    'CreateEventWriteKey',
    'CreateWorkspace',
    'DeleteAccessKey',
    'DeleteAction',
    'DeleteConnection',
    'DeleteEventWriteKey',
    'DeleteWorkspace',
    'EndActionExecution',
    'EndAlterUserSchema',
    'EndIdentityResolution',
    'ExecuteAction',
    'LinkConnection',
    'PurgeActions',
    'RenameConnection',
    'RenameWorkspace',
    'SetAccount',
    'SetActionFormatSettings',
    'SetActionSchedulePeriod',
    'SetActionStatus',
    'SetConnectionSettings',
    'StartAlterUserSchema',
    'StartIdentityResolution',
    'UnlinkConnection',
    'UpdateAction',
    'UpdateConnection',
    'UpdateIdentityPropertiesToUnset',
    'UpdateIdentityResolutionSettings',
    'UpdateWarehouse',
    'UpdateWarehouseMode',
    'UpdateWorkspace'
);

-- 18
ALTER TABLE notifications
    ALTER COLUMN name TYPE notification_name_new
        USING name::text::notification_name_new;

-- 19
DROP TYPE notification_name;

-- 20
ALTER TYPE notification_name_new RENAME TO notification_name

-- 21
ALTER TABLE actions
    ALTER COLUMN filter DROP DEFAULT,
    ALTER COLUMN filter TYPE jsonb
        USING NULLIF(filter, '')::jsonb,
    ALTER COLUMN filter DROP NOT NULL;
