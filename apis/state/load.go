//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"encoding/json"
	"reflect"
	"sync"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/postgres"
)

var (
	sheetsType    = reflect.TypeOf((*chichi.Sheets)(nil)).Elem()
	uiHandlerType = reflect.TypeOf((*chichi.UIHandler)(nil)).Elem()
)

// load loads the state.
func (state *State) load() error {

	n := LoadState{ID: state.id}

	ctx := state.close.ctx

	err := state.Transaction(ctx, func(tx *Tx) error {

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
					app := chichi.RegisteredApp(c.Name)
					c.Targets = ConnectorTargets(app.Targets)
					c.SourceDescription = app.SourceDescription
					c.DestinationDescription = app.DestinationDescription
					c.TermForUsers = app.TermForUsers
					c.TermForGroups = app.TermForGroups
					switch app.SendingMode {
					case chichi.Cloud:
						mode := Cloud
						c.SendingMode = &mode
					case chichi.Device:
						mode := Device
						c.SendingMode = &mode
					case chichi.Combined:
						mode := Combined
						c.SendingMode = &mode
					}
					c.IdentityIDLabel = app.IdentityIDLabel
					c.SuggestedDisplayedProperty = app.SuggestedDisplayedProperty
					c.WebhooksPer = WebhooksPer(app.WebhooksPer)
					if app.OAuth.AuthURL != "" {
						c.OAuth = &OAuth{
							OAuth:        app.OAuth,
							ClientSecret: oauthClientSecret,
							ClientID:     oauthClientID,
						}
					}
					c.TimeLayouts = TimeLayouts(app.TimeLayouts)
					c.Icon = app.Icon
					ct = app.ReflectType()
				case DatabaseType:
					database := chichi.RegisteredDatabase(c.Name)
					c.Targets = UsersFlag | GroupsFlag
					c.TimeLayouts = TimeLayouts(database.TimeLayouts)
					c.SampleQuery = database.SampleQuery
					c.Icon = database.Icon
					ct = database.ReflectType()
				case FileType:
					file := chichi.RegisteredFile(c.Name)
					c.FileExtension = file.Extension
					c.TimeLayouts = TimeLayouts(file.TimeLayouts)
					c.Icon = file.Icon
					ct = file.ReflectType()
					c.HasSheets = ct.Implements(sheetsType)
				case FileStorageType:
					storage := chichi.RegisteredFileStorage(c.Name)
					c.Targets = UsersFlag | GroupsFlag
					c.Icon = storage.Icon
					ct = storage.ReflectType()
				case MobileType:
					mobile := chichi.RegisteredMobile(c.Name)
					c.SourceDescription = mobile.SourceDescription
					c.DestinationDescription = mobile.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = EventsFlag | UsersFlag | GroupsFlag
					c.Icon = mobile.Icon
					ct = mobile.ReflectType()
				case ServerType:
					server := chichi.RegisteredServer(c.Name)
					c.SourceDescription = server.SourceDescription
					c.DestinationDescription = server.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = EventsFlag | UsersFlag | GroupsFlag
					c.Icon = server.Icon
					ct = server.ReflectType()
				case StreamType:
					stream := chichi.RegisteredStream(c.Name)
					c.SourceDescription = stream.SourceDescription
					c.DestinationDescription = stream.DestinationDescription
					c.Targets = EventsFlag
					c.Icon = stream.Icon
					ct = stream.ReflectType()
				case WebsiteType:
					website := chichi.RegisteredWebsite(c.Name)
					c.SourceDescription = website.SourceDescription
					c.DestinationDescription = website.DestinationDescription
					c.TermForUsers = "users"
					c.TermForGroups = "groups"
					c.Targets = EventsFlag | UsersFlag | GroupsFlag
					c.Icon = website.Icon
					ct = website.ReflectType()
				}
				c.HasUI = ct.Implements(uiHandlerType)
				state.connectors[c.ID] = &c
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Read all organizations.
		state.organizations = map[int]*Organization{}
		err = state.db.QueryScan(ctx, "SELECT id, name FROM organizations", func(rows *postgres.Rows) error {
			var id int
			var name string
			for rows.Next() {
				if err := rows.Scan(&id, &name); err != nil {
					return err
				}
				organization := &Organization{
					mu:   new(sync.Mutex),
					ID:   id,
					Name: name,
				}
				organization.workspaces = map[int]*Workspace{}
				state.organizations[id] = organization
			}
			return nil
		})
		if err != nil {
			return err
		}

		// Read all workspaces.
		state.workspaces = map[int]*Workspace{}
		err = state.db.QueryScan(ctx, "SELECT id, organization, name, warehouse_type, warehouse_settings, users_schema,\n"+
			"identifiers, privacy_region, displayed_image, displayed_first_name, displayed_last_name, displayed_information\n"+
			"FROM workspaces",
			func(rows *postgres.Rows) error {
				var organizationID int
				var warehouseType *WarehouseType
				var displayedImage string
				var displayedFirstName string
				var displayedLastName string
				var displayedInformation string
				var usersSchema []byte
				var warehouseSettings []byte
				for rows.Next() {
					ws := &Workspace{
						mu:          new(sync.Mutex),
						connections: map[int]*Connection{},
						resources:   map[int]*Resource{},
					}
					if err := rows.Scan(&ws.ID, &organizationID, &ws.Name, &warehouseType, &warehouseSettings,
						&usersSchema, &ws.Identifiers, &ws.PrivacyRegion, &displayedImage,
						&displayedFirstName, &displayedLastName, &displayedInformation); err != nil {
						return err
					}
					ws.organization = state.organizations[organizationID]
					if warehouseType != nil {
						ws.Warehouse = &Warehouse{
							Type:     *warehouseType,
							Settings: warehouseSettings,
						}
					}
					err = json.Unmarshal(usersSchema, &ws.UsersSchema)
					if err != nil {
						return err
					}
					ws.DisplayedProperties = DisplayedProperties{
						Image:       displayedImage,
						FirstName:   displayedFirstName,
						LastName:    displayedLastName,
						Information: displayedInformation,
					}
					ws.organization.workspaces[ws.ID] = ws
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
			" resource, strategy, sending_mode, website_host, event_connections,"+
			" settings, health FROM connections", func(rows *postgres.Rows) error {
			for rows.Next() {
				var workspaceID, connector, resource int
				c := Connection{}
				if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &c.Role, &c.Enabled, &connector,
					&resource, &c.Strategy, &c.SendingMode, &c.WebsiteHost, &c.EventConnections, &c.Settings, &c.Health,
				); err != nil {
					return err
				}
				workspace := state.workspaces[workspaceID]
				c.mu = new(sync.Mutex)
				c.organization = workspace.organization
				c.workspace = workspace
				c.connector = state.connectors[connector]
				c.actions = map[int]*Action{}
				if resource > 0 {
					c.resource = state.resources[resource]
				}
				if c.SendingMode == nil && c.Role == Destination && c.connector.SendingMode != nil {
					mode := Cloud
					if sm := *c.connector.SendingMode; sm == Device {
						mode = Device
					}
					c.SendingMode = &mode
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
			"schedule_period, in_schema, out_schema, filter, transformation_mapping, transformation_source,\n"+
			"transformation_language, transformation_version, query, connector, path, sheet, compression::TEXT,\n"+
			"settings, table_name, identity_property, displayed_property, last_change_time_property, last_change_time_format,\n"+
			"(user_cursor).id, (user_cursor).last_change_time, health, export_mode,\n"+
			"matching_properties_internal, matching_properties_external, export_on_duplicated_users\n"+
			"FROM actions",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var eventType string
					var rawInSchema, rawOutSchema, filter, mapping []byte
					var function TransformationFunction
					var matchPropInternal, matchPropExternal []byte
					var connector *int
					action := Action{}
					err := rows.Scan(&action.ID, &connectionID, &action.Target, &eventType, &action.Name,
						&action.Enabled, &action.ScheduleStart, &action.SchedulePeriod, &rawInSchema, &rawOutSchema,
						&filter, &mapping, &function.Source, &function.Language, &function.Version, &action.Query,
						&connector, &action.Path, &action.Sheet, &action.Compression, &action.Settings,
						&action.TableName, &action.IdentityProperty, &action.DisplayedProperty, &action.LastChangeTimeProperty,
						&action.LastChangeTimeFormat, &action.UserCursor.ID, &action.UserCursor.LastChangeTime, &action.Health,
						&action.ExportMode, &matchPropInternal, &matchPropExternal, &action.ExportOnDuplicatedUsers)
					if err != nil {
						return err
					}
					c := state.connections[connectionID]
					if connector != nil {
						action.connector = state.connectors[*connector]
					}
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
						err = json.Unmarshal(mapping, &action.Transformation.Mapping)
						if err != nil {
							return err
						}
					}
					if function.Source != "" {
						action.Transformation.Function = &function
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
