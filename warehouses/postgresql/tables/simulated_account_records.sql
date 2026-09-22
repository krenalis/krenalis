CREATE TABLE IF NOT EXISTS krenalis_simulated_account_records (
    simulated_account_id VARCHAR NOT NULL,
    external_id VARCHAR NOT NULL,
    data JSONB NOT NULL,
    PRIMARY KEY (simulated_account_id, external_id)
);
