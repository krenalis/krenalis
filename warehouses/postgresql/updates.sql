-- Copyright 2025 Open2b. All rights reserved.
-- Use of this source code is governed by the MIT license
-- that can be found in the LICENSE file.

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

ALTER TABLE meergo_destination_profiles RENAME __external_id__        TO _external_id;
ALTER TABLE meergo_destination_profiles RENAME __out_matching_value__ TO _out_matching_value;
ALTER TABLE meergo_destination_profiles RENAME __pipeline__           TO _pipeline;
ALTER TABLE meergo_identities           RENAME __anonymous_ids__      TO _anonymous_ids;
ALTER TABLE meergo_identities           RENAME __cluster__            TO _cluster;
ALTER TABLE meergo_identities           RENAME __connection__         TO _connection;
ALTER TABLE meergo_identities           RENAME __execution__          TO _execution;
ALTER TABLE meergo_identities           RENAME __identity_id__        TO _identity_id;
ALTER TABLE meergo_identities           RENAME __is_anonymous__       TO _is_anonymous;
ALTER TABLE meergo_identities           RENAME __last_change_time__   TO _last_change_time;
ALTER TABLE meergo_identities           RENAME __mpid__               TO _mpid;
ALTER TABLE meergo_identities           RENAME __pipeline__           TO _pipeline;
ALTER TABLE meergo_identities           RENAME __pk__                 TO _pk;
ALTER TABLE profiles                    RENAME __last_change_time__   TO _last_change_time;
ALTER TABLE profiles                    RENAME __mpid__               TO _mpid;

-- NOTE: replace 'meergo_profiles_0' with the correct name of the table you
-- currently have in your data warehouse.
ALTER TABLE meergo_profiles_0 RENAME __identities__       TO _identities;
ALTER TABLE meergo_profiles_0 RENAME __last_change_time__ TO _last_change_time;
ALTER TABLE meergo_profiles_0 RENAME __mpid__             TO _mpid;

-- NOTE: replace 'meergo_profiles_0' with the correct name of the table you
-- currently have in your data warehouse.
ALTER TABLE meergo_identities RENAME _last_change_time TO _updated_at;
ALTER TABLE profiles          RENAME _last_change_time TO _updated_at;
ALTER TABLE meergo_profiles_0 RENAME _last_change_time TO _updated_at;

ALTER TABLE meergo_identities RENAME COLUMN _execution TO _run;

ALTER INDEX last_change_time_idx RENAME TO updated_atx;
