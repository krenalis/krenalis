//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"chichi/apis/postgres"
	"chichi/apis/types"

	"github.com/google/uuid"
)

// Load loads the state and returns it.
func Load(ctx context.Context, db *postgres.DB) (*State, error) {

	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	state := &State{
		id:               id,
		db:               db,
		mu:               new(sync.Mutex),
		notifications:    db.ListenToNotifications(ctx),
		accounts:         map[int]*Account{},
		connectors:       map[int]*Connector{},
		workspaces:       map[int]*Workspace{},
		connections:      map[int]*Connection{},
		connectionsByKey: map[string]*Connection{},
		resources:        map[int]*Resource{},
	}

	n := LoadStateNotification{ID: state.id}

	err = state.db.Transaction(func(tx *postgres.Tx) error {

		// Read all connectors.
		state.connectors = map[int]*Connector{}
		err := state.db.QueryScan("SELECT id, name, type, has_settings, logo_url, webhooks_per, oauth_url, oauth_client_id,"+
			" oauth_client_secret, oauth_token_endpoint, oauth_default_token_type, oauth_default_expires_in,"+
			" oauth_forced_expires_in FROM connectors", func(rows *postgres.Rows) error {
			for rows.Next() {
				c := Connector{}
				oauth := ConnectorOAuth{}
				if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.HasSettings, &c.LogoURL, &c.WebhooksPer, &oauth.URL,
					&oauth.ClientID, &oauth.ClientSecret, &oauth.TokenEndpoint, &oauth.DefaultTokenType,
					&oauth.DefaultExpiresIn, &oauth.ForcedExpiresIn); err != nil {
					return err
				}
				if oauth.URL != "" {
					c.OAuth = &oauth
				}
				state.connectors[c.ID] = &c
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Read all accounts.
		state.accounts = map[int]*Account{}
		err = state.db.QueryScan("SELECT id, name, email, internal_ips FROM accounts", func(rows *postgres.Rows) error {
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
				state.accounts[id] = account
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Read all workspaces.
		state.workspaces = map[int]*Workspace{}
		err = state.db.QueryScan("SELECT id, account, warehouse_type, warehouse_settings, schemas FROM workspaces",
			func(rows *postgres.Rows) error {
				var id, accountID int
				var warehouseType *WarehouseType
				var warehouseSettings, schemas []byte
				for rows.Next() {
					if err := rows.Scan(&id, &accountID, &warehouseType, &warehouseSettings, &schemas); err != nil {
						return err
					}
					account := state.accounts[accountID]
					workspace := &Workspace{
						mu:        new(sync.Mutex),
						ID:        id,
						account:   account,
						resources: map[int]*Resource{},
					}
					if warehouseType != nil {
						workspace.Warehouse, err = openWarehouse(*warehouseType, warehouseSettings)
						if err != nil {
							log.Fatalf("cannot open data warehouse of workspace %d: %state", id, err)
						}
						workspace.Schemas = map[string]*types.Type{}
						if len(schemas) > 0 {
							err = json.Unmarshal(schemas, &workspace.Schemas)
							if err != nil {
								log.Fatalf("cannot unmarshal schemas of workspace %d: %state", id, err)
							}
						}
					}
					workspace.connections = map[int]*Connection{}
					//workspace.EventListeners = &EventListeners{workspace}  // TODO(marco)
					account.workspaces[id] = workspace
					state.workspaces[id] = workspace
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read all resources.
		state.resources = map[int]*Resource{}
		err = state.db.QueryScan("SELECT id, workspace, connector, code, access_token, refresh_token, expires_in FROM resources",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					r := Resource{}
					var workspaceID, connectorID int
					if err := rows.Scan(&r.ID, &workspaceID, &connectorID, &r.Code, &r.AccessToken, &r.RefreshToken, &r.ExpiresIn); err != nil {
						return err
					}
					r.mu = new(sync.Mutex)
					r.workspace = state.workspaces[workspaceID]
					r.connector = state.connectors[connectorID]
					r.workspace.resources[r.ID] = &r
					state.resources[r.ID] = &r
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read all connections.
		state.connections = map[int]*Connection{}
		err = state.db.QueryScan("SELECT id, workspace, name, role, enabled, connector, COALESCE(storage, 0),"+
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
				workspace := state.workspaces[workspaceID]
				c.mu = new(sync.Mutex)
				c.account = workspace.account
				c.workspace = workspace
				c.connector = state.connectors[connector]
				if storage > 0 {
					if st, ok := state.connections[storage]; ok {
						c.storage = st
					} else {
						c.storage = &Connection{}
						state.connections[storage] = c.storage
					}
				}
				if stream > 0 {
					if st, ok := state.connections[stream]; ok {
						c.stream = st
					} else {
						c.stream = &Connection{}
						state.connections[stream] = c.stream
					}
				}
				if resource > 0 {
					c.resource = state.resources[resource]
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
				connection, ok := state.connections[c.ID]
				if ok {
					*connection = c
				} else {
					connection = &Connection{}
					*connection = c
				}
				workspace.connections[c.ID] = connection
				state.connections[c.ID] = connection
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Read all keys.
		err = state.db.QueryScan(`SELECT connection, value FROM connections_keys ORDER BY connection, creation_time`,
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var value string
					if err := rows.Scan(&connectionID, &value); err != nil {
						return err
					}
					connection := state.connections[connectionID]
					connection.Keys = append(connection.Keys, value)
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read the mappings.
		var mappings []*Mapping
		inSchemas := [][]byte{}
		outSchemas := [][]byte{}
		connectionIDs := []int{}
		err = state.db.QueryScan("SELECT id, connection, \"in\", predefined_func, source_code, out FROM connections_mappings", func(rows *postgres.Rows) error {
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
			connection := state.connections[connectionIDs[i]]
			connection.mappings = append(connection.mappings, m)
		}
		if err != nil {
			return err
		}

		return tx.Notify(n)
	})
	if err != nil {
		return nil, err
	}

	return state, nil
}
