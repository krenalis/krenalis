

CREATE TYPE connector_type AS ENUM ('App', 'Database', 'File', 'Mobile', 'Server', 'Storage', 'Stream', 'Website');
CREATE TYPE action_target AS ENUM ('Events', 'Users', 'Groups');

CREATE TABLE connectors (
    id SERIAL,
    name varchar(200) NOT NULL DEFAULT '',
    type connector_type NOT NULL DEFAULT 'App',
    oauth_client_id varchar(500) NOT NULL DEFAULT '',
    oauth_client_secret varchar(500) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

INSERT INTO connectors (name, type, oauth_client_id, oauth_client_secret) VALUES
    ('HubSpot', 'App', 'cef1005a-72be-4047-a301-ef6057588325', '136e50df-5b89-478f-bf01-4a71547fa668'),
    ('MySQL', 'Database', '', ''),
    ('Dummy', 'App', '', ''),
    ('Mailchimp', 'App', '631597222767', '90c2d1a1383de35e5ecca5a73f0e2c19e751056d0e3cdd81ac'),
    ('CSV', 'File', '', ''),
    ('SFTP', 'Storage', '', ''),
    ('HTTP', 'Storage', '', ''),
    ('Excel', 'File', '', ''),
    ('S3', 'Storage', '', ''),
    ('PostgreSQL', 'Database', '', ''),
    ('Parquet', 'File', '', ''),
    ('JavaScript', 'Website', '', ''),
    ('Kafka', 'Stream', '', ''),
    ('RabbitMQ', 'Stream', '', ''),
    ('UISample', 'App', '', ''),
    ('.NET', 'Server', '', ''),
    ('Klaviyo', 'App', '', ''),
    ('Google Analytics 4', 'App', '', ''),
    ('Filesystem', 'Storage', '', ''),
    ('ClickHouse', 'Database', '', ''),
    ('JSON', 'File', '', ''),
    ('Mixpanel', 'App', '', ''),
    ('Snowflake', 'Database', '', ''),
    ('Stripe', 'App', '', ''),
    ('Go', 'Server', '', ''),
    ('Java', 'Server', '', ''),
    ('Node.js', 'Server', '', ''),
    ('PHP', 'Server', '', ''),
    ('Python', 'Server', '', '');

CREATE TABLE organizations (
    id SERIAL,
    name varchar(45) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

INSERT INTO organizations (name) VALUES ('ACME inc');

CREATE TYPE warehouse_type AS ENUM ('BigQuery', 'ClickHouse', 'PostgreSQL', 'Redshift', 'Snowflake');
CREATE TYPE privacy_region AS ENUM ('', 'Europe');

CREATE TABLE workspaces (
    id integer NOT NULL,
    organization integer NOT NULL REFERENCES organizations ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    warehouse_type warehouse_type DEFAULT NULL,
    warehouse_settings varchar(65535) NOT NULL DEFAULT '',
    identifiers text[] NOT NULL DEFAULT '{}',
    anonymous_identifiers_priority text[] NOT NULL DEFAULT '{}',
    anonymous_identifiers_mapping jsonb NOT NULL DEFAULT '{}'::jsonb,
    privacy_region privacy_region NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

INSERT INTO workspaces (id, organization, name, warehouse_type, warehouse_settings, anonymous_identifiers_priority, anonymous_identifiers_mapping)
VALUES (1, 1, 'Workspace', NULL, '', '{}', '{}');

CREATE TYPE role AS ENUM ('Source', 'Destination');

CREATE TYPE health AS ENUM ('Healthy', 'NoRecentData', 'RecentError', 'AccessDenied');

CREATE TYPE compression AS ENUM ('', 'Zip', 'Gzip', 'Snappy');

CREATE TABLE connections (
    id integer NOT NULL,
    workspace integer NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    name varchar(100) NOT NULL DEFAULT '',
    type connector_type NOT NULL,
    role role NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    connector integer NOT NULL REFERENCES connectors ON DELETE RESTRICT,
    storage integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    compression compression NOT NULL DEFAULT '',
    resource integer NOT NULL DEFAULT 0,
    website_host varchar(261) NOT NULL DEFAULT '',
    settings varchar(65535),
    health health NOT NULL DEFAULT 'Healthy',
    PRIMARY KEY (id)
);

CREATE TYPE cursor AS (
    id text,
    timestamp timestamp(6)
);

CREATE TYPE export_mode AS ENUM ('CreateOnly', 'UpdateOnly', 'CreateOrUpdate');
CREATE TYPE transformation_language AS ENUM ('JavaScript', 'Python');

CREATE TABLE actions (
    id SERIAL,
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    target action_target NOT NULL,
    event_type varchar(100) NOT NULL,
    name varchar(60) NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT FALSE,
    schedule_start smallint NOT NULL DEFAULT 0 CHECK (schedule_start >= 0 AND schedule_start < 1440),
    schedule_period smallint NOT NULL DEFAULT 60 CHECK(schedule_period IN (5, 15, 30, 60, 120, 180, 360, 480, 720, 1440)),
    in_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    out_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    filter text NOT NULL DEFAULT '',
    transformation_mapping jsonb DEFAULT NULL,
    transformation_source text NOT NULL DEFAULT '',
    transformation_language transformation_language NOT NULL,
    transformation_version varchar(128) NOT NULL DEFAULT '',
    query text NOT NULL DEFAULT '',
    path varchar(1024) NOT NULL DEFAULT '',
    table_name varchar(1024) NOT NULL DEFAULT '',
    sheet varchar(31) NOT NULL DEFAULT '',
    identity_column varchar(1024) NOT NULL DEFAULT '',
    timestamp_column varchar(1024) NOT NULL DEFAULT '',
    timestamp_format varchar(64) NOT NULL DEFAULT '',
    user_cursor cursor NOT NULL DEFAULT '("", "0001-01-01 00:00:00+00")',
    health health NOT NULL DEFAULT 'Healthy',
    export_mode export_mode DEFAULT NULL,
    matching_properties_internal text NOT NULL,
    matching_properties_external text NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE actions_executions (
    id SERIAL,
    action integer NOT NULL REFERENCES actions ON DELETE CASCADE,
    storage integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    reimport boolean NOT NULL DEFAULT FALSE,
    start_time timestamp NOT NULL,
    end_time timestamp DEFAULT NULL,
    error varchar(1000) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

CREATE TABLE actions_executions_stats (
    timeslot integer NOT NULL,
    action integer NOT NULL REFERENCES actions ON DELETE CASCADE,
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
    PRIMARY KEY (timeslot, action)
);

CREATE TABLE connections_keys (
    connection INT NOT NULL REFERENCES connections ON DELETE CASCADE,
    value char(32) NOT NULL,
    creation_time timestamp NOT NULL,
    PRIMARY KEY (connection, value)
);

CREATE TABLE connections_stats (
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    time_slot integer NOT NULL,
    user_identities integer NOT NULL,
    PRIMARY KEY (connection, time_slot)
);

CREATE TABLE connections_stats_events (
    hour integer NOT NULL,
    source integer DEFAULT NULL REFERENCES connections ON DELETE CASCADE,
    good_events integer NOT NULL,
    bad_events integer NOT NULL,
    UNIQUE (hour, source)
);

CREATE TABLE election (
    number integer NOT NULL,
    leader uuid NOT NULL,
    date timestamp NOT NULL,
    PRIMARY KEY (number)
);

INSERT INTO election (number, leader, date) VALUES (1, '00000000-0000-0000-0000-000000000000', '2023-01-01 00:00:00.000000');

CREATE TABLE event_collected (
    id bytea NOT NULL,
    source bytea NOT NULL,
    PRIMARY KEY (id)
);

CREATE TYPE event_job_state AS ENUM ('Running', 'Delivered', 'TransformationFailed');

CREATE TABLE event_processed (
    id bytea NOT NULL,
    action integer NOT NULL REFERENCES actions ON DELETE CASCADE,
    timestamp timestamp NOT NULL,
    state event_job_state NOT NULL DEFAULT 'Running',
    error varchar(1000) NOT NULL DEFAULT ''
);

CREATE TYPE avatar_mime_type AS ENUM ('image/jpeg', 'image/png');

CREATE TYPE avatar AS (
    image bytea,
    mime_type avatar_mime_type
);

CREATE TABLE members (
    id SERIAL,
    organization integer NOT NULL REFERENCES organizations ON DELETE CASCADE,
    name varchar(45) NOT NULL DEFAULT '',
    avatar avatar,
    email varchar(120) NOT NULL DEFAULT '',
    password varchar(72) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

INSERT INTO members (organization, name, avatar, email, password) VALUES (1, 'ACME inc', NULL, 'acme@open2b.com', '$2a$10$iMuokZyvwdAQOJJmJvG83eSGGWTV3DOjI2DRU6SjuLEuK.vknUJVC'); -- Password: foopass2

CREATE TABLE resources (
    id SERIAL,
    workspace integer NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    connector integer NOT NULL REFERENCES connectors ON DELETE CASCADE,
    code varchar(100) NOT NULL,
    access_token varchar(500) NOT NULL DEFAULT '',
    refresh_token varchar(500) NOT NULL DEFAULT '',
    expires_in timestamp(0),
    PRIMARY KEY (id)
);

CREATE INDEX ON resources (connector);

CREATE TYPE task_status AS ENUM ('pending', 'running', 'completed', 'failed');

CREATE TABLE tasks (
  id          SERIAL PRIMARY KEY,
  type        VARCHAR(255) NOT NULL,
  data        JSONB NOT NULL,
  status      task_status NOT NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
  started_at  TIMESTAMP,
  completed_at TIMESTAMP,
  result      JSONB
);
