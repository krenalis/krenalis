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
	"sync"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/postgres"
)

var (
	sheetsType    = reflect.TypeOf((*chichi.Sheets)(nil)).Elem()
	uiHandlerType = reflect.TypeOf((*chichi.UIHandler)(nil)).Elem()
)

// load loads the state.
func (state *State) load(connectorSettings map[string]*ConnectorSetting) error {

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
		connectors := chichi.Connectors()
		state.connectors = make(map[string]*Connector, len(connectors))
		for name, connector := range connectors {
			c := Connector{}
			var ct reflect.Type
			switch connector := connector.(type) {
			case chichi.AppInfo:
				c.Name = connector.Name
				c.Type = AppType
				c.Targets = ConnectorTargets(connector.Targets)
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = connector.TermForUsers
				c.TermForGroups = connector.TermForGroups
				switch connector.SendingMode {
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
				c.IdentityIDLabel = connector.IdentityIDLabel
				c.WebhooksPer = WebhooksPer(connector.WebhooksPer)
				if connector.OAuth.AuthURL != "" {
					c.OAuth = &OAuth{
						OAuth: connector.OAuth,
					}
				}
				c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
				c.Icon = connector.Icon
				if connectorSettings != nil {
					if setting, ok := connectorSettings[c.Name]; ok {
						c.OAuth.ClientID = setting.OAuthClientID
						c.OAuth.ClientSecret = setting.OAuthClientSecret
					}
				}
				ct = connector.ReflectType()
			case chichi.DatabaseInfo:
				c.Name = connector.Name
				c.Type = DatabaseType
				c.Targets = UsersFlag | GroupsFlag
				c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
				c.SampleQuery = connector.SampleQuery
				c.Icon = connector.Icon
				ct = connector.ReflectType()
			case chichi.FileInfo:
				c.Name = connector.Name
				c.Type = FileType
				c.FileExtension = connector.Extension
				c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
				c.Icon = connector.Icon
				ct = connector.ReflectType()
				c.HasSheets = ct.Implements(sheetsType)
			case chichi.FileStorageInfo:
				c.Name = connector.Name
				c.Type = FileStorageType
				c.Targets = UsersFlag | GroupsFlag
				c.Icon = connector.Icon
				ct = connector.ReflectType()
			case chichi.MobileInfo:
				c.Name = connector.Name
				c.Type = MobileType
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = "users"
				c.TermForGroups = "groups"
				c.Targets = EventsFlag | UsersFlag | GroupsFlag
				c.Icon = connector.Icon
				ct = connector.ReflectType()
			case chichi.ServerInfo:
				c.Name = connector.Name
				c.Type = ServerType
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = "users"
				c.TermForGroups = "groups"
				c.Targets = EventsFlag | UsersFlag | GroupsFlag
				c.Icon = connector.Icon
				ct = connector.ReflectType()
			case chichi.StreamInfo:
				c.Name = connector.Name
				c.Type = StreamType
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.Targets = EventsFlag
				c.Icon = connector.Icon
				ct = connector.ReflectType()
			case chichi.WebsiteInfo:
				c.Name = connector.Name
				c.Type = WebsiteType
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = "users"
				c.TermForGroups = "groups"
				c.Targets = EventsFlag | UsersFlag | GroupsFlag
				c.Icon = connector.Icon
				ct = connector.ReflectType()
			}
			c.HasUI = ct.Implements(uiHandlerType)
			state.connectors[name] = &c
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
		err = state.db.QueryScan(ctx, "SELECT id, organization, name, warehouse_type, warehouse_mode,"+
			" warehouse_settings, user_schema, identifiers, privacy_region, displayed_image, displayed_first_name,"+
			" displayed_last_name, displayed_information\n"+
			"FROM workspaces",
			func(rows *postgres.Rows) error {
				var organizationID int
				var warehouseType *WarehouseType
				var warehouseMode *WarehouseMode
				var displayedImage string
				var displayedFirstName string
				var displayedLastName string
				var displayedInformation string
				var userSchema []byte
				var warehouseSettings []byte
				for rows.Next() {
					ws := &Workspace{
						mu:          new(sync.Mutex),
						connections: map[int]*Connection{},
						accounts:    map[int]*Account{},
					}
					if err := rows.Scan(&ws.ID, &organizationID, &ws.Name, &warehouseType, &warehouseMode,
						&warehouseSettings, &userSchema, &ws.Identifiers, &ws.PrivacyRegion, &displayedImage,
						&displayedFirstName, &displayedLastName, &displayedInformation); err != nil {
						return err
					}
					ws.organization = state.organizations[organizationID]
					if warehouseType != nil {
						ws.Warehouse = &Warehouse{
							Type:     *warehouseType,
							Mode:     *warehouseMode,
							Settings: warehouseSettings,
						}
					}
					err = json.Unmarshal(userSchema, &ws.UserSchema)
					if err != nil {
						return err
					}
					ws.UserPrimarySources = map[string]int{}
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

		// Read all accounts.
		state.accounts = map[int]*Account{}
		err = state.db.QueryScan(ctx, "SELECT id, workspace, connector, code, access_token, refresh_token, expires_in FROM accounts",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					a := Account{}
					var workspaceID int
					var connectorName string
					if err := rows.Scan(&a.ID, &workspaceID, &connectorName, &a.Code, &a.AccessToken, &a.RefreshToken, &a.ExpiresIn); err != nil {
						return err
					}
					a.mu = new(sync.Mutex)
					a.workspace = state.workspaces[workspaceID]
					a.connector = state.connectors[connectorName]
					a.workspace.accounts[a.ID] = &a
					state.accounts[a.ID] = &a
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read all connections.
		state.connections = map[int]*Connection{}
		err = state.db.QueryScan(ctx, "SELECT id, workspace, name, role, enabled, connector,"+
			" account, strategy, sending_mode, website_host, event_connections,"+
			" settings, health FROM connections", func(rows *postgres.Rows) error {
			for rows.Next() {
				var workspaceID, account int
				var connector string
				c := Connection{}
				if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &c.Role, &c.Enabled, &connector,
					&account, &c.Strategy, &c.SendingMode, &c.WebsiteHost, &c.EventConnections, &c.Settings, &c.Health,
				); err != nil {
					return err
				}
				workspace := state.workspaces[workspaceID]
				c.mu = new(sync.Mutex)
				c.organization = workspace.organization
				c.workspace = workspace
				c.connector = state.connectors[connector]
				if c.connector == nil {
					return fmt.Errorf("the %s connector is required but not registered. (Possibly forgotten import?)", connector)
				}
				c.actions = map[int]*Action{}
				if account > 0 {
					c.account = state.accounts[account]
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
			"settings, table_name, identity_property, last_change_time_property, last_change_time_format,\n"+
			"user_cursor, health, file_ordering_property_path, export_mode, matching_properties_internal,\n"+
			"matching_properties_external, export_on_duplicated_users\n"+
			"FROM actions",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var eventType string
					var rawInSchema, rawOutSchema, filter, mapping []byte
					var function TransformationFunction
					var matchPropInternal, matchPropExternal []byte
					var connector *string
					action := Action{}
					err := rows.Scan(&action.ID, &connectionID, &action.Target, &eventType, &action.Name,
						&action.Enabled, &action.ScheduleStart, &action.SchedulePeriod, &rawInSchema, &rawOutSchema,
						&filter, &mapping, &function.Source, &function.Language, &function.Version, &action.Query,
						&connector, &action.Path, &action.Sheet, &action.Compression, &action.Settings,
						&action.TableName, &action.IdentityProperty, &action.LastChangeTimeProperty,
						&action.LastChangeTimeFormat, &action.UserCursor, &action.Health, &action.FileOrderingPropertyPath,
						&action.ExportMode, &matchPropInternal, &matchPropExternal, &action.ExportOnDuplicatedUsers)
					if err != nil {
						return err
					}
					c := state.connections[connectionID]
					if connector != nil {
						action.connector = state.connectors[*connector]
						if action.connector == nil {
							return fmt.Errorf("the %s connector is required but not registered. (Possibly forgotten import?)", *connector)
						}
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

		// Read all primary sources.
		err = state.db.QueryScan(ctx, "SELECT source, path FROM user_schema_primary_sources",
			func(rows *postgres.Rows) error {
				var source int
				var path string
				for rows.Next() {
					if err := rows.Scan(&source, &path); err != nil {
						return err
					}
					c := state.connections[source]
					c.workspace.UserPrimarySources[path] = source
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
