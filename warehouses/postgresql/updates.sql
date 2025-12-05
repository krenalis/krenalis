-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

-- TODO: fare il revert dei rinomini qui dentro, poi aggiungere in fondo le
-- query per il rinomino corretto.

ALTER TABLE _destinations_users RENAME TO _destinations_profiles;
ALTER TABLE _user_identities RENAME TO _identities;
ALTER TABLE _identities RENAME COLUMN __muid__ TO __mpid__;
ALTER TABLE _user_schema_versions RENAME TO _profile_schema_versions;
ALTER TABLE _users_0 RENAME TO _profiles_0;
ALTER TABLE _profiles_0 RENAME COLUMN __muid__ TO __mpid__;
ALTER TABLE users RENAME TO profiles;
ALTER VIEW profiles RENAME COLUMN __muid__ TO __mpid__;
ALTER TABLE events RENAME COLUMN muid TO mpid;
ALTER TYPE _operation RENAME VALUE 'AlterUserSchema' TO 'AlterProfileSchema';

ALTER TABLE _destinations_profiles RENAME TO meergo_destination_profiles;
ALTER TABLE _operations RENAME TO meergo_system_operations;
ALTER TABLE _profile_schema_versions RENAME TO meergo_profile_schema_versions;
ALTER TABLE events RENAME TO meergo_events;
ALTER TABLE _clusters_to_merge RENAME TO meergo_graph_merge_clusters;
ALTER TABLE _edges RENAME TO meergo_graph_edges;
ALTER TABLE _identities RENAME TO meergo_identities;
ALTER TABLE _profiles_3 RENAME TO meergo_profiles_3;
ALTER TYPE _operation RENAME TO system_operation_type;
CREATE VIEW events AS SELECT * FROM meergo_events;

ALTER TABLE meergo_destination_profiles RENAME COLUMN __action__ TO __pipeline__;
ALTER TABLE meergo_identities RENAME COLUMN __action__ TO __pipeline__;
