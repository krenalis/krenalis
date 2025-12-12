-- rename text to string

UPDATE workspaces
SET alter_profile_schema_schema = REPLACE(alter_profile_schema_schema::text, '"kind": "text"', '"kind": "string"')::jsonb
WHERE alter_profile_schema_schema IS NOT NULL AND alter_profile_schema_schema::text LIKE '%"kind": "text"%';

UPDATE workspaces
SET profile_schema = REPLACE(profile_schema::text, '"kind": "text"', '"kind": "string"')::jsonb
WHERE profile_schema::text LIKE '%"kind": "text"%';

UPDATE pipelines
SET in_schema = REPLACE(in_schema::text, '"kind": "text"', '"kind": "string"')::jsonb
WHERE in_schema IS NOT NULL AND  in_schema::text LIKE '%"kind": "text"%';

UPDATE pipelines
SET out_schema = REPLACE(out_schema::text, '"kind": "text"', '"kind": "string"')::jsonb
WHERE out_schema IS NOT NULL AND  out_schema::text LIKE '%"kind": "text"%';

-- rename charLen to maxLength

UPDATE workspaces
SET alter_profile_schema_schema = REPLACE(alter_profile_schema_schema::text, '"charLen": ', '"maxLength": ')::jsonb
WHERE alter_profile_schema_schema IS NOT NULL AND alter_profile_schema_schema::text LIKE '%"charLen": %';

UPDATE workspaces
SET profile_schema = REPLACE(profile_schema::text, '"charLen": ', '"maxLength": ')::jsonb
WHERE profile_schema::text LIKE '%"charLen": %';

UPDATE pipelines
SET in_schema = REPLACE(in_schema::text, '"charLen": ', '"maxLength": ')::jsonb
WHERE in_schema IS NOT NULL AND  in_schema::text LIKE '%"charLen": %';

UPDATE pipelines
SET out_schema = REPLACE(out_schema::text, '"charLen": ', '"maxLength": ')::jsonb
WHERE out_schema IS NOT NULL AND  out_schema::text LIKE '%"charLen": %';

-- rename byteLen to maxByteLength

UPDATE workspaces
SET alter_profile_schema_schema = REPLACE(alter_profile_schema_schema::text, '"byteLen": ', '"maxByteLength": ')::jsonb
WHERE alter_profile_schema_schema IS NOT NULL AND alter_profile_schema_schema::text LIKE '%"byteLen": %';

UPDATE workspaces
SET profile_schema = REPLACE(profile_schema::text, '"byteLen": ', '"maxByteLength": ')::jsonb
WHERE profile_schema::text LIKE '%"byteLen": %';

UPDATE pipelines
SET in_schema = REPLACE(in_schema::text, '"byteLen": ', '"maxByteLength": ')::jsonb
WHERE in_schema IS NOT NULL AND  in_schema::text LIKE '%"byteLen": %';

UPDATE pipelines
SET out_schema = REPLACE(out_schema::text, '"byteLen": ', '"maxByteLength": ')::jsonb
WHERE out_schema IS NOT NULL AND  out_schema::text LIKE '%"byteLen": %';

-- rename regexp to pattern

UPDATE workspaces
SET alter_profile_schema_schema = REPLACE(alter_profile_schema_schema::text, '"regexp": ', '"pattern": ')::jsonb
WHERE alter_profile_schema_schema IS NOT NULL AND alter_profile_schema_schema::text LIKE '%"regexp": %';

UPDATE workspaces
SET profile_schema = REPLACE(profile_schema::text, '"regexp": ', '"pattern": ')::jsonb
WHERE profile_schema::text LIKE '%"regexp": %';

UPDATE pipelines
SET in_schema = REPLACE(in_schema::text, '"regexp": ', '"pattern": ')::jsonb
WHERE in_schema IS NOT NULL AND  in_schema::text LIKE '%"regexp": %';

UPDATE pipelines
SET out_schema = REPLACE(out_schema::text, '"regexp": ', '"pattern": ')::jsonb
WHERE out_schema IS NOT NULL AND  out_schema::text LIKE '%"regexp": %';

-- rename inet to ip

UPDATE workspaces
SET alter_profile_schema_schema = REPLACE(alter_profile_schema_schema::text, '"kind": "inet"', '"kind": "ip"')::jsonb
WHERE alter_profile_schema_schema IS NOT NULL AND alter_profile_schema_schema::text LIKE '%"kind": "inet"%';

UPDATE workspaces
SET profile_schema = REPLACE(profile_schema::text, '"kind": "inet"', '"kind": "ip"')::jsonb
WHERE profile_schema::text LIKE '%"kind": "inet"%';

UPDATE pipelines
SET in_schema = REPLACE(in_schema::text, '"kind": "inet"', '"kind": "ip"')::jsonb
WHERE in_schema IS NOT NULL AND  in_schema::text LIKE '%"kind": "inet"%';

UPDATE pipelines
SET out_schema = REPLACE(out_schema::text, '"kind": "inet"', '"kind": "ip"')::jsonb
WHERE out_schema IS NOT NULL AND  out_schema::text LIKE '%"kind": "inet"%';

-- rename maxByteLength to maxBytes

UPDATE pipelines
SET in_schema = REPLACE(in_schema::text, '"maxByteLength": ', '"maxBytes": ')::jsonb
WHERE in_schema IS NOT NULL AND  in_schema::text LIKE '%"maxByteLength": %';

UPDATE pipelines
SET out_schema = REPLACE(out_schema::text, '"maxByteLength": ', '"maxBytes": ')::jsonb
WHERE out_schema IS NOT NULL AND  out_schema::text LIKE '%"maxByteLength": %';

-- replace uint with unsigned int

UPDATE workspaces
SET alter_profile_schema_schema = REPLACE(alter_profile_schema_schema::text, '"kind": "uint"', '"kind": "int", "unsigned": true')::jsonb
WHERE alter_profile_schema_schema IS NOT NULL AND alter_profile_schema_schema::text LIKE '%"kind": "uint"%';

UPDATE workspaces
SET profile_schema = REPLACE(profile_schema::text, '"kind": "uint"', '"kind": "int", "unsigned": true')::jsonb
WHERE profile_schema::text LIKE '%"kind": "uint"%';

UPDATE pipelines
SET in_schema = REPLACE(in_schema::text, '"kind": "uint"', '"kind": "int", "unsigned": true')::jsonb
WHERE in_schema IS NOT NULL AND  in_schema::text LIKE '%"kind": "uint"%';

UPDATE pipelines
SET out_schema = REPLACE(out_schema::text, '"kind": "uint"', '"kind": "int", "unsigned": true')::jsonb
WHERE out_schema IS NOT NULL AND  out_schema::text LIKE '%"kind": "uint"%';

-- rename 'executions' to 'runs'

ALTER TABLE pipelines_executions RENAME TO pipelines_runs;

DROP INDEX pipelines_executions_function_idx;

CREATE INDEX pipelines_runs_function_idx
    ON pipelines_runs (function)
    WHERE function != '' AND end_time IS NOT NULL;

ALTER TABLE pipelines_runs
    RENAME CONSTRAINT actions_executions_pkey TO pipelines_runs_pkey;

ALTER TABLE pipelines_runs
    RENAME CONSTRAINT actions_executions_action_fkey
        TO pipelines_runs_pipeline_fkey;

DROP INDEX actions_executions_function_idx;

ALTER TYPE notification_name RENAME VALUE 'ExecutePipeline' TO 'RunPipeline';
ALTER TYPE notification_name RENAME VALUE 'EndPipelineExecution' TO 'EndPipelineRun';

-- convert warehouse settings columns from varchar to jsonb

BEGIN;

ALTER TABLE workspaces
    ALTER COLUMN warehouse_settings
        TYPE jsonb
        USING warehouse_settings::jsonb,
    ALTER COLUMN warehouse_mcp_settings
        TYPE jsonb
        USING warehouse_mcp_settings::jsonb;

ALTER TABLE workspaces
    ALTER COLUMN warehouse_mcp_settings
        SET DEFAULT 'null'::jsonb;

COMMIT;

ALTER TABLE connections ALTER COLUMN settings TYPE jsonb USING NULLIF(btrim(settings), '')::jsonb;
ALTER TABLE pipelines ALTER COLUMN format_settings TYPE jsonb USING NULLIF(btrim(format_settings), '')::jsonb;
