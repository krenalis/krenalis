//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"
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
	err = s.db.QueryScan("SELECT id, account,  warehouse_type, warehouse_settings, schema FROM workspaces",
		func(rows *postgres.Rows) error {
			var id, accountID int
			var warehouseType *warehouses.Type
			var warehouseSettings, schema []byte
			for rows.Next() {
				if err := rows.Scan(&id, &accountID, &warehouseType, &warehouseSettings, &schema); err != nil {
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
				if warehouseType != nil {
					workspace.warehouse, err = openWarehouse(*warehouseType, warehouseSettings)
					if err != nil {
						log.Fatalf("cannot open data warehouse of workspace %d: %s", id, err)
					}
					workspace.schema = map[string]*types.Type{}
					if len(schema) > 0 {
						err = json.Unmarshal(schema, &workspace.schema)
						if err != nil {
							log.Fatalf("cannot unmarshal schema of workspace %d: %s", id, err)
						}
					}
				}
				workspace.Connections = newConnections(workspace, &connectionsState{ids: map[int]*Connection{}})
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
				c.schema, err = types.Parse(rawSchema, nil)
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

	// defaultStream receives events from the collector for which the source connector
	// does not have its own stream.
	defaultStream := newPostgresEventStream(context.Background(), s.db)

	s.eventCollector, err = newEventCollector(context.Background(), connections, defaultStream)
	if err != nil {
		return err
	}

	s.eventProcessor = newEventProcessor(s.db, s.chDB, connections, defaultStream)
	go s.eventProcessor.Run(context.Background())

	// Read the mappings.
	var mappings []*Mapping
	inSchemas := [][]byte{}
	outSchemas := [][]byte{}
	connectionIDs := []int{}
	err = s.db.QueryScan("SELECT id, connection, \"in\", predefined_func, source_code, out FROM connections_mappings", func(rows *postgres.Rows) error {
		for rows.Next() {
			m := &Mapping{}
			var inSchema, outSchema []byte
			var connectionID int
			err := rows.Scan(&m.id, &connectionID, &inSchema, &m.predefinedFunc, &m.sourceCode, &outSchema)
			if err != nil {
				return err
			}
			mappings = append(mappings, m)
			inSchemas = append(inSchemas, inSchema)
			outSchemas = append(outSchemas, outSchema)
			connectionIDs = append(connectionIDs, connectionID)
		}
		return nil
	})
	for i, m := range mappings {
		var err error
		m.in, err = types.Parse(string(inSchemas[i]), nil)
		if err != nil {
			return err
		}
		m.out, err = types.Parse(string(outSchemas[i]), nil)
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
