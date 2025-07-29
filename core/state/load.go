//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package state

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/db"
	_json "github.com/meergo/meergo/json"
)

// load loads the state.
func (state *State) load(connectorsOAuth map[string]*ConnectorOAuth) error {

	// Read all connectors.
	connectors := meergo.Connectors()
	state.connectors = make(map[string]*Connector, len(connectors))
	for name, connector := range connectors {
		c := Connector{}
		switch connector := connector.(type) {
		case meergo.AppInfo:
			c.Name = connector.Name
			c.Type = App
			c.Categories = connector.Categories
			if asSource := connector.AsSource; asSource != nil {
				c.SourceTargets = ConnectorTargets(asSource.Targets)
				c.HasSourceSettings = asSource.HasSettings
				c.Documentation.Source.Summary = asSource.Documentation.Summary
				c.Documentation.Source.Overview = asSource.Documentation.Overview
			}
			if asDest := connector.AsDestination; asDest != nil {
				c.DestinationTargets = ConnectorTargets(asDest.Targets)
				c.HasDestinationSettings = asDest.HasSettings
				c.Documentation.Destination.Summary = asDest.Documentation.Summary
				c.Documentation.Destination.Overview = asDest.Documentation.Overview
			}
			c.Terms = ConnectorTerms(connector.Terms)
			switch connector.AsDestination.SendingMode {
			case meergo.Client:
				mode := Client
				c.SendingMode = &mode
			case meergo.Server:
				mode := Server
				c.SendingMode = &mode
			case meergo.ClientAndServer:
				mode := ClientAndServer
				c.SendingMode = &mode
			}
			c.IdentityIDLabel = connector.IdentityIDLabel
			// c.WebhooksPer = WebhooksPer(connector.WebhooksPer) TODO(marco): implement webhooks
			if connector.OAuth.AuthURL != "" {
				c.OAuth = &OAuth{
					OAuth: connector.OAuth,
				}
			}
			c.EndpointGroups = connector.EndpointGroups
			c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
			c.Icon = connector.Icon
			if connectorsOAuth != nil {
				if oAuth, ok := connectorsOAuth[c.Name]; ok {
					c.OAuth.ClientID = oAuth.ClientID
					c.OAuth.ClientSecret = oAuth.ClientSecret
				}
			}
		case meergo.DatabaseInfo:
			c.Name = connector.Name
			c.Type = Database
			c.Categories = connector.Categories
			// It is assumed that each Database connector supports both read
			// and write operations.
			c.SourceTargets = UsersFlag
			c.DestinationTargets = UsersFlag
			// It is assumed that each Database connector always have both
			// source and destination settings.
			c.HasSourceSettings = true
			c.HasDestinationSettings = true
			c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
			c.SampleQuery = connector.SampleQuery
			c.Icon = connector.Icon
			c.Documentation = connector.Documentation
			if summary := c.Documentation.Source.Summary; summary == "" {
				c.Documentation.Source.Summary = "Import users from " + article(c.Name) + " " + c.Name + " database"
			}
			if summary := c.Documentation.Destination.Summary; summary == "" {
				c.Documentation.Destination.Summary = "Exports users to " + article(c.Name) + " " + c.Name + " database"
			}
		case meergo.FileInfo:
			c.Name = connector.Name
			c.Type = File
			c.Categories = connector.Categories
			if asSource := connector.AsSource; asSource != nil {
				c.SourceTargets = UsersFlag
				c.HasSourceSettings = asSource.HasSettings
				c.Documentation.Source = asSource.Documentation
				if c.Documentation.Source.Summary == "" {
					c.Documentation.Source.Summary = "Import users from " + article(c.Name) + " " + c.Name + " file"
				}
			}
			if asDest := connector.AsDestination; asDest != nil {
				c.DestinationTargets = UsersFlag
				c.HasDestinationSettings = connector.AsDestination.HasSettings
				c.Documentation.Destination = asDest.Documentation
				if c.Documentation.Destination.Summary == "" {
					c.Documentation.Destination.Summary = "Export users to " + article(c.Name) + " " + c.Name + " file"
				}
			}
			c.FileExtension = connector.Extension
			c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
			c.Icon = connector.Icon
			c.HasSheets = connector.HasSheets
		case meergo.FileStorageInfo:
			c.Name = connector.Name
			c.Type = FileStorage
			c.Categories = connector.Categories
			if asSource := connector.AsSource; asSource != nil {
				c.SourceTargets = UsersFlag
				// It is assumed that, if a FileStorage connector can be
				// used as a source, it always has source settings.
				c.HasSourceSettings = true
				c.Documentation.Source = asSource.Documentation
				if c.Documentation.Source.Summary == "" {
					c.Documentation.Source.Summary = "Import users from a file on " + c.Name
				}
			}
			if asDest := connector.AsDestination; asDest != nil {
				c.DestinationTargets = UsersFlag
				// It is assumed that, if a FileStorage connector can be
				// used as a destination, it always has destination
				// settings.
				c.HasDestinationSettings = true
				c.Documentation.Destination = asDest.Documentation
				if c.Documentation.Source.Summary == "" {
					c.Documentation.Source.Summary = "Exports users to a file on " + c.Name
				}
			}
			c.Icon = connector.Icon
		case meergo.SDKInfo:
			c.Name = connector.Name
			c.Type = SDK
			c.Categories = connector.Categories
			c.Terms = ConnectorTerms{
				User:  "user",
				Users: "users",
				// Group:  "group", TODO(marco): Implement groups
				// Groups: "groups",
			}
			c.SourceTargets = EventsFlag | UsersFlag
			c.Icon = connector.Icon
			c.Strategies = connector.Strategies
			c.Documentation = connector.Documentation
		case meergo.StreamInfo:
			c.Name = connector.Name
			c.Type = Stream
			c.Categories = connector.Categories
			c.SourceTargets = EventsFlag
			// It is assumed that a stream connector always have settings.
			c.HasSourceSettings = true
			c.HasDestinationSettings = true
			c.Icon = connector.Icon
			c.Documentation = connector.Documentation
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

	ctx := state.close.ctx

	tx, err := state.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		// Calling Rollback is always safe.
		_ = tx.Rollback(ctx)
	}()

	// Read the latest election.
	err = tx.QueryRow(ctx, "SELECT number, leader FROM election LIMIT 1").
		Scan(&state.election.number, &state.election.leader)
	if err != nil {
		return err
	}

	// Read all organizations.
	state.organizations = map[int]*Organization{}
	err = tx.QueryScan(ctx, "SELECT id, name FROM organizations", func(rows *db.Rows) error {
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
	err = tx.QueryScan(ctx, "SELECT id, organization, name, warehouse_type,"+
		" warehouse_mode, warehouse_settings, warehouse_mcp_settings, alter_user_schema_id,"+
		" alter_user_schema_schema, alter_user_schema_primary_sources, alter_user_schema_operations,"+
		" alter_user_schema_start_time, alter_user_schema_end_time,"+
		" alter_user_schema_error, user_schema, resolve_identities_on_batch_import,"+
		" identifiers, ir_id, ir_start_time, ir_end_time, ui_user_profile_image,"+
		" ui_user_profile_first_name, ui_user_profile_last_name, ui_user_profile_extra, actions_to_purge "+
		"FROM workspaces",
		func(rows *db.Rows) error {
			var organizationID int
			var warehouseType string
			var warehouseMode WarehouseMode
			var userSchema []byte
			var alterUserSchemaSchema []byte
			var warehouseSettings, warehouseMCPSettings []byte
			for rows.Next() {
				ws := &Workspace{
					mu:          new(sync.Mutex),
					connections: map[int]*Connection{},
					executions:  map[int]*ActionExecution{},
					accounts:    map[int]*Account{},
				}
				if err := rows.Scan(&ws.ID, &organizationID, &ws.Name, &warehouseType,
					&warehouseMode, &warehouseSettings, &warehouseMCPSettings, &ws.AlterUserSchema.ID,
					&alterUserSchemaSchema, &ws.AlterUserSchema.PrimarySources,
					&ws.AlterUserSchema.Operations, &ws.AlterUserSchema.StartTime,
					&ws.AlterUserSchema.EndTime, &ws.AlterUserSchema.Err, &userSchema,
					&ws.ResolveIdentitiesOnBatchImport, &ws.Identifiers, &ws.IR.ID,
					&ws.IR.StartTime, &ws.IR.EndTime, &ws.UIPreferences.UserProfile.Image,
					&ws.UIPreferences.UserProfile.FirstName, &ws.UIPreferences.UserProfile.LastName,
					&ws.UIPreferences.UserProfile.Extra, &ws.actionsToPurge); err != nil {
					return err
				}
				ws.organization = state.organizations[organizationID]
				if _, ok := state.warehouseTypes[warehouseType]; !ok {
					return fmt.Errorf("warehouse driver for type %q is required but not registered. (Possibly forgotten import?)", warehouseType)
				}
				ws.Warehouse.Type = warehouseType
				ws.Warehouse.Mode = warehouseMode
				ws.Warehouse.Settings = warehouseSettings
				if _json.Value(warehouseMCPSettings).IsNull() {
					ws.Warehouse.MCPSettings = nil
				} else {
					ws.Warehouse.MCPSettings = warehouseMCPSettings
				}
				err = json.Unmarshal(userSchema, &ws.UserSchema)
				if err != nil {
					return err
				}
				err = json.Unmarshal(alterUserSchemaSchema, &ws.AlterUserSchema.Schema)
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
	err = tx.QueryScan(ctx, "SELECT id, organization, workspace, token FROM api_keys",
		func(rows *db.Rows) error {
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
	err = tx.QueryScan(ctx, "SELECT id, workspace, connector, code, access_token, refresh_token, expires_in FROM accounts",
		func(rows *db.Rows) error {
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
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all connections.
	state.connections = map[int]*Connection{}
	err = tx.QueryScan(ctx, "SELECT id, workspace, name, connector, role,"+
		" account, strategy, sending_mode, linked_connections, settings,"+
		" health FROM connections", func(rows *db.Rows) error {
		for rows.Next() {
			var workspaceID, account int
			var connector string
			c := Connection{}
			if err := rows.Scan(&c.ID, &workspaceID, &c.Name, &connector, &c.Role,
				&account, &c.Strategy, &c.SendingMode, &c.LinkedConnections, &c.Settings, &c.Health,
			); err != nil {
				return err
			}
			ws := state.workspaces[workspaceID]
			c.mu = new(sync.Mutex)
			c.organization = ws.organization
			c.workspace = ws
			c.connector = state.connectors[connector]
			if c.connector == nil {
				return fmt.Errorf("the %s connector required by the %s '%s' is not included in the executable. "+
					"Recompile the executable with the %s connector to resolve the issue", connector, c.Role, c.Name, connector)
			}
			c.actions = map[int]*Action{}
			if account > 0 {
				c.account = ws.accounts[account]
			}
			if c.SendingMode == nil && c.Role == Destination && c.connector.SendingMode != nil {
				mode := Server
				if sm := *c.connector.SendingMode; sm == Client {
					mode = Client
				}
				c.SendingMode = &mode
			}
			connection, ok := state.connections[c.ID]
			if ok {
				*connection = c
			} else {
				connection = &Connection{}
				*connection = c
			}
			ws.connections[c.ID] = connection
			state.connections[c.ID] = connection
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Read all event write keys.
	err = tx.QueryScan(ctx, `SELECT connection, key FROM event_write_keys ORDER BY connection, created_at`,
		func(rows *db.Rows) error {
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
	err = tx.QueryScan(ctx, "SELECT id, connection, target, event_type, name, enabled, schedule_start,\n"+
		"schedule_period, in_schema, out_schema, filter, transformation_mapping, transformation_id,\n"+
		"transformation_version, transformation_language, transformation_source, transformation_preserve_json,\n"+
		"transformation_in_paths, transformation_out_paths, query, format, path, sheet, compression::TEXT,\n"+
		"order_by, format_settings, export_mode, matching_in, matching_out, update_on_duplicates, table_name,\n"+
		"table_key, identity_column, last_change_time_column, last_change_time_format, health, properties_to_unset\n"+
		"FROM actions",
		func(rows *db.Rows) error {
			for rows.Next() {
				var connectionID int
				var eventType string
				var rawInSchema, rawOutSchema, filter, mapping []byte
				var function TransformationFunction
				var format *string
				action := Action{}
				err := rows.Scan(&action.ID, &connectionID, &action.Target, &eventType, &action.Name,
					&action.Enabled, &action.ScheduleStart, &action.SchedulePeriod, &rawInSchema, &rawOutSchema,
					&filter, &mapping, &function.ID, &function.Version, &function.Language, &function.Source, &function.PreserveJSON,
					&action.Transformation.InPaths, &action.Transformation.OutPaths, &action.Query, &format,
					&action.Path, &action.Sheet, &action.Compression, &action.OrderBy, &action.FormatSettings, &action.ExportMode,
					&action.Matching.In, &action.Matching.Out, &action.UpdateOnDuplicates, &action.TableName,
					&action.TableKey, &action.IdentityColumn, &action.LastChangeTimeColumn,
					&action.LastChangeTimeFormat, &action.Health, &action.propertiesToUnset)
				if err != nil {
					return err
				}
				c := state.connections[connectionID]
				if format != nil {
					action.format = state.connectors[*format]
					if action.format == nil {
						return fmt.Errorf("the %s connector required by the %s '%s' is not included in the executable. "+
							"Recompile the executable with the %s connector to resolve the issue", *format, c.Role, c.Name, *format)
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
	err = tx.QueryScan(ctx, "SELECT id, action, cursor, incremental, start_time\n"+
		"FROM actions_executions\nWHERE end_time IS NULL",
		func(rows *db.Rows) error {
			for rows.Next() {
				exe := ActionExecution{
					mu: &sync.Mutex{},
				}
				var actionID int
				err := rows.Scan(&exe.ID, &actionID, &exe.Cursor, &exe.Incremental, &exe.StartTime)
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
	err = tx.QueryScan(ctx, "SELECT source, path FROM user_schema_primary_sources",
		func(rows *db.Rows) error {
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

	// Read the metadata, which include the installation ID and the encryption
	// key.
	err = tx.QueryScan(ctx, "SELECT key, value FROM metadata",
		func(rows *db.Rows) error {
			for rows.Next() {
				var key, value string
				err := rows.Scan(&key, &value)
				if err != nil {
					return err
				}
				switch key {
				case "encryption_key":
					state.metadata.encryptionKey, err = base64.StdEncoding.DecodeString(value)
					if err != nil {
						return fmt.Errorf("cannot decode value for 'encryption_key' as Base64: %s", err)
					}
				case "installation_id":
					state.metadata.installationID = value
				default:
					return fmt.Errorf("unexpected key %q in metadata", key)
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	if state.metadata.encryptionKey == nil {
		return errors.New("missing key 'encryption_key' in table 'metadata'")
	}
	if len(state.metadata.encryptionKey) != 64 {
		return errors.New("value for 'encryption_key' must be a Base64 string that encodes 64 bytes")
	}
	if state.metadata.installationID == "" {
		return errors.New("missing key 'installation_id' in table 'metadata'")
	}

	return state.notifications.CommitAndStartListening(ctx, tx, state.metadata.encryptionKey[:32])
}

// article returns "a" or "an" based on the first letter of the name.
func article(name string) string {
	switch name[0] {
	case 'A', 'E', 'I', 'O', 'U', 'a', 'e', 'i', 'o', 'u':
		return "an"
	}
	return "a"
}
