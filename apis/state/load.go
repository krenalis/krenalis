//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"chichi/apis/postgres"
	"chichi/connector"
	"chichi/connector/types"
)

var (
	uiType                  = reflect.TypeOf((*connector.UI)(nil)).Elem()
	appEventsConnectionType = reflect.TypeOf((*connector.AppEventsConnection)(nil)).Elem()
	appUsersConnectionType  = reflect.TypeOf((*connector.AppUsersConnection)(nil)).Elem()
	appGroupsConnectionType = reflect.TypeOf((*connector.AppGroupsConnection)(nil)).Elem()
	sheetsType              = reflect.TypeOf((*connector.Sheets)(nil)).Elem()
)

// Load loads the state.
func (state *State) Load() error {

	// Keep the state updated.
	go state.keepState()

	n := LoadState{ID: state.id}

	ctx := state.close.ctx

	err := state.db.Transaction(ctx, func(tx *postgres.Tx) error {

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
					app := connector.RegisteredApp(c.Name)
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
					c.Layouts.DateTime = app.DateTimeLayout
					c.Layouts.Date = app.DateLayout
					c.Layouts.Time = app.TimeLayout
				case DatabaseType:
					database := connector.RegisteredDatabase(c.Name)
					c.SourceDescription = database.SourceDescription
					c.DestinationDescription = database.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = UsersFlag | GroupsFlag
					c.Icon = database.Icon
					c.SampleQuery = database.SampleQuery
					ct = database.ConnectionReflectType()
				case FileType:
					file := connector.RegisteredFile(c.Name)
					c.SourceDescription = file.SourceDescription
					c.DestinationDescription = file.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = UsersFlag | GroupsFlag
					c.Icon = file.Icon
					c.FileExtension = file.Extension
					ct = file.ConnectionReflectType()
					c.HasSheets = ct.Implements(sheetsType)
				case MobileType:
					mobile := connector.RegisteredMobile(c.Name)
					c.SourceDescription = mobile.SourceDescription
					c.DestinationDescription = mobile.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = EventsFlag | UsersFlag | GroupsFlag
					c.Icon = mobile.Icon
					ct = mobile.ConnectionReflectType()
				case ServerType:
					server := connector.RegisteredServer(c.Name)
					c.SourceDescription = server.SourceDescription
					c.DestinationDescription = server.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = EventsFlag | UsersFlag | GroupsFlag
					c.Icon = server.Icon
					ct = server.ConnectionReflectType()
				case StorageType:
					storage := connector.RegisteredStorage(c.Name)
					c.SourceDescription = storage.SourceDescription
					c.DestinationDescription = storage.DestinationDescription
					c.Icon = storage.Icon
					ct = storage.ConnectionReflectType()
				case StreamType:
					stream := connector.RegisteredStream(c.Name)
					c.SourceDescription = stream.SourceDescription
					c.DestinationDescription = stream.DestinationDescription
					c.Targets = EventsFlag
					c.Icon = stream.Icon
					ct = stream.ConnectionReflectType()
				case WebsiteType:
					website := connector.RegisteredWebsite(c.Name)
					c.SourceDescription = website.SourceDescription
					c.DestinationDescription = website.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = EventsFlag | UsersFlag | GroupsFlag
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
		err = state.db.QueryScan(ctx, "SELECT id, account, name, warehouse_type, warehouse_settings,\n"+
			"identifiers, anonymous_identifiers_priority, anonymous_identifiers_mapping, privacy_region, schemas\n"+
			"FROM workspaces",
			func(rows *postgres.Rows) error {
				var accountID int
				var warehouseType *WarehouseType
				var warehouseSettings, mapping, schemas []byte
				for rows.Next() {
					ws := &Workspace{
						mu:          new(sync.Mutex),
						connections: map[int]*Connection{},
						resources:   map[int]*Resource{},
					}
					if err := rows.Scan(&ws.ID, &accountID, &ws.Name, &warehouseType, &warehouseSettings,
						&ws.Identifiers, &ws.AnonymousIdentifiers.Priority, &mapping, &ws.PrivacyRegion,
						&schemas); err != nil {
						return err
					}
					ws.account = state.accounts[accountID]
					if warehouseType != nil {
						ws.Warehouse = &Warehouse{
							Type:     *warehouseType,
							Settings: warehouseSettings,
						}
					}
					ws.Schemas = map[string]*types.Type{}
					if len(schemas) > 0 {
						err = json.Unmarshal(schemas, &ws.Schemas)
						if err != nil {
							return fmt.Errorf("cannot unmarshal schemas of workspace %d: %s", ws.ID, err)
						}
					}
					err = json.Unmarshal(mapping, &ws.AnonymousIdentifiers.Mapping)
					if err != nil {
						return err
					}
					ws.account.workspaces[ws.ID] = ws
					state.workspaces[ws.ID] = ws
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
			" COALESCE(storage, 0), compression::TEXT, resource, website_host,"+
			" settings, health FROM connections", func(rows *postgres.Rows) error {
			for rows.Next() {
				var workspaceID, connector, storage, resource int
				c := Connection{}
				if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &c.Role, &c.Enabled, &connector,
					&storage, &c.Compression, &resource, &c.WebsiteHost, &c.Settings, &c.Health,
				); err != nil {
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
		err = state.db.QueryScan(ctx, "SELECT id, connection, target, event_type, name, enabled, schedule_start,\n"+
			"schedule_period, in_schema, out_schema, filter, mapping, transformation_source, transformation_language,\n"+
			"transformation_version, query, path, table_name, sheet, identity_column, timestamp_column, timestamp_format,\n"+
			"(user_cursor).id, (user_cursor).timestamp, health, export_mode, matching_properties_internal,\n"+
			"matching_properties_external\nFROM actions",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var eventType string
					var rawInSchema, rawOutSchema, filter, mapping []byte
					var transformation Transformation
					var matchPropInternal, matchPropExternal []byte
					action := Action{}
					err := rows.Scan(&action.ID, &connectionID, &action.Target, &eventType, &action.Name,
						&action.Enabled, &action.ScheduleStart, &action.SchedulePeriod, &rawInSchema, &rawOutSchema,
						&filter, &mapping, &transformation.Source, &transformation.Language, &transformation.Version,
						&action.Query, &action.Path, &action.TableName, &action.Sheet, &action.IdentityColumn,
						&action.TimestampColumn, &action.TimestampFormat, &action.UserCursor.ID, &action.UserCursor.Timestamp,
						&action.Health, &action.ExportMode, &matchPropInternal, &matchPropExternal)
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
					err = action.InSchema.UnmarshalJSON(rawInSchema)
					if err != nil {
						// TODO(marco) disable the action instead of returning an error
						return err
					}
					err = action.OutSchema.UnmarshalJSON(rawOutSchema)
					if err != nil {
						// TODO(marco) disable the action instead of returning an error
						return err
					}
					if len(mapping) > 0 {
						err = json.Unmarshal(mapping, &action.Mapping)
						if err != nil {
							return err
						}
					}
					if transformation.Source != "" {
						action.Transformation = &transformation
					}
					if len(matchPropInternal) > 0 {
						action.MatchingProperties = &MatchingProperties{}
						err = json.Unmarshal(matchPropInternal, &action.MatchingProperties.Internal)
						if err != nil {
							return err
						}
						err = json.Unmarshal(matchPropExternal, &action.MatchingProperties.External)
						if err != nil {
							return err
						}
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

	return err
}
