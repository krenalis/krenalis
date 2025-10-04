-- TODO(Gianluca): fill this file with update queries.

ALTER TABLE user_schema_primary_sources RENAME TO primary_sources;

--

BEGIN;
ALTER TYPE notification_name
    ADD VALUE IF NOT EXISTS 'AddMember' BEFORE 'CreateAccessKey';
ALTER TYPE notification_name
    ADD VALUE IF NOT EXISTS 'DeleteMember' AFTER 'DeleteEventWriteKey';
COMMIT;

--

BEGIN;

WITH mapping(label, code) AS (
    VALUES
        ('.NET', 'dotnet'),
        ('Android', 'android'),
        ('ClickHouse', 'clickhouse'),
        ('CSV', 'csv'),
        ('Dummy', 'dummy'),
        ('Excel', 'excel'),
        ('Filesystem', 'filesystem'),
        ('Go', 'go'),
        ('Google Analytics', 'google-analytics'),
        ('HTTP GET', 'http-get'),
        ('HTTP POST', 'http-post'),
        ('HubSpot', 'hubspot'),
        ('Java', 'java'),
        ('JavaScript', 'javascript'),
        ('JSON', 'json'),
        ('Kafka', 'kafka'),
        ('Klaviyo', 'klaviyo'),
        ('Mailchimp', 'mailchimp'),
        ('Meergo API', 'meergo-api'),
        ('Mixpanel', 'mixpanel'),
        ('MySQL', 'mysql'),
        ('Node.js', 'nodejs'),
        ('Parquet', 'parquet'),
        ('PostgreSQL', 'postgresql'),
        ('Python', 'python'),
        ('RabbitMQ', 'rabbitmq'),
        ('RudderStack', 'rudderstack'),
        ('S3', 's3'),
        ('Segment', 'segment'),
        ('SFTP', 'sftp'),
        ('Snowflake', 'snowflake'),
        ('Stripe', 'stripe'),
        ('UISample', 'ui-sample')
),
     updated_connections AS (
         UPDATE connections c
             SET connector = m.code
             FROM mapping m
             WHERE c.connector = m.label
             RETURNING 1
     ),
     updated_actions AS (
         UPDATE actions a
             SET format = m.code
             FROM mapping m
             WHERE a.format = m.label
             RETURNING 1
     )
UPDATE accounts a
SET connector = m.code
FROM mapping m
WHERE a.connector = m.label;

COMMIT;
