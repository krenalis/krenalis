//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"chichi/apis/postgres"
	"chichi/apis/types"
)

// load loads the state from the database.
func (s *State) load() error {

	// Read all connectors.
	s.connectors = map[int]*Connector{}
	err := s.db.QueryScan("SELECT id, name, type, logo_url, webhooks_per, oauth_url, oauth_client_id,"+
		" oauth_client_secret, oauth_token_endpoint, oauth_default_token_type, oauth_default_expires_in,"+
		" oauth_forced_expires_in FROM connectors", func(rows *postgres.Rows) error {
		for rows.Next() {
			c := Connector{}
			oauth := ConnectorOAuth{}
			if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.LogoURL, &c.WebhooksPer, &oauth.URL, &oauth.ClientID, &oauth.ClientSecret,
				&oauth.TokenEndpoint, &oauth.DefaultTokenType, &oauth.DefaultExpiresIn, &oauth.ForcedExpiresIn); err != nil {
				return err
			}
			if oauth.URL != "" {
				c.OAuth = &oauth
			}
			s.connectors[c.ID] = &c
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Read all accounts.
	s.accounts = map[int]*Account{}
	err = s.db.QueryScan("SELECT id, name, email, internal_ips FROM accounts", func(rows *postgres.Rows) error {
		var id int
		var name, email, ips string
		for rows.Next() {
			if err := rows.Scan(&id, &name, &email, &ips); err != nil {
				return err
			}
			account := &Account{
				mu:          new(sync.Mutex),
				ID:          id,
				Name:        name,
				Email:       email,
				InternalIPs: strings.Fields(ips),
			}
			account.workspaces = map[int]*Workspace{}
			s.accounts[id] = account
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Read all workspaces.
	s.workspaces = map[int]*Workspace{}
	err = s.db.QueryScan("SELECT id, account, warehouse_type, warehouse_settings, schemas FROM workspaces",
		func(rows *postgres.Rows) error {
			var id, accountID int
			var warehouseType *WarehouseType
			var warehouseSettings, schemas []byte
			for rows.Next() {
				if err := rows.Scan(&id, &accountID, &warehouseType, &warehouseSettings, &schemas); err != nil {
					return err
				}
				account := s.accounts[accountID]
				workspace := &Workspace{
					mu:        new(sync.Mutex),
					ID:        id,
					account:   account,
					resources: map[int]*Resource{},
				}
				if warehouseType != nil {
					workspace.Warehouse, err = openWarehouse(*warehouseType, warehouseSettings)
					if err != nil {
						log.Fatalf("cannot open data warehouse of workspace %d: %s", id, err)
					}
					workspace.Schemas = map[string]*types.Type{}
					if len(schemas) > 0 {
						err = json.Unmarshal(schemas, &workspace.Schemas)
						if err != nil {
							log.Fatalf("cannot unmarshal schemas of workspace %d: %s", id, err)
						}
					}
				}
				workspace.connections = map[int]*Connection{}
				//workspace.EventListeners = &EventListeners{workspace}  // TODO(marco)
				account.workspaces[id] = workspace
				s.workspaces[id] = workspace
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all resources.
	s.resources = map[int]*Resource{}
	err = s.db.QueryScan("SELECT id, workspace, connector, code, access_token, refresh_token, expires_in FROM resources",
		func(rows *postgres.Rows) error {
			for rows.Next() {
				r := Resource{}
				var workspaceID, connectorID int
				if err := rows.Scan(&r.ID, &workspaceID, &connectorID, &r.Code, &r.AccessToken, &r.RefreshToken, &r.ExpiresIn); err != nil {
					return err
				}
				r.mu = new(sync.Mutex)
				r.workspace = s.workspaces[workspaceID]
				r.connector = s.connectors[connectorID]
				r.workspace.resources[r.ID] = &r
				s.resources[r.ID] = &r
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all connections.
	s.connections = map[int]*Connection{}
	err = s.db.QueryScan("SELECT id, workspace, name, role, enabled, connector, COALESCE(storage, 0),"+
		" COALESCE(stream, 0), resource, website_host, user_cursor, identity_column, timestamp_column, settings,"+
		" schema, users_query FROM connections", func(rows *postgres.Rows) error {
		for rows.Next() {
			var workspaceID, connector, storage, stream, resource int
			var rawSchema string
			c := Connection{}
			if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &c.Role, &c.Enabled, &connector, &storage, &stream, &resource,
				&c.WebsiteHost, &c.UserCursor, &c.IdentityColumn, &c.TimestampColumn, &c.Settings, &rawSchema,
				&c.UsersQuery); err != nil {
				return err
			}
			workspace := s.workspaces[workspaceID]
			c.mu = new(sync.Mutex)
			c.account = workspace.account
			c.workspace = workspace
			c.connector = s.connectors[connector]
			if storage > 0 {
				if st, ok := s.connections[storage]; ok {
					c.storage = st
				} else {
					c.storage = &Connection{}
					s.connections[storage] = c.storage
				}
			}
			if stream > 0 {
				if st, ok := s.connections[stream]; ok {
					c.stream = st
				} else {
					c.stream = &Connection{}
					s.connections[stream] = c.stream
				}
			}
			if resource > 0 {
				c.resource = s.resources[resource]
			}
			if c.connector.Type == ServerType {
				c.Keys = []string{}
			}
			if len(rawSchema) > 0 {
				c.Schema, err = types.Parse(rawSchema, nil)
				if err != nil {
					// TODO(marco) disable the connection instead of returning an error
					return err
				}
			}
			c.mappings = []*Mapping{}
			connection, ok := s.connections[c.ID]
			if ok {
				*connection = c
			} else {
				connection = &Connection{}
				*connection = c
			}
			workspace.connections[c.ID] = connection
			s.connections[c.ID] = connection
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
				connection := s.connections[connectionID]
				connection.Keys = append(connection.Keys, value)
			}
			return nil
		})
	if err != nil {
		return err
	}

	// TODO(marco): uncomment the following code.

	// Handle events if the workspace has the "events" schema.
	//if workspace, ok := s.workspaces[1]; ok && workspace.Schemas["events"] != nil {
	//
	//	// defaultStream receives events from the collector for which the source connector
	//	// does not have its own stream.
	//	defaultStream := newPostgresEventStream(context.Background(), s.db)
	//
	//	s.eventCollector, err = newEventCollector(context.Background(), connections, defaultStream)
	//	if err != nil {
	//		return err
	//	}
	//
	//	s.eventProcessor = newEventProcessor(s.db, workspace.Warehouse, connections, defaultStream)
	//	go s.eventProcessor.Run(context.Background())
	//
	//}

	// Read the mappings.
	var mappings []*Mapping
	inSchemas := [][]byte{}
	outSchemas := [][]byte{}
	connectionIDs := []int{}
	err = s.db.QueryScan("SELECT id, connection, \"in\", predefined_func, source_code, out FROM connections_mappings", func(rows *postgres.Rows) error {
		for rows.Next() {
			m := &Mapping{
				mu: new(sync.Mutex),
			}
			var inSchema, outSchema []byte
			var connectionID int
			err := rows.Scan(&m.ID, &connectionID, &inSchema, &m.PredefinedFunc, &m.SourceCode, &outSchema)
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
		m.In, err = types.Parse(string(inSchemas[i]), nil)
		if err != nil {
			return err
		}
		m.Out, err = types.Parse(string(outSchemas[i]), nil)
		if err != nil {
			return err
		}
		connection := s.connections[connectionIDs[i]]
		connection.mappings = append(connection.mappings, m)
	}

	return err
}
