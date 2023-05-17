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
	"reflect"
	"strings"
	"sync"

	"chichi/apis/postgres"
	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/google/uuid"
)

var (
	uiType                  = reflect.TypeOf((*_connector.UI)(nil)).Elem()
	appEventsConnectionType = reflect.TypeOf((*_connector.AppEventsConnection)(nil)).Elem()
	appUsersConnectionType  = reflect.TypeOf((*_connector.AppUsersConnection)(nil)).Elem()
	appGroupsConnectionType = reflect.TypeOf((*_connector.AppGroupsConnection)(nil)).Elem()
	sheetsType              = reflect.TypeOf((*_connector.Sheets)(nil)).Elem()
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
		err = state.db.QueryScan(ctx, "SELECT id, name, type, oauth_client_id, oauth_client_secret FROM connectors", func(rows *postgres.Rows) error {
			for rows.Next() {
				c := Connector{}
				var oauthClientID, oauthClientSecret string
				if err := rows.Scan(&c.ID, &c.Name, &c.Type, &oauthClientID, &oauthClientSecret); err != nil {
					return err
				}
				var ct reflect.Type
				switch c.Type {
				case AppType:
					app := _connector.RegisteredApp(c.Name)
					c.SourceDescription = app.SourceDescription
					c.DestinationDescription = app.DestinationDescription
					c.TermForUsers = app.TermForUsers
					c.TermForGroups = app.TermForGroups
					ct = app.ConnectionReflectType()
					if ct.Implements(appEventsConnectionType) {
						c.Targets |= EventsFlag
					}
					if ct.Implements(appUsersConnectionType) {
						c.Targets |= UsersFlag
					}
					if ct.Implements(appGroupsConnectionType) {
						c.Targets |= GroupsFlag
					}
					c.Icon = app.Icon
					c.WebhooksPer = WebhooksPer(app.WebhooksPer)
					if app.OAuth.AuthURL != "" {
						c.OAuth = &OAuth{
							OAuth:        app.OAuth,
							ClientSecret: oauthClientSecret,
							ClientID:     oauthClientID,
						}
					}
				case DatabaseType:
					database := _connector.RegisteredDatabase(c.Name)
					c.SourceDescription = database.SourceDescription
					c.DestinationDescription = database.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = UsersFlag | GroupsFlag
					c.Icon = database.Icon
					ct = database.ConnectionReflectType()
				case FileType:
					file := _connector.RegisteredFile(c.Name)
					c.SourceDescription = file.SourceDescription
					c.DestinationDescription = file.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = UsersFlag | GroupsFlag
					c.Icon = file.Icon
					ct = file.ConnectionReflectType()
					c.HasSheets = ct.Implements(sheetsType)
				case MobileType:
					mobile := _connector.RegisteredMobile(c.Name)
					c.SourceDescription = mobile.SourceDescription
					c.DestinationDescription = mobile.DestinationDescription
					c.Targets = EventsFlag
					c.Icon = mobile.Icon
					ct = mobile.ConnectionReflectType()
				case ServerType:
					server := _connector.RegisteredServer(c.Name)
					c.SourceDescription = server.SourceDescription
					c.DestinationDescription = server.DestinationDescription
					c.Targets = EventsFlag
					c.Icon = server.Icon
					ct = server.ConnectionReflectType()
				case StorageType:
					storage := _connector.RegisteredStorage(c.Name)
					c.SourceDescription = storage.SourceDescription
					c.DestinationDescription = storage.DestinationDescription
					c.Icon = storage.Icon
					ct = storage.ConnectionReflectType()
				case StreamType:
					stream := _connector.RegisteredStream(c.Name)
					c.SourceDescription = stream.SourceDescription
					c.DestinationDescription = stream.DestinationDescription
					c.Targets = EventsFlag
					c.Icon = stream.Icon
					ct = stream.ConnectionReflectType()
				case WebsiteType:
					website := _connector.RegisteredWebsite(c.Name)
					c.SourceDescription = website.SourceDescription
					c.DestinationDescription = website.DestinationDescription
					c.Targets = EventsFlag
					c.Icon = website.Icon
					ct = website.ConnectionReflectType()
				}
				c.HasSettings = ct.Implements(uiType)
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
		err = state.db.QueryScan(ctx, "SELECT id, account, name, warehouse_type, warehouse_settings, privacy_region, schemas FROM workspaces",
			func(rows *postgres.Rows) error {
				var id, accountID int
				var name string
				var privacyRegion PrivacyRegion
				var warehouseType *WarehouseType
				var warehouseSettings, schemas []byte
				for rows.Next() {
					if err := rows.Scan(&id, &accountID, &name, &warehouseType, &warehouseSettings, &privacyRegion, &schemas); err != nil {
						return err
					}
					account := state.accounts[accountID]
					workspace := &Workspace{
						mu:            new(sync.Mutex),
						ID:            id,
						account:       account,
						Name:          name,
						resources:     map[int]*Resource{},
						PrivacyRegion: privacyRegion,
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
			" COALESCE(storage, 0), resource, website_host, identity_column, timestamp_column,"+
			" settings, health FROM connections", func(rows *postgres.Rows) error {
			for rows.Next() {
				var workspaceID, connector, storage, resource int
				c := Connection{}
				if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &c.Role, &c.Enabled, &connector, &storage, &resource,
					&c.WebsiteHost, &c.IdentityColumn, &c.TimestampColumn,
					&c.Settings, &c.Health); err != nil {
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

		// Read all actions.
		err = state.db.QueryScan(ctx, "SELECT id, connection, target, event_type, name,\n"+
			"enabled, schedule_start, schedule_period, filter, schema, mapping,\n"+
			"(transformation).in_types, (transformation).out_types,\n"+
			"(transformation).python_source, query, path, sheet, user_cursor, health FROM actions",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var eventType string
					var filter, rawSchema, mapping, transformIn, transformOut, pythonSource []byte
					action := Action{}
					err := rows.Scan(&action.ID, &connectionID, &action.Target, &eventType, &action.Name,
						&action.Enabled, &action.ScheduleStart, &action.SchedulePeriod, &filter,
						&rawSchema, &mapping, &transformIn, &transformOut, &pythonSource, &action.Query,
						&action.Path, &action.Sheet, &action.UserCursor, &action.Health)
					if err != nil {
						return err
					}
					c := state.connections[connectionID]
					action.mu = new(sync.Mutex)
					action.connection = c
					action.EventType = eventType
					if len(filter) > 0 {
						err = json.Unmarshal(filter, &action.Filter)
						if err != nil {
							return err
						}
					}
					if len(rawSchema) > 0 {
						err := action.Schema.UnmarshalJSON(rawSchema)
						if err != nil {
							// TODO(marco) disable the action instead of returning an error
							return err
						}
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
					state.actions[action.ID] = &action
					c.actions[action.ID] = &action
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read the non-terminated action executions.
		err = state.db.QueryScan(ctx, "SELECT id, action, storage, reimport, start_time\n"+
			"FROM actions_executions\nWHERE end_time IS NULL",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					exe := ActionExecution{}
					var actionID int
					var storage *int
					err := rows.Scan(&exe.ID, &actionID, &storage, &exe.Reimport, &exe.StartTime)
					if err != nil {
						return err
					}
					exe.action = state.actions[actionID]
					if storage != nil {
						exe.storage = state.connections[*storage]
					}
					exe.action.execution = &exe
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
