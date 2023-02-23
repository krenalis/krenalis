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
		ctx:              ctx,
		notifications:    db.ListenToNotifications(ctx),
		accounts:         map[int]*Account{},
		connectors:       map[int]*Connector{},
		workspaces:       map[int]*Workspace{},
		connections:      map[int]*Connection{},
		connectionsByKey: map[string]*Connection{},
		actions:          map[int]*Action{},
		resources:        map[int]*Resource{},
	}

	n := LoadStateNotification{ID: state.id}

	err = state.db.Transaction(ctx, func(tx *postgres.Tx) error {

		// Read the latest election.
		err := state.db.QueryRow(ctx, "SELECT number, leader FROM election LIMIT 1").
			Scan(&state.election.number, &state.election.leader)
		if err != nil {
			return err
		}

		// Read all connectors.
		state.connectors = map[int]*Connector{}
		err = state.db.QueryScan(ctx, "SELECT id, name, type, has_settings, logo_url, webhooks_per, oauth_url, oauth_client_id,"+
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
		err = state.db.QueryScan(ctx, "SELECT id, name, email, internal_ips FROM accounts", func(rows *postgres.Rows) error {
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
		err = state.db.QueryScan(ctx, "SELECT id, account, name, warehouse_type, warehouse_settings, schemas FROM workspaces",
			func(rows *postgres.Rows) error {
				var id, accountID int
				var name string
				var warehouseType *WarehouseType
				var warehouseSettings, schemas []byte
				for rows.Next() {
					if err := rows.Scan(&id, &accountID, &name, &warehouseType, &warehouseSettings, &schemas); err != nil {
						return err
					}
					account := state.accounts[accountID]
					workspace := &Workspace{
						mu:        new(sync.Mutex),
						ID:        id,
						account:   account,
						Name:      name,
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
					state.workspaces[id] = workspace
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read all resources.
		state.resources = map[int]*Resource{}
		err = state.db.QueryScan(ctx, "SELECT id, workspace, connector, code, access_token, refresh_token, expires_in FROM resources",
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
		err = state.db.QueryScan(ctx, "SELECT id, workspace, name, role, enabled, connector,"+
			" COALESCE(storage, 0), resource, website_host, user_cursor, identity_column, timestamp_column,"+
			" (transformation).in_types, (transformation).out_types, (transformation).python_source,"+
			" settings, schema, users_query, health FROM connections", func(rows *postgres.Rows) error {
			for rows.Next() {
				var workspaceID, connector, storage, resource int
				var transformIn, transformOut, transformSrc string
				var rawSchema string
				c := Connection{}
				if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &c.Role, &c.Enabled, &connector, &storage, &resource,
					&c.WebsiteHost, &c.UserCursor, &c.IdentityColumn, &c.TimestampColumn,
					&transformIn, &transformOut, &transformSrc, &c.Settings, &rawSchema, &c.UsersQuery, &c.Health); err != nil {
					return err
				}
				workspace := state.workspaces[workspaceID]
				c.mu = new(sync.Mutex)
				c.account = workspace.account
				c.workspace = workspace
				c.connector = state.connectors[connector]
				c.actions = map[int]*Action{}
				if storage > 0 {
					if st, ok := state.connections[storage]; ok {
						c.storage = st
					} else {
						c.storage = &Connection{}
						state.connections[storage] = c.storage
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
				// Load the connection's transformation, if present.
				if transformIn != "" {
					t := &Transformation{PythonSource: transformSrc}
					err := json.Unmarshal([]byte(transformIn), &t.In)
					if err != nil {
						return err
					}
					err = json.Unmarshal([]byte(transformOut), &t.Out)
					if err != nil {
						return err
					}
					c.transformation = t
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
		err = state.db.QueryScan(ctx, `SELECT connection, value FROM connections_keys ORDER BY connection, creation_time`,
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var value string
					if err := rows.Scan(&connectionID, &value); err != nil {
						return err
					}
					connection := state.connections[connectionID]
					connection.Keys = append(connection.Keys, value)
					state.connectionsByKey[value] = connection
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read the actions.
		err = state.db.QueryScan(ctx, "SELECT id, connection, action_type, name,\n"+
			"enabled, filter, mapping, (transformation).in_types, (transformation).out_types,\n"+
			"(transformation).python_source FROM actions", func(rows *postgres.Rows) error {
			for rows.Next() {
				var id, connectionID, actionType int
				var name string
				var enabled bool
				var filter, mapping, transformIn, transformOut, pythonSource []byte
				err := rows.Scan(&id, &connectionID, &actionType, &name, &enabled,
					&filter, &mapping, &transformIn, &transformOut, &pythonSource)
				if err != nil {
					return err
				}
				c := state.connections[connectionID]
				action := &Action{
					mu:         new(sync.Mutex),
					ID:         id,
					connection: c,
					ActionType: actionType,
					Name:       name,
					Enabled:    enabled,
				}
				err = json.Unmarshal(filter, &action.Filter)
				if err != nil {
					return err
				}
				if len(mapping) > 0 {
					err = json.Unmarshal(mapping, &action.Mapping)
					if err != nil {
						return err
					}
				}
				if len(transformIn) > 0 {
					t := &Transformation{PythonSource: string(pythonSource)}
					err := json.Unmarshal(transformIn, &t.In)
					if err != nil {
						return err
					}
					err = json.Unmarshal(transformOut, &t.Out)
					if err != nil {
						return err
					}
					action.Transformation = t
				}
				state.actions[id] = action
				c.actions[id] = action
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Read the mappings.
		err = state.db.QueryScan(ctx, "SELECT connection, in_properties, out_properties, predefined_func,\n"+
			"(custom_func).in_types, (custom_func).out_types, (custom_func).source\n"+
			"FROM connections_mappings", func(rows *postgres.Rows) error {
			for rows.Next() {
				m := &Mapping{}
				var connectionID int
				var inTypes, outTypes []byte // custom func only.
				var src string               // custom func only.
				err := rows.Scan(&connectionID, &m.InProperties, &m.OutProperties, &m.PredefinedFunc,
					&inTypes, &outTypes, &src)
				if err != nil {
					return err
				}
				if len(inTypes) > 0 { // custom func.
					m.CustomFunc = &MappingCustomFunc{}
					err := json.Unmarshal(inTypes, &m.CustomFunc.InTypes)
					if err != nil {
						return err
					}
					err = json.Unmarshal(outTypes, &m.CustomFunc.OutTypes)
					if err != nil {
						return err
					}
					m.CustomFunc.Source = src
				}
				connection := state.connections[connectionID]
				connection.mappings = append(connection.mappings, m)
			}
			return nil
		})
		if err != nil {
			return err
		}

		return tx.Notify(ctx, n)
	})
	if err != nil {
		return nil, err
	}

	return state, nil
}
