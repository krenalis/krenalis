
CREATE TABLE accounts (
    id SERIAL,
    name varchar(45) NOT NULL DEFAULT '',
    email varchar(120) NOT NULL DEFAULT '',
    password varchar(60) NOT NULL DEFAULT '',
    internal_ips VARCHAR(160) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

INSERT INTO accounts (name, email, password) VALUES
    ('ACME inc', 'acme@open2b.com', '$2a$10$iMuokZyvwdAQOJJmJvG83eSGGWTV3DOjI2DRU6SjuLEuK.vknUJVC'); -- Password: foopass2

CREATE TYPE connector_type AS ENUM ('App', 'Database', 'File', 'Mobile', 'Server', 'Storage', 'Stream', 'Website');
CREATE TYPE action_target AS ENUM ('Events', 'Users', 'Groups');

CREATE TABLE connectors (
    id SERIAL,
    name varchar(200) NOT NULL DEFAULT '',
    type connector_type NOT NULL DEFAULT 'App',
    logo_url varchar(500) NOT NULL DEFAULT '',
    oauth_url varchar(500) NOT NULL DEFAULT '',
    oauth_client_id varchar(500) NOT NULL DEFAULT '',
    oauth_client_secret varchar(500) NOT NULL DEFAULT '',
    oauth_token_endpoint varchar(500) NOT NULL DEFAULT '',
    oauth_default_token_type varchar(10) NOT NULL DEFAULT '',
    oauth_default_expires_in integer NOT NULL DEFAULT 0,
    oauth_forced_expires_in integer NOT NULL DEFAULT 0,
    PRIMARY KEY (id)
);

INSERT INTO connectors (name, type, logo_url, oauth_url, oauth_client_id, oauth_client_secret, oauth_token_endpoint, oauth_forced_expires_in) VALUES
    ('HubSpot', 'App', 'https://cdn4.iconfinder.com/data/icons/logos-and-brands/512/168_Hubspot_logo_logos-512.png', 'https://app-eu1.hubspot.com/oauth/authorize?client_id=cef1005a-72be-4047-a301-ef6057588325&redirect_uri=https://localhost:9090/admin/oauth/authorize&scope=crm.objects.contacts.read%20crm.objects.contacts.write%20crm.schemas.contacts.read', 'cef1005a-72be-4047-a301-ef6057588325', '136e50df-5b89-478f-bf01-4a71547fa668', 'https://api.hubapi.com/oauth/v1/token', 0),
    ('MySQL', 'Database', 'https://cdn4.iconfinder.com/data/icons/logos-3/181/MySQL-512.png', '', '', '', '', 0),
    ('Dummy', 'App', 'https://cdn3.iconfinder.com/data/icons/education-209/64/tube-lab-science-school-256.png', '', '', '', '', 0),
    ('Mailchimp', 'App', 'https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/mailchimp-512.png', 'https://login.mailchimp.com/oauth2/authorize?response_type=code&client_id=631597222767&redirect_uri=https://127.0.0.1:9090/admin/oauth/authorize', '631597222767', '90c2d1a1383de35e5ecca5a73f0e2c19e751056d0e3cdd81ac', 'https://login.mailchimp.com/oauth2/token',2147483647),
    ('CSV', 'File', 'https://cdn3.iconfinder.com/data/icons/cad-database-presentation-spreadsheet-vector-fil-2/512/19-512.png', '', '', '', '', 0),
    ('SFTP', 'Storage', 'https://cdn2.iconfinder.com/data/icons/whcompare-servers-web-hosting/50/sftp-512.png', '', '', '', '', 0),
    ('HTTP', 'Storage', 'https://cdn4.iconfinder.com/data/icons/application-windows-3/32/HTTP-512.png', '', '', '', '', 0),
    ('Excel', 'File', 'https://cdn0.iconfinder.com/data/icons/logos-microsoft-office-365/128/Microsoft_Office-02-512.png', '', '', '', '', 0),
    ('S3', 'Storage', 'https://cdn2.iconfinder.com/data/icons/amazon-aws-stencils/100/Storage__Content_Delivery_Amazon_S3-512.png', '', '', '', '', 0),
    ('PostgreSQL', 'Database', 'https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/postgresql-512.png', '', '', '', '', 0),
    ('Parquet', 'File', '', '', '', '', '', 0),
    ('Website', 'Website', 'https://cdn2.iconfinder.com/data/icons/free-simple-line-mix/48/22-Website-512.png', '', '', '', '', 0),
    ('Kafka', 'Stream', 'https://cdn.icon-icons.com/icons2/2248/PNG/512/apache_kafka_icon_138937.png', '', '', '', '', 0),
    ('RabbitMQ', 'Stream', 'https://cdn.icon-icons.com/icons2/2699/PNG/512/rabbitmq_logo_icon_170810.png', '', '', '', '', 0),
    ('UISample', 'Stream', '', '', '', '', '', 0),
    ('Server', 'Server', 'https://cdn1.iconfinder.com/data/icons/unicons-line-vol-5/24/server-512.png', '', '', '', '', 0),
    ('Klaviyo', 'App', 'https://cdn3.iconfinder.com/data/icons/font-awesome-solid/512/k-256.png', '', '', '', '', 0),
    ('Google Analytics 4', 'App', 'https://cdn4.iconfinder.com/data/icons/social-media-logos-6/512/108-analytics_google_analytics_google-256.png', '', '', '', '', 0),
    ('Filesystem', 'Storage', 'https://cdn2.iconfinder.com/data/icons/audio-music-5/65/cd-case-256.png', '', '', '', '', 0);

CREATE TYPE warehouse_type AS ENUM ('BigQuery', 'ClickHouse', 'PostgreSQL', 'Redshift', 'Snowflake');
CREATE TYPE privacy_region AS ENUM ('', 'Europe');

CREATE TABLE workspaces (
    id integer NOT NULL,
    account integer NOT NULL REFERENCES accounts ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    warehouse_type warehouse_type DEFAULT NULL,
    warehouse_settings varchar(65535) NOT NULL DEFAULT '',
    privacy_region privacy_region NOT NULL DEFAULT '',
    schemas text NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

INSERT INTO workspaces (id, account, name, warehouse_type, warehouse_settings)
VALUES (1, 1, 'Workspace', NULL, '');

CREATE TYPE role AS ENUM ('Source', 'Destination');

CREATE TYPE health AS ENUM ('Healthy', 'NoRecentData', 'RecentError', 'AccessDenied');

CREATE TABLE connections (
    id integer NOT NULL,
    workspace integer NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    name varchar(100) NOT NULL DEFAULT '',
    type connector_type NOT NULL,
    role role NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    connector integer NOT NULL REFERENCES connectors ON DELETE RESTRICT,
    storage integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    resource integer NOT NULL DEFAULT 0,
    website_host varchar(261) NOT NULL DEFAULT '',
    user_cursor varchar(500) NOT NULL DEFAULT '',
    identity_column varchar(100) NOT NULL DEFAULT '',
    timestamp_column varchar(100) NOT NULL DEFAULT '',
    settings varchar(65535),
    health health NOT NULL DEFAULT 'Healthy',
    PRIMARY KEY (id)
);

CREATE TYPE transformation AS (
    in_types text,
    out_types text,
    python_source text
);

CREATE TABLE actions (
    id SERIAL,
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    target action_target NOT NULL,
    event_type varchar(100) NOT NULL,
    name varchar(60) NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT FALSE,
    schedule_start smallint NOT NULL DEFAULT 0 CHECK (schedule_start >= 0 AND schedule_start < 1440),
    schedule_period smallint NOT NULL DEFAULT 60 CHECK(schedule_period IN (5, 15, 30, 60, 120, 180, 360, 480, 720, 1140)),
    filter text NOT NULL DEFAULT '',
    schema text NOT NULL DEFAULT '',
    mapping text NOT NULL DEFAULT '',
    transformation transformation NOT NULL DEFAULT '("", "", "")',
    query text NOT NULL DEFAULT '',
    path varchar(1024) NOT NULL DEFAULT '',
    sheet varchar(100) NOT NULL DEFAULT '',
    health health NOT NULL DEFAULT 'Healthy',
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

CREATE TABLE connections_keys (
    connection INT NOT NULL REFERENCES connections ON DELETE CASCADE,
    value char(32) NOT NULL,
    creation_time timestamp NOT NULL,
    PRIMARY KEY (connection, value)
);

CREATE TABLE connections_stats (
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    time_slot integer NOT NULL,
    users integer NOT NULL,
    PRIMARY KEY (connection, time_slot)
);

CREATE TABLE connections_stats_events (
    hour integer NOT NULL,
    source integer DEFAULT NULL REFERENCES connections ON DELETE CASCADE,
    server integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    stream integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    good_events integer NOT NULL,
    bad_events integer NOT NULL,
    UNIQUE (hour, source, server, stream)
);

CREATE INDEX ON connections_stats_events (server);
CREATE INDEX ON connections_stats_events (stream);

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

CREATE TABLE properties (
    id SERIAL,
    code char(10) NOT NULL UNIQUE,
    account integer NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX ON properties (account);

INSERT INTO properties VALUES (1, '1234567890', 1);

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
