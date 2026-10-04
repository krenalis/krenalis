CREATE TABLE IF NOT EXISTS krenalis_simulated_system_records (
    simulated_system VARCHAR NOT NULL,
    external_id VARCHAR NOT NULL,
    updated_at TIMESTAMP(6) WITHOUT TIME ZONE NOT NULL,
    data JSONB NOT NULL,
    PRIMARY KEY (simulated_system, external_id)
);
