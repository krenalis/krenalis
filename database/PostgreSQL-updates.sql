INSERT INTO connectors (name, type, webhooks_per) VALUES 
    ('UISample', 'EventStream', 'None');

INSERT INTO connectors (name, type, webhooks_per) VALUES
    ('UISample', 'EventStream', 'None');

ALTER TABLE connections
    CHANGE COLUMN workspace workspace integer NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    CHANGE COLUMN connector connector integer NOT NULL REFERENCES connectors ON DELETE RESTRICT,
    CHANGE COLUMN storage storage integer DEFAULT NULL REFERENCES connectors ON DELETE SET NULL,
    CHANGE COLUMN stream stream integer DEFAULT NULL REFERENCES connectors ON DELETE SET NULL;

ALTER TABLE connections_imports
    CHANGE COLUMN connection connection integer NOT NULL REFERENCES connections ON DELETE CASCADE;

ALTER TABLE connections_keys
    CHANGE COLUMN connection connection INT NOT NULL REFERENCES connections ON DELETE CASCADE;

ALTER TABLE connections_stats
    CHANGE COLUMN connection connection INT NOT NULL REFERENCES connections ON DELETE CASCADE;

ALTER TABLE connections_stats_events
    CHANGE COLUMN source source integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    CHANGE COLUMN server server integer DEFAULT NULL REFERENCES connectors ON DELETE SET NULL,
    CHANGE COLUMN stream stream integer DEFAULT NULL REFERENCES connectors ON DELETE SET NULL;

ALTER TABLE connections_users
    CHANGE COLUMN connection connection integer NOT NULL REFERENCES connections ON DELETE CASCADE;
