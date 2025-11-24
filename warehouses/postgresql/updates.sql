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
