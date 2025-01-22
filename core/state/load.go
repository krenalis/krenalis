//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/postgres"
)

var (
	sheetsType    = reflect.TypeOf((*meergo.Sheets)(nil)).Elem()
	uiHandlerType = reflect.TypeOf((*meergo.UIHandler)(nil)).Elem()
)

// load loads the state.
func (state *State) load(connectorsOAuth map[string]*ConnectorOAuth) error {

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
		connectors := meergo.Connectors()
		state.connectors = make(map[string]*Connector, len(connectors))
		for name, connector := range connectors {
			c := Connector{}
			var ct reflect.Type
			switch connector := connector.(type) {
			case meergo.AppInfo:
				ct = connector.ReflectType()
				c.Name = connector.Name
				c.Type = App
				c.Role = Role(connector.Role)
				c.Targets = ConnectorTargets(connector.Targets)
				if c.Targets.Contains(Groups) {
					// TODO(Gianluca): https://github.com/meergo/meergo/issues/895.
					return errors.New("target Groups is not supported by this installation of Meergo")
				}
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = connector.TermForUsers
				c.TermForGroups = connector.TermForGroups
				switch connector.SendingMode {
				case meergo.Cloud:
					mode := Cloud
					c.SendingMode = &mode
				case meergo.Device:
					mode := Device
					c.SendingMode = &mode
				case meergo.Combined:
					mode := Combined
					c.SendingMode = &mode
				}
				if ct.Implements(uiHandlerType) {
					c.HasSourceSettings = connector.HasSettings == meergo.Source || connector.HasSettings == meergo.Both
					c.HasDestinationSettings = connector.HasSettings == meergo.Destination || connector.HasSettings == meergo.Both
				}
				c.IdentityIDLabel = connector.IdentityIDLabel
				c.WebhooksPer = WebhooksPer(connector.WebhooksPer)
				if connector.OAuth.AuthURL != "" {
					c.OAuth = &OAuth{
						OAuth: connector.OAuth,
					}
				}
				c.BackoffPolicy = connector.BackoffPolicy
				c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
				c.Icon = connector.Icon
				if connectorsOAuth != nil {
					if oAuth, ok := connectorsOAuth[c.Name]; ok {
						c.OAuth.ClientID = oAuth.ClientID
						c.OAuth.ClientSecret = oAuth.ClientSecret
					}
				}
			case meergo.DatabaseInfo:
				ct = connector.ReflectType()
				c.Name = connector.Name
				c.Type = Database
				c.Role = Both
				c.Targets = UsersFlag
				c.HasSourceSettings = true
				c.HasDestinationSettings = true
				c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
				c.SampleQuery = connector.SampleQuery
				c.Icon = connector.Icon
			case meergo.FileInfo:
				ct = connector.ReflectType()
				c.Name = connector.Name
				c.Type = File
				c.Role = Both
				if ct.Implements(uiHandlerType) {
					c.HasSourceSettings = connector.HasSettings == meergo.Source || connector.HasSettings == meergo.Both
					c.HasDestinationSettings = connector.HasSettings == meergo.Destination || connector.HasSettings == meergo.Both
				}
				c.FileExtension = connector.Extension
				c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
				c.Icon = connector.Icon
				c.HasSheets = ct.Implements(sheetsType)
			case meergo.FileStorageInfo:
				c.Name = connector.Name
				c.Type = FileStorage
				c.Role = Both
				c.HasSourceSettings = true
				c.HasDestinationSettings = true
				c.Targets = UsersFlag
				c.Icon = connector.Icon
				ct = connector.ReflectType()
			case meergo.MobileInfo:
				ct = connector.ReflectType()
				c.Name = connector.Name
				c.Type = Mobile
				c.Role = Source
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = "users"
				c.TermForGroups = "groups"
				c.Targets = EventsFlag | UsersFlag
				c.Icon = connector.Icon
			case meergo.ServerInfo:
				ct = connector.ReflectType()
				c.Name = connector.Name
				c.Type = Server
				c.Role = Source
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = "users"
				c.TermForGroups = "groups"
				c.Targets = EventsFlag | UsersFlag
				c.Icon = connector.Icon
			case meergo.StreamInfo:
				ct = connector.ReflectType()
				c.Name = connector.Name
				c.Type = Stream
				c.Role = Both
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.Targets = EventsFlag
				c.HasSourceSettings = true
				c.HasDestinationSettings = true
				c.Icon = connector.Icon
			case meergo.WebsiteInfo:
				ct = connector.ReflectType()
				c.Name = connector.Name
				c.Type = Website
				c.Role = Source
				c.SourceDescription = connector.SourceDescription
				c.DestinationDescription = connector.DestinationDescription
				c.TermForUsers = "users"
				c.TermForGroups = "groups"
				c.Targets = EventsFlag | UsersFlag
				c.Icon = connector.Icon
			}
			state.connectors[name] = &c
		}

		// Read all warehouse types.
		drivers := meergo.WarehouseDrivers()
		state.warehouseTypes = make(map[string]WarehouseType, len(drivers))
		for _, driver := range drivers {
			state.warehouseTypes[driver.Name] = WarehouseType{
				Name: driver.Name,
				Icon: driver.Icon,
			}
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
			" warehouse_settings, user_schema, resolve_identities_on_batch_import,"+
			" identifiers, ui_user_profile_image, ui_user_profile_first_name,"+
			" ui_user_profile_last_name, ui_user_profile_extra, actions_to_purge "+
			"FROM workspaces",
			func(rows *postgres.Rows) error {
				var organizationID int
				var warehouseType string
				var warehouseMode WarehouseMode
				var userSchema []byte
				var warehouseSettings []byte
				for rows.Next() {
					ws := &Workspace{
						mu:          new(sync.Mutex),
						connections: map[int]*Connection{},
						executions:  map[int]*ActionExecution{},
						accounts:    map[int]*Account{},
					}
					if err := rows.Scan(&ws.ID, &organizationID, &ws.Name, &warehouseType,
						&warehouseMode, &warehouseSettings, &userSchema,
						&ws.ResolveIdentitiesOnBatchImport, &ws.Identifiers,
						&ws.UIPreferences.UserProfile.Image, &ws.UIPreferences.UserProfile.FirstName,
						&ws.UIPreferences.UserProfile.LastName, &ws.UIPreferences.UserProfile.Extra,
						&ws.actionsToPurge); err != nil {
						return err
					}
					ws.organization = state.organizations[organizationID]
					if _, ok := state.warehouseTypes[warehouseType]; !ok {
						return fmt.Errorf("warehouse driver for type %q is required but not registered. (Possibly forgotten import?)", warehouseType)
					}
					ws.Warehouse.Type = warehouseType
					ws.Warehouse.Mode = warehouseMode
					ws.Warehouse.Settings = warehouseSettings
					err = json.Unmarshal(userSchema, &ws.UserSchema)
					if err != nil {
						return err
					}
					ws.UserPrimarySources = map[string]int{}
					ws.organization.workspaces[ws.ID] = ws
					state.workspaces[ws.ID] = ws
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read all API keys.
		state.apiKeyByToken = map[string]*APIKey{}
		err = state.db.QueryScan(ctx, "SELECT id, organization, workspace, token FROM api_keys",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					k := APIKey{}
					var token string
					var workspace *int
					if err := rows.Scan(&k.ID, &k.Organization, &workspace, &token); err != nil {
						return err
					}
					if workspace != nil {
						k.Workspace = *workspace
					}
					state.apiKeyByToken[token] = &k
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
		err = state.db.QueryScan(ctx, "SELECT id, workspace, name, connector, role,"+
			" account, strategy, sending_mode, website_host, linked_connections,"+
			" settings, health FROM connections", func(rows *postgres.Rows) error {
			for rows.Next() {
				var workspaceID, account int
				var connector string
				c := Connection{}
				if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &connector, &c.Role,
					&account, &c.Strategy, &c.SendingMode, &c.WebsiteHost, &c.LinkedConnections, &c.Settings, &c.Health,
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
				if c.connector.Type == Server {
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

		// Read all event write keys.
		err = state.db.QueryScan(ctx, `SELECT connection, key FROM event_write_keys ORDER BY connection, creation_time`,
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var key string
					if err := rows.Scan(&connectionID, &key); err != nil {
						return err
					}
					connection := state.connections[connectionID]
					connection.Keys = append(connection.Keys, key)
					state.connectionsByKey[key] = connection
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read all actions.
		err = state.db.QueryScan(ctx, "SELECT id, connection, target, event_type, name, enabled, schedule_start,\n"+
			"schedule_period, in_schema, out_schema, filter, transformation_mapping, transformation_source,\n"+
			"transformation_language, transformation_version, transformation_preserve_json, transformation_in_paths,\n"+
			"transformation_out_paths, query, format, path, sheet, compression::TEXT, format_settings, export_mode,\n"+
			"matching_in, matching_out, allow_duplicates, table_name, table_key, identity_property,\n"+
			"last_change_time_property, last_change_time_format, health, file_ordering_property_path\n"+
			"FROM actions",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					var connectionID int
					var eventType string
					var rawInSchema, rawOutSchema, filter, mapping []byte
					var function TransformationFunction
					var format *string
					action := Action{}
					err := rows.Scan(&action.ID, &connectionID, &action.Target, &eventType, &action.Name,
						&action.Enabled, &action.ScheduleStart, &action.SchedulePeriod, &rawInSchema, &rawOutSchema,
						&filter, &mapping, &function.Source, &function.Language, &function.Version, &function.PreserveJSON,
						&action.Transformation.InPaths, &action.Transformation.OutPaths, &action.Query, &format,
						&action.Path, &action.Sheet, &action.Compression, &action.FormatSettings, &action.ExportMode,
						&action.Matching.In, &action.Matching.Out, &action.ExportOnDuplicates, &action.TableName,
						&action.TableKey, &action.IdentityProperty, &action.LastChangeTimeProperty,
						&action.LastChangeTimeFormat, &action.Health, &action.FileOrderingPropertyPath)
					if err != nil {
						return err
					}
					c := state.connections[connectionID]
					if format != nil {
						action.format = state.connectors[*format]
						if action.format == nil {
							return fmt.Errorf("the %s connector is required but not registered. (Possibly forgotten import?)", *format)
						}
					}
					action.mu = new(sync.Mutex)
					action.connection = c
					action.EventType = eventType
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
					if len(filter) > 0 {
						action.Filter, err = unmarshalWhere(filter, action.InSchema)
						if err != nil {
							return err
						}
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
					state.actions[action.ID] = &action
					c.actions[action.ID] = &action
				}
				return nil
			})
		if err != nil {
			return err
		}

		// Read running action executions.
		err = state.db.QueryScan(ctx, "SELECT id, action, cursor, reload, start_time\n"+
			"FROM actions_executions\nWHERE end_time IS NULL",
			func(rows *postgres.Rows) error {
				for rows.Next() {
					exe := ActionExecution{}
					var actionID int
					err := rows.Scan(&exe.ID, &actionID, &exe.Cursor, &exe.Reload, &exe.StartTime)
					if err != nil {
						return err
					}
					action := state.actions[actionID]
					exe.action = action
					action.execution = &exe
					ws := exe.action.connection.workspace
					ws.executions[exe.ID] = &exe
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
