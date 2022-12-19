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
	"chichi/connector"
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
				workspace.Transformations = newTransformations(workspace, &transformationsState{ofConnection: map[int][]*Transformation{}})
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
			connection, ok := connections[c.id]
			if ok {
				*connection = c
			} else {
				connection = &Connection{}
				*connection = c
			}
			workspace.Connections.state.ids[c.id] = connection
			workspace.Transformations.state.ofConnection[c.id] = []*Transformation{}
			connections[c.id] = connection
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Read all keys.
	err = s.db.QueryScan(`SELECT connection, value FROM connections_keys ORDER BY connection`, func(rows *postgres.Rows) error {
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

	// Read the source event stream collectors and the source connections that
	// send the events into the stream with their keys.
	var streams []*eventCollectorStream
	err = s.db.QueryScan(
		"SELECT s.id, co.name AS connector, s.settings, ci.id AS event_collector_producer, ci.type, k.value\n"+
			"FROM connections AS s\n"+
			"INNER JOIN connectors AS co ON co.id = s.connector\n"+
			"INNER JOIN connections AS ci ON ci.stream = s.id\n"+
			"INNER JOIN connections_keys AS k ON k.connection = ci.id\n"+
			"WHERE s.type = 'EventStream' AND s.role = 'Source' AND s.settings <> '' AND s.enabled AND ci.enabled",
		func(rows *postgres.Rows) error {
		Rows:
			for rows.Next() {
				var stream eventCollectorStream
				var producerID int
				var producerType ConnectorType
				var producerKey string
				if err := rows.Scan(&stream.ID, &stream.Connector, &stream.Settings, &producerID, &producerType, &producerKey); err != nil {
					return err
				}
				for _, s := range streams {
					if s.ID == stream.ID {
						for _, p := range s.Producers {
							if p.ID == producerID {
								p.Keys = append(p.Keys, producerKey)
								continue Rows
							}
						}
						s.Producers = append(s.Producers, &eventCollectorProducer{
							ID:   producerID,
							Type: connector.Type(producerType),
							Keys: []string{producerKey},
						})
						continue Rows
					}
				}
				stream.Producers = []*eventCollectorProducer{{
					ID:   producerID,
					Type: connector.Type(producerType),
					Keys: []string{producerKey},
				}}
				streams = append(streams, &stream)
			}
			return nil
		})
	if err != nil {
		return err
	}

	s.eventCollector, err = newEventCollector(context.Background(), streams)
	if err != nil {
		return err
	}

	s.eventProcessor = newEventProcessor(s.db, s.chDB, connections)
	go s.eventProcessor.Run(context.Background())

	// Read all the transformations.
	var transformations []*Transformation
	schemas := [][]byte{}
	err = s.db.QueryScan("SELECT id, connection, \"in\", source_code, out FROM transformations", func(rows *postgres.Rows) error {
		for rows.Next() {
			t := &Transformation{}
			var schema []byte
			err := rows.Scan(&t.ID, &t.Connection, &schema, &t.SourceCode, &t.Out)
			if err != nil {
				return err
			}
			transformations = append(transformations, t)
			schemas = append(schemas, schema)
		}
		return nil
	})
	for i, t := range transformations {
		var err error
		t.In, err = types.ParseSchema(bytes.NewReader(schemas[i]), nil)
		if err != nil {
			return err
		}
		c := t.Connection
		ws := connections[c].workspace
		ws.Transformations.state.ofConnection[c] = append(
			ws.Transformations.state.ofConnection[c], t,
		)
	}

	s.workspaces = workspaces
	s.connections = connections
	s.resources = resources

	return err
}
