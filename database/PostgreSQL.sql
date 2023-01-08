
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

CREATE TYPE connector_type AS ENUM ('App', 'Database', 'EventStream', 'File', 'Mobile', 'Server', 'Storage', 'Website');
CREATE TYPE webhooks_per AS ENUM ('None', 'Connector', 'Resource', 'Source');

CREATE TABLE connectors (
    id SERIAL,
    name varchar(200) NOT NULL DEFAULT '',
    type connector_type NOT NULL DEFAULT 'App',
    logo_url varchar(500) NOT NULL DEFAULT '',
    webhooks_per webhooks_per NOT NULL DEFAULT 'None',
    oauth_url varchar(500) NOT NULL DEFAULT '',
    oauth_client_id varchar(500) NOT NULL DEFAULT '',
    oauth_client_secret varchar(500) NOT NULL DEFAULT '',
    oauth_token_endpoint varchar(500) NOT NULL DEFAULT '',
    oauth_default_token_type varchar(10) NOT NULL DEFAULT '',
    oauth_default_expires_in integer NOT NULL DEFAULT 0,
    oauth_forced_expires_in integer NOT NULL DEFAULT 0,
    PRIMARY KEY (id)
);

INSERT INTO connectors (name, type, logo_url, webhooks_per, oauth_url, oauth_client_id, oauth_client_secret, oauth_token_endpoint, oauth_forced_expires_in) VALUES
    ('HubSpot', 'App', 'https://cdn4.iconfinder.com/data/icons/logos-and-brands/512/168_Hubspot_logo_logos-512.png', 'Connector', 'https://app-eu1.hubspot.com/oauth/authorize?client_id=cef1005a-72be-4047-a301-ef6057588325&redirect_uri=https://localhost:9090/admin/oauth/authorize&scope=crm.objects.contacts.read%20crm.objects.contacts.write%20crm.schemas.contacts.read', 'cef1005a-72be-4047-a301-ef6057588325', '136e50df-5b89-478f-bf01-4a71547fa668', 'https://api.hubapi.com/oauth/v1/token', 0),
    ('MySQL', 'Database', 'https://cdn4.iconfinder.com/data/icons/logos-3/181/MySQL-512.png', 'None', '', '', '', '', 0),
    ('Dummy', 'App', '', 'Connector', '', '', '', '', 0),
    ('Mailchimp', 'App', 'https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/mailchimp-512.png', 'Source', 'https://login.mailchimp.com/oauth2/authorize?response_type=code&client_id=631597222767&redirect_uri=https://127.0.0.1:9090/admin/oauth/authorize', '631597222767', '90c2d1a1383de35e5ecca5a73f0e2c19e751056d0e3cdd81ac', 'https://login.mailchimp.com/oauth2/token',2147483647),
    ('CSV', 'File', 'https://cdn3.iconfinder.com/data/icons/cad-database-presentation-spreadsheet-vector-fil-2/512/19-512.png', 'None', '', '', '', '', 0),
    ('SFTP', 'Storage', 'https://cdn2.iconfinder.com/data/icons/whcompare-servers-web-hosting/50/sftp-512.png', 'None', '', '', '', '', 0),
    ('HTTP', 'Storage', 'https://cdn4.iconfinder.com/data/icons/application-windows-3/32/HTTP-512.png', 'None', '', '', '', '', 0),
    ('Excel', 'File', 'https://cdn0.iconfinder.com/data/icons/logos-microsoft-office-365/128/Microsoft_Office-02-512.png', 'None', '', '', '', '', 0),
    ('S3', 'Storage', 'https://cdn2.iconfinder.com/data/icons/amazon-aws-stencils/100/Storage__Content_Delivery_Amazon_S3-512.png', 'None', '', '', '', '', 0),
    ('PostgreSQL', 'Database', 'https://cdn4.iconfinder.com/data/icons/logos-brands-5/24/postgresql-512.png', 'None', '', '', '', '', 0),
    ('Parquet', 'File', '', 'None', '', '', '', '', 0),
    ('Website', 'Website', 'https://cdn2.iconfinder.com/data/icons/free-simple-line-mix/48/22-Website-512.png', 'None', '', '', '', '', 0),
    ('Kafka', 'EventStream', 'https://cdn.icon-icons.com/icons2/2248/PNG/512/apache_kafka_icon_138937.png', 'None', '', '', '', '', 0),
    ('RabbitMQ', 'EventStream', 'https://cdn.icon-icons.com/icons2/2699/PNG/512/rabbitmq_logo_icon_170810.png', 'None', '', '', '', '', 0),
    ('UISample', 'EventStream', '', 'None', '', '', '', '', 0);

CREATE TYPE warehouse_type AS ENUM ('BigQuery', 'ClickHouse', 'PostgreSQL', 'Redshift', 'Snowflake');

CREATE TABLE workspaces (
    id SERIAL,
    account integer NOT NULL,
    name varchar(100) NOT NULL,
    warehouse_type warehouse_type DEFAULT NULL,
    warehouse_settings varchar(65535) NOT NULL DEFAULT '',
    schema text NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

INSERT INTO workspaces (id, account, name, warehouse_type, warehouse_settings)
VALUES (1, 1, 'Workspace', NULL, '');

CREATE TYPE role AS ENUM ('Source', 'Destination');

CREATE TABLE connections (
    id integer NOT NULL,
    workspace integer NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    name varchar(120) NOT NULL DEFAULT '',
    type connector_type NOT NULL,
    role role NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    connector integer NOT NULL REFERENCES connectors ON DELETE RESTRICT,
    storage integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    stream integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    resource integer NOT NULL DEFAULT 0,
    website_host varchar(261) NOT NULL DEFAULT '',
    user_cursor varchar(500) NOT NULL DEFAULT '',
    identity_column varchar(100) NOT NULL DEFAULT '',
    timestamp_column varchar(100) NOT NULL DEFAULT '',
    settings varchar(65535) NOT NULL DEFAULT '',
    schema text NOT NULL DEFAULT '',
    users_query text NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

CREATE TABLE connections_imports (
    id SERIAL,
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    storage integer NOT NULL,
    start_time timestamp NOT NULL,
    end_time timestamp DEFAULT NULL,
    error varchar(1000) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

CREATE TABLE connections_exports (
    id SERIAL,
    connection integer NOT NULL,
    storage integer NOT NULL,
    start_time timestamp NOT NULL,
    end_time timestamp DEFAULT NULL,
    error varchar(1000) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);

CREATE TABLE connections_keys (
    connection INT NOT NULL REFERENCES connections ON DELETE CASCADE,
    value BYTEA NOT NULL,
    creation_time timestamp NOT NULL,
    PRIMARY KEY (connection, value)
);

CREATE TABLE connections_stats (
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    time_slot integer NOT NULL,
    users_in integer NOT NULL,
    PRIMARY KEY (connection, time_slot)
);

CREATE TABLE connections_stats_events (
    hour integer NOT NULL,
    source integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    server integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    stream integer DEFAULT NULL REFERENCES connections ON DELETE SET NULL,
    good_events integer NOT NULL,
    bad_events integer NOT NULL,
    UNIQUE (hour, source, server, stream)
);

CREATE INDEX ON connections_stats_events (server);
CREATE INDEX ON connections_stats_events (stream);

CREATE TABLE devices (
    source INT NOT NULL,
    id char(28) NOT NULL,
    "user" integer DEFAULT NULL,
    PRIMARY KEY (source, id)
);

CREATE TABLE domains (
    source integer NOT NULL,
    name varchar(255) NOT NULL,
    PRIMARY KEY (source, name)
);

CREATE TABLE event_stream_queue (
    timestamp timestamp NOT NULL,
    event bytea NOT NULL
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

CREATE TABLE smart_events (
    source integer NOT NULL,
    id SERIAL,
    name varchar(255) NOT NULL DEFAULT '',
    event varchar(50) NOT NULL DEFAULT '',
    pages varchar(65535) NOT NULL,
    buttons varchar(65535) NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO smart_events VALUES
    (1, 50, 'View Nissan Car', 'pageview', '[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]', 'null'),(1,51, 'Configure a Nissan Car', 'click', '[{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"cars/nissan/\",\"Domain\":\"english.example.com\"},{\"Field\":\"path\",\"Operator\":\"StartsWith\",\"Value\":\"auto/nissan/\",\"Domain\":\"italian.example.com\"}]', '[{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configure your car\",\"Domain\":\"english.example.com\"},{\"Field\":\"text\",\"Operator\":\"Equals\",\"Value\":\"Configura la tua auto\",\"Domain\":\"italian.example.com\"}]'),
    (1, 52, 'Click on Login Button', 'click', 'null', '[{\"Field\":\"text\",\"Operator\":\"Contains\",\"Value\":\"Log in\"}]');

CREATE TABLE connections_mappings (
    id SERIAL,
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    "in" text NOT NULL,
    predefined_func integer NOT NULL DEFAULT 0,
    source_code text NOT NULL,
    out text NOT NULL,
    PRIMARY KEY (id)
);

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

CREATE TABLE users (
    source integer NOT NULL,
    id integer NOT NULL,
    device char(28) DEFAULT NULL,
    PRIMARY KEY (source, id)
);
