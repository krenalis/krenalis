-- TODO(Gianluca): fill this file with update queries.

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
