CREATE COLLATION case_insensitive (provider = icu, locale = 'und-u-ks-level2', deterministic = false);

CREATE TYPE pipeline_target AS ENUM ('Event', 'User', 'Group');

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE organizations (
    id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
    name varchar(255) NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT FALSE,
    members_limit integer NOT NULL CHECK (members_limit BETWEEN 1 AND 10000),
    access_keys_limit integer NOT NULL CHECK (access_keys_limit BETWEEN 0 AND 1000),
    workspaces_limit integer NOT NULL CHECK (workspaces_limit BETWEEN 0 AND 1000),
    connectors_limit integer NOT NULL CHECK (connectors_limit BETWEEN 0 AND 1000),
    connections_limit integer NOT NULL CHECK (connections_limit BETWEEN 0 AND 10000),
    pipelines_limit integer NOT NULL CHECK (pipelines_limit BETWEEN 0 AND 10000),
    PRIMARY KEY (id)
);

CREATE TYPE avatar_mime_type AS ENUM ('image/jpeg', 'image/png');

CREATE TYPE avatar AS (
    image bytea,
    mime_type avatar_mime_type
);

CREATE TABLE members (
    id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
    organization varchar(12) NOT NULL REFERENCES organizations ON DELETE CASCADE,
    name varchar(255) NOT NULL DEFAULT '',
    avatar avatar,
    email varchar(255) NOT NULL COLLATE case_insensitive,
    password varchar(72) NOT NULL DEFAULT '',
    workos_user_id varchar(255) NOT NULL DEFAULT '',
    invitation_token varchar(44) NOT NULL DEFAULT '',
    reset_password_token varchar(44) NOT NULL DEFAULT '',
    reset_password_token_created_at timestamp,
    created_at timestamp,
    UNIQUE (organization, email),
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX invitation_token_index ON members (invitation_token) WHERE invitation_token <> '';
CREATE UNIQUE INDEX reset_password_token_index ON members (reset_password_token) WHERE reset_password_token <> '';
CREATE UNIQUE INDEX members_workos_user_id_idx ON members (organization, workos_user_id) WHERE workos_user_id <> '';

CREATE TYPE warehouse_mode AS ENUM ('Normal', 'Inspection', 'Maintenance');

CREATE TABLE workspaces (
    id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
    organization varchar(12) NOT NULL REFERENCES organizations ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    warehouse_name varchar NOT NULL,
    warehouse_mode warehouse_mode NOT NULL,
    warehouse_settings bytea NOT NULL,
    warehouse_mcp_settings bytea DEFAULT NULL,
    kms_encrypted_warehouse_settings_key bytea NOT NULL,
    kms_encrypted_warehouse_mcp_settings_key bytea NOT NULL,
    alter_profile_schema_id uuid,
    alter_profile_schema_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    alter_profile_schema_primary_sources jsonb,
    alter_profile_schema_operations jsonb,
    alter_profile_schema_start_time timestamp(3),
    alter_profile_schema_end_time timestamp(3),
    alter_profile_schema_error varchar,
    profile_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    resolve_identities_on_batch_import boolean NOT NULL DEFAULT false,
    identifiers text[] NOT NULL DEFAULT '{}',
    ir_id uuid,
    ir_start_time timestamp(3),
    ir_end_time timestamp(3),
    ui_profile_image varchar(100) NOT NULL DEFAULT '',
    ui_profile_first_name varchar(100) NOT NULL DEFAULT '',
    ui_profile_last_name varchar(100) NOT NULL DEFAULT '',
    ui_profile_extra varchar(100) NOT NULL DEFAULT '',
    pipelines_to_purge varchar(12)[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (id)
);

CREATE INDEX workspaces_organization_idx ON workspaces (organization);

CREATE TYPE access_key_type AS ENUM ('API', 'MCP');

CREATE TABLE access_keys (
    id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
    organization varchar(12) NOT NULL REFERENCES organizations ON DELETE CASCADE,
    workspace varchar(12) REFERENCES workspaces ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    type access_key_type NOT NULL,
    hmac bytea NOT NULL UNIQUE,
    hint varchar(13) NOT NULL,
    created_at timestamp(0) NOT NULL,
    PRIMARY KEY (id)
);

CREATE TYPE role AS ENUM ('Source', 'Destination');

CREATE TYPE health AS ENUM ('Healthy', 'NoRecentData', 'RecentError');

CREATE TYPE compression AS ENUM ('', 'Zip', 'Gzip', 'Snappy');

CREATE TYPE strategy AS ENUM ('Conversion', 'Fusion', 'Isolation', 'Preservation');

CREATE TYPE sending_mode as ENUM ('Client', 'Server', 'ClientAndServer');

CREATE TABLE connections (
    id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
    workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    name varchar(100) NOT NULL DEFAULT '',
    connector varchar,
    role role NOT NULL,
    account integer NOT NULL DEFAULT 0,
    strategy strategy,
    sending_mode sending_mode,
    linked_connections varchar(12)[],
    settings bytea,
    kms_encrypted_settings_key bytea NOT NULL,
    health health NOT NULL DEFAULT 'Healthy',
    PRIMARY KEY (id)
);

CREATE INDEX connections_workspace_idx ON connections (workspace);

CREATE TYPE export_mode AS ENUM ('', 'CreateOnly', 'UpdateOnly', 'CreateOrUpdate');
CREATE TYPE transformation_language AS ENUM ('JavaScript', 'Python');

CREATE TABLE pipelines (
    id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
    connection varchar(12) NOT NULL REFERENCES connections ON DELETE CASCADE,
    target pipeline_target NOT NULL,
    event_type varchar(100) NOT NULL,
    name varchar(60) NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT FALSE,
    schedule_start smallint NOT NULL DEFAULT 0 CHECK (schedule_start >= 0 AND schedule_start < 1440),
    schedule_period smallint NOT NULL DEFAULT 0 CHECK(schedule_period IN (0, 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440)),
    in_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    out_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    filter jsonb,
    transformation_mapping jsonb,
    transformation_id varchar(200) NOT NULL DEFAULT '',
    transformation_version varchar(128) NOT NULL DEFAULT '',
    transformation_language transformation_language NOT NULL,
    transformation_source text NOT NULL DEFAULT '',
    transformation_preserve_json boolean NOT NULL DEFAULT false,
    transformation_in_paths varchar[],
    transformation_out_paths varchar[],
    query text NOT NULL DEFAULT '',
    format varchar,
    path varchar(1024) NOT NULL DEFAULT '',
    sheet varchar(31) NOT NULL DEFAULT '',
    compression compression NOT NULL DEFAULT '',
    order_by varchar(1024) NOT NULL DEFAULT '',
    format_settings jsonb,
    export_mode export_mode NOT NULL DEFAULT '',
    matching_in text NOT NULL,
    matching_out text NOT NULL,
    update_on_duplicates boolean NOT NULL,
    table_name varchar(1024) NOT NULL DEFAULT '',
    table_key text NOT NULL,
    user_id_column varchar(1024) NOT NULL DEFAULT '',
    updated_at_column varchar(1024) NOT NULL DEFAULT '',
    updated_at_format varchar(64) NOT NULL DEFAULT '',
    incremental boolean NOT NULL DEFAULT FALSE,
    cursor timestamp NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    health health NOT NULL DEFAULT 'Healthy',
    properties_to_unset varchar[],
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX pipelines_transformation_id_idx ON pipelines (transformation_id) WHERE transformation_id <> '';

-- Connectors can be referenced by both connections and pipeline formats.
-- Keep those references in one place so limit checks do not need to duplicate
-- that rule. References are intentionally not deduplicated: callers decide
-- whether to count distinct connectors, check for a specific connector, or
-- exclude a resource.
CREATE VIEW organization_connector_references AS
SELECT
    ws.organization,
    c.connector,
    'connection' AS resource_type,
    c.id AS resource
FROM connections c
JOIN workspaces ws ON ws.id = c.workspace
UNION ALL
SELECT
    ws.organization,
    p.format AS connector,
    'pipeline' AS resource_type,
    p.id AS resource
FROM pipelines p
JOIN connections c ON c.id = p.connection
JOIN workspaces ws ON ws.id = c.workspace
WHERE p.format IS NOT NULL;

CREATE TABLE pipelines_runs (
    id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
    pipeline varchar(12) NOT NULL REFERENCES pipelines ON DELETE CASCADE,
    function varchar(200) NOT NULL DEFAULT '',
    node uuid,
    incremental boolean NOT NULL DEFAULT FALSE,
    cursor timestamp NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    start_time timestamp NOT NULL,
    ping_time timestamp NOT NULL,
    end_time timestamp,
    passed_0 integer NOT NULL DEFAULT 0,
    passed_1 integer NOT NULL DEFAULT 0,
    passed_2 integer NOT NULL DEFAULT 0,
    passed_3 integer NOT NULL DEFAULT 0,
    passed_4 integer NOT NULL DEFAULT 0,
    passed_5 integer NOT NULL DEFAULT 0,
    failed_0 integer NOT NULL DEFAULT 0,
    failed_1 integer NOT NULL DEFAULT 0,
    failed_2 integer NOT NULL DEFAULT 0,
    failed_3 integer NOT NULL DEFAULT 0,
    failed_4 integer NOT NULL DEFAULT 0,
    failed_5 integer NOT NULL DEFAULT 0,
    error varchar NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

CREATE INDEX pipelines_runs_function_idx
    ON pipelines_runs (function)
    WHERE function != '' AND end_time IS NULL;

CREATE UNIQUE INDEX pipelines_one_live_run_idx
    ON pipelines_runs (pipeline)
    WHERE end_time IS NULL;

CREATE TABLE pipelines_errors (
    pipeline varchar(12) NOT NULL REFERENCES pipelines ON DELETE CASCADE,
    timeslot integer NOT NULL,
    step smallint NOT NULL,
    count integer NOT NULL,
    message varchar NOT NULL
);

CREATE INDEX ON pipelines_errors (pipeline);
CREATE INDEX ON pipelines_errors (timeslot);
CREATE INDEX ON pipelines_errors (step);

CREATE TABLE pipelines_metrics (
    pipeline varchar(12) NOT NULL REFERENCES pipelines ON DELETE CASCADE,
    timeslot integer NOT NULL,
    passed_0 integer NOT NULL,
    passed_1 integer NOT NULL,
    passed_2 integer NOT NULL,
    passed_3 integer NOT NULL,
    passed_4 integer NOT NULL,
    passed_5 integer NOT NULL,
    failed_0 integer NOT NULL,
    failed_1 integer NOT NULL,
    failed_2 integer NOT NULL,
    failed_3 integer NOT NULL,
    failed_4 integer NOT NULL,
    failed_5 integer NOT NULL,
    PRIMARY KEY (pipeline, timeslot)
);

CREATE INDEX ON pipelines_metrics (pipeline);
CREATE INDEX ON pipelines_metrics (timeslot);

CREATE TABLE discontinued_functions (
    id varchar(200) NOT NULL,
    discontinued_at timestamp(0) NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE election (
    number integer NOT NULL,
    leader uuid NOT NULL,
    date timestamp NOT NULL,
    PRIMARY KEY (number)
);

INSERT INTO election (number, leader, date) VALUES (1, '00000000-0000-0000-0000-000000000000', '2023-01-01 00:00:00.000000');

CREATE TABLE event_write_keys (
    connection varchar(12) NOT NULL REFERENCES connections ON DELETE CASCADE,
    key char(32) NOT NULL,
    created_at timestamp NOT NULL,
    PRIMARY KEY (connection, key)
);

CREATE TABLE primary_sources (
    source varchar(12) NOT NULL REFERENCES connections ON DELETE CASCADE,
    path varchar NOT NULL,
    PRIMARY KEY (source, path)
);

CREATE TABLE accounts (
    id integer GENERATED BY DEFAULT AS IDENTITY,
    workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    connector varchar NOT NULL,
    code varchar(100) NOT NULL,
    access_token varchar(500) NOT NULL DEFAULT '',
    refresh_token varchar(500) NOT NULL DEFAULT '',
    expires_in timestamp(0),
    PRIMARY KEY (id)
);

CREATE INDEX ON accounts (connector);

CREATE TYPE notification_name AS ENUM (
    'AcceptInvitation',
    'AddMember',
    'CreateAccessKey',
    'CreateConnection',
    'CreateEventWriteKey',
    'CreateOrganization',
    'CreatePipeline',
    'CreateWorkspace',
    'DeleteAccessKey',
    'DeleteConnection',
    'DeleteEventWriteKey',
    'DeleteMember',
    'DeleteMembers',
    'DeleteOrganization',
    'DeletePipeline',
    'DeleteWorkspace',
    'EndAlterProfileSchema',
    'EndIdentityResolution',
    'EndPipelineRun',
    'LinkConnection',
    'PurgePipelines',
    'RenameConnection',
    'RenameWorkspace',
    'RunPipeline',
    'SetAccount',
    'SetConnectionSettings',
    'SetOrganizationStatus',
    'SetPipelineFormatSettings',
    'SetPipelineSchedulePeriod',
    'SetPipelineStatus',
    'StartAlterProfileSchema',
    'StartIdentityResolution',
    'UnlinkConnection',
    'UpdateConnection',
    'UpdateIdentityPropertiesToUnset',
    'UpdateIdentityResolutionSettings',
    'UpdateOrganization',
    'UpdatePipeline',
    'UpdateWarehouse',
    'UpdateWarehouseMode',
    'UpdateWorkspace'
);

CREATE TABLE notifications (
    id bigint NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    name notification_name NOT NULL,
    payload jsonb NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE metadata (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    installation_id text UNIQUE NOT NULL,
    kms_encrypted_cookie_key bytea NOT NULL,
    kms_encrypted_oauth_key bytea NOT NULL,
    kms_encrypted_notification_key bytea NOT NULL,
    kms_encrypted_api_key_pepper bytea NOT NULL
);

INSERT INTO metadata (
    installation_id,
    kms_encrypted_cookie_key,
    kms_encrypted_oauth_key,
    kms_encrypted_notification_key,
    kms_encrypted_api_key_pepper
) VALUES (
    gen_random_uuid(),
    '\x'::bytea,
    '\x'::bytea,
    '\x'::bytea,
    '\x'::bytea
);
