//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"strings"

	"chichi/apis/postgres"
	"chichi/apis/types"
)

// loadState loads the state from the database.
func (s *stateKeeper) loadState() error {

	// Read all connectors.
	connectors := map[int]*Connector{}
	err := s.db.QueryScan("SELECT id, name, type, logo_url, webhooks_per, oauth_url, oauth_client_id,"+
		" oauth_client_secret, oauth_token_endpoint, oauth_default_token_type, oauth_default_expires_in,"+
		" oauth_forced_expires_in FROM connectors", func(rows *postgres.Rows) error {
		for rows.Next() {
			c := Connector{}
			oauth := ConnectorOAuth{}
			if err := rows.Scan(&c.id, &c.name, &c.typ, &c.logoURL, &c.webhooksPer, &oauth.URL, &oauth.ClientID, &oauth.ClientSecret,
				&oauth.TokenEndpoint, &oauth.DefaultTokenType, &oauth.DefaultExpiresIn, &oauth.ForcedExpiresIn); err != nil {
				return err
			}
			if oauth.URL != "" {
				c.oAuth = &oauth
			}
			connectors[c.id] = &c
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.Connectors = newConnectors(s.APIs, &connectorsState{ids: connectors})

	// Read all accounts.
	accounts := map[int]*Account{}
	err = s.db.QueryScan("SELECT id, name, email, internal_ips FROM accounts", func(rows *postgres.Rows) error {
		var id int
		var name, email, ips string
		for rows.Next() {
			if err := rows.Scan(&id, &name, &email, &ips); err != nil {
				return err
			}
			account := &Account{
				apis:        s.APIs,
				db:          s.db,
				chDB:        s.chDB,
				id:          id,
				name:        name,
				email:       email,
				internalIPs: strings.Fields(ips),
			}
			account.Workspaces = newWorkspaces(account, &workspacesState{ids: map[int]*Workspace{}})
			accounts[id] = account
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.Accounts = newAccounts(s.APIs, &accountsState{ids: accounts})

	// Read all workspaces.
	workspaces := map[int]*Workspace{}
	err = s.db.QueryScan("SELECT id, account, user_schema, group_schema FROM workspaces",
		func(rows *postgres.Rows) error {
			var id, accountID int
			var userSchema, groupSchema string
			for rows.Next() {
				if err := rows.Scan(&id, &accountID, &userSchema, &groupSchema); err != nil {
					return err
				}
				account := accounts[accountID]
				workspace := &Workspace{
					db:        s.db,
					chDB:      s.chDB,
					id:        id,
					account:   account,
					resources: &resourcesState{ids: map[int]*Resource{}},
				}
				if userSchema != "" {
					workspace.schema.user, err = types.ParseSchema(strings.NewReader(userSchema), nil)
					if err != nil {
						return err
					}
				}
				if groupSchema != "" {
					workspace.schema.group, err = types.ParseSchema(strings.NewReader(groupSchema), nil)
					if err != nil {
						return err
					}
				}
				workspace.schemaSources.user = userSchema
				workspace.schemaSources.group = groupSchema
				workspace.Connections = newConnections(workspace, &connectionsState{ids: map[int]*Connection{}})
				workspace.EventTypes = newEventTypes(workspace, &eventTypesState{ids: map[int]*EventType{}})
				workspace.EventDataTypes = newEventDataTypes(workspace, &eventDataTypesState{names: map[string]*EventDataType{}})
				workspace.EventListeners = &EventListeners{workspace}
				account.Workspaces.state.ids[id] = workspace
				workspaces[id] = workspace
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all resources.
	resources := map[int]*Resource{}
	err = s.db.QueryScan("SELECT id, workspace, connector, code, access_token, refresh_token, expires_in FROM resources",
		func(rows *postgres.Rows) error {
			for rows.Next() {
				r := Resource{}
				var workspaceID, connectorID int
				if err := rows.Scan(&r.id, &workspaceID, &connectorID, &r.code, &r.accessToken, &r.refreshToken, &r.expiresIn); err != nil {
					return err
				}
				r.workspace = workspaces[workspaceID]
				r.connector = connectors[connectorID]
				r.workspace.resources.ids[r.id] = &r
				resources[r.id] = &r
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all connections.
	connections := map[int]*Connection{}
	err = s.db.QueryScan("SELECT id, workspace, name, role, enabled, connector, COALESCE(storage, 0),"+
		" COALESCE(stream, 0), resource, website_host, user_cursor, identity_column, timestamp_column, settings,"+
		" schema, users_query FROM connections", func(rows *postgres.Rows) error {
		for rows.Next() {
			var workspaceID, connector, storage, stream, resource int
			var rawSchema string
			c := Connection{}
			if err := rows.Scan(&c.id, &workspaceID, &c.name, &c.role, &c.enabled, &connector, &storage, &stream, &resource,
				&c.websiteHost, &c.userCursor, &c.identityColumn, &c.timestampColumn, &c.settings, &rawSchema,
				&c.usersQuery); err != nil {
				return err
			}
			workspace := workspaces[workspaceID]
			c.account = workspace.account
			c.workspace = workspace
			c.connector = connectors[connector]
			if storage > 0 {
				if s, ok := connections[storage]; ok {
					c.storage = s
				} else {
					c.storage = &Connection{}
					connections[storage] = c.storage
				}
			}
			if stream > 0 {
				if s, ok := connections[stream]; ok {
					c.stream = s
				} else {
					c.stream = &Connection{}
					connections[stream] = c.stream
				}
			}
			if resource > 0 {
				c.resource = resources[resource]
			}
			if c.connector.typ == ServerType {
				c.keys = []string{}
			}
			if len(rawSchema) > 0 {
				c.schema, err = types.ParseSchema(strings.NewReader(rawSchema), nil)
				if err != nil {
					// TODO(marco) disable the connection instead of returning an error
					return err
				}
			}
			c.mappings = []*Mapping{}
			connection, ok := connections[c.id]
			if ok {
				*connection = c
			} else {
				connection = &Connection{}
				*connection = c
			}
			workspace.Connections.state.ids[c.id] = connection
			connections[c.id] = connection
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Read all keys.
	err = s.db.QueryScan(`SELECT connection, value FROM connections_keys ORDER BY connection, creation_time`,
		func(rows *postgres.Rows) error {
			for rows.Next() {
				var connectionID int
				var value string
				if err := rows.Scan(&connectionID, &value); err != nil {
					return err
				}
				connection := connections[connectionID]
				connection.keys = append(connection.keys, value)
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all event types.
	err = s.db.QueryScan("SELECT workspace, id, name, description, schema FROM event_types",
		func(rows *postgres.Rows) error {
			for rows.Next() {
				t := EventType{}
				var workspaceID int
				if err := rows.Scan(&workspaceID, &t.id, &t.name, &t.description, &t.schemaSource); err != nil {
					return err
				}
				if t.schemaSource != "" {
					t.schema, err = types.ParseSchema(strings.NewReader(t.schemaSource), nil)
					if err != nil {
						// TODO(marco) disable the type instead of returning an error?
						return err
					}
				}
				workspaces[workspaceID].EventTypes.state.ids[t.id] = &t
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all event data types.
	err = s.db.QueryScan("SELECT workspace, name, description, schema FROM event_data_types",
		func(rows *postgres.Rows) error {
			for rows.Next() {
				t := EventDataType{}
				var workspaceID int
				if err := rows.Scan(&workspaceID, &t.name, &t.description, &t.schemaSource); err != nil {
					return err
				}
				if t.schemaSource != "" {
					t.schema, err = types.ParseSchema(strings.NewReader(t.schemaSource), nil)
					if err != nil {
						// TODO(marco) disable the type instead of returning an error?
						return err
					}
				}
				workspaces[workspaceID].EventDataTypes.state.names[t.name] = &t
			}
			return nil
		})
	if err != nil {
		return err
	}

	s.eventCollector, err = newEventCollector(context.Background(), connections, nil)
	if err != nil {
		return err
	}

	s.eventProcessor = newEventProcessor(s.db, s.chDB, connections)
	go s.eventProcessor.Run(context.Background())

	// Read the mappings.
	var mappings []*Mapping
	schemas := [][]byte{}
	connectionIDs := []int{}
	err = s.db.QueryScan("SELECT id, connection, \"in\", source_code, out FROM connections_mappings", func(rows *postgres.Rows) error {
		for rows.Next() {
			m := &Mapping{}
			var schema []byte
			var connectionID int
			err := rows.Scan(&m.id, &connectionID, &schema, &m.sourceCode, &m.out)
			if err != nil {
				return err
			}
			mappings = append(mappings, m)
			schemas = append(schemas, schema)
			connectionIDs = append(connectionIDs, connectionID)
		}
		return nil
	})
	for i, m := range mappings {
		var err error
		m.in, err = types.ParseSchema(bytes.NewReader(schemas[i]), nil)
		if err != nil {
			return err
		}
		conn := connections[connectionIDs[i]]
		m.connection = conn
		conn.mappings = append(conn.mappings, m)
	}

	s.workspaces = workspaces
	s.connections = connections
	s.resources = resources

	return err
}
