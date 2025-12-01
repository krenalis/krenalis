-- TODO(Gianluca): fill this file with update queries.

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
