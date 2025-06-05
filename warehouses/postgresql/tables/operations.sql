DO $$
    BEGIN
    IF NOT EXISTS (SELECT FROM pg_type WHERE typname = '_operation') THEN
        CREATE TYPE _operation AS ENUM ('IdentityResolution', 'AlterUserSchema');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS _operations (
    id uuid NOT NULL,
    operation_type _operation,
    completed_at timestamp(3),
    error text NOT NULL DEFAULT '',
    PRIMARY KEY ("id")
);
