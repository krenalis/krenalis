
-- 1
CREATE TYPE sending_mode_new AS ENUM ('Client', 'Server', 'ClientAndServer');

-- 2
ALTER TABLE connections
ADD COLUMN sending_mode_tmp sending_mode_new;

-- 3
UPDATE connections
SET sending_mode_tmp =
    CASE sending_mode
        WHEN 'Device' THEN 'Client'
        WHEN 'Cloud' THEN 'Server'
        WHEN 'Combined' THEN 'ClientAndServer'
        ELSE NULL
    END;

-- 4
ALTER TABLE connections
DROP COLUMN sending_mode;

-- 5
ALTER TABLE connections
RENAME COLUMN sending_mode_tmp TO sending_mode;

-- 6
DROP TYPE sending_mode;

-- 7
ALTER TYPE sending_mode_new RENAME TO sending_mode;
