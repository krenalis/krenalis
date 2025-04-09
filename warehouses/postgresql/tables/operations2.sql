DO $$
    BEGIN
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = '_operation2') THEN
        CREATE TYPE _operation2 AS ENUM ('IdentityResolution', 'AlterUserColumns');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS _operations2 (
    id uuid NOT NULL,
    operation_type _operation2,
    completed_at timestamp(3),
    error text NOT NULL DEFAULT '',
    PRIMARY KEY ("id")
);