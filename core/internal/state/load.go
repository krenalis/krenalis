// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/db"
	_json "github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/warehouses"

	"github.com/google/uuid"
)

// load loads the state.
func (state *State) load(oauthCredentials map[string]*OAuthCredentials) error {

	// Read all connectors.
	conns := connectors.Connectors()
	state.connectors = make(map[string]*Connector, len(conns))
	for code, connector := range conns {
		c := Connector{}
		switch connector := connector.(type) {
		case connectors.APISpec:
			c.Code = connector.Code
			c.Label = connector.Label
			c.Type = API
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
			case connectors.Client:
				mode := Client
				c.SendingMode = &mode
			case connectors.Server:
				mode := Server
				c.SendingMode = &mode
			case connectors.ClientAndServer:
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
			if oauthCredentials != nil {
				if oAuth, ok := oauthCredentials[c.Code]; ok {
					c.OAuth.ClientID = oAuth.ClientID
					c.OAuth.ClientSecret = oAuth.ClientSecret
				}
			}
		case connectors.DatabaseSpec:
			c.Code = connector.Code
			c.Label = connector.Label
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
			c.Documentation = connector.Documentation
			if summary := c.Documentation.Source.Summary; summary == "" {
				c.Documentation.Source.Summary = "Import users from " + article(c.Label) + " " + c.Label + " database"
			}
			if summary := c.Documentation.Destination.Summary; summary == "" {
				c.Documentation.Destination.Summary = "Exports users to " + article(c.Label) + " " + c.Label + " database"
			}
		case connectors.FileSpec:
			c.Code = connector.Code
			c.Label = connector.Label
			c.Type = File
			c.Categories = connector.Categories
			if asSource := connector.AsSource; asSource != nil {
				c.SourceTargets = UsersFlag
				c.HasSourceSettings = asSource.HasSettings
				c.Documentation.Source = asSource.Documentation
				if c.Documentation.Source.Summary == "" {
					c.Documentation.Source.Summary = "Import users from " + article(c.Label) + " " + c.Label + " file"
				}
			}
			if asDest := connector.AsDestination; asDest != nil {
				c.DestinationTargets = UsersFlag
				c.HasDestinationSettings = connector.AsDestination.HasSettings
				c.Documentation.Destination = asDest.Documentation
				if c.Documentation.Destination.Summary == "" {
					c.Documentation.Destination.Summary = "Export users to " + article(c.Label) + " " + c.Label + " file"
				}
			}
			c.FileExtension = connector.Extension
			c.TimeLayouts = TimeLayouts(connector.TimeLayouts)
			c.HasSheets = connector.HasSheets
		case connectors.FileStorageSpec:
			c.Code = connector.Code
			c.Label = connector.Label
			c.Type = FileStorage
			c.Categories = connector.Categories
			if asSource := connector.AsSource; asSource != nil {
				c.SourceTargets = UsersFlag
				// It is assumed that, if a FileStorage connector can be
				// used as a source, it always has source settings.
				c.HasSourceSettings = true
				c.Documentation.Source = asSource.Documentation
				if c.Documentation.Source.Summary == "" {
					c.Documentation.Source.Summary = "Import users from a file on " + c.Label
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
					c.Documentation.Source.Summary = "Exports users to a file on " + c.Label
				}
			}
		case connectors.MessageBrokerSpec:
			c.Code = connector.Code
			c.Label = connector.Label
			c.Type = MessageBroker
			c.Categories = connector.Categories
			c.SourceTargets = EventsFlag
			// It is assumed that a message broker connector always have settings.
			c.HasSourceSettings = true
			c.HasDestinationSettings = true
			c.Documentation = connector.Documentation
		case connectors.SDKSpec:
			c.Code = connector.Code
			c.Label = connector.Label
			c.Type = SDK
			c.Categories = connector.Categories
			c.Terms = ConnectorTerms{
				User:  "user",
				Users: "users",
				// Group:  "group", TODO(marco): Implement groups
				// Groups: "groups",
			}
			c.SourceTargets = EventsFlag | UsersFlag
			c.Strategies = connector.Strategies
			c.FallbackToRequestIP = connector.FallbackToRequestIP
			c.Documentation = connector.Documentation
		case connectors.WebhookSpec:
			c.Code = connector.Code
			c.Label = connector.Label
			c.Type = Webhook
			c.Categories = connector.Categories
			c.Terms = ConnectorTerms{
				User:  "user",
				Users: "users",
				// Group:  "group", TODO(marco): Implement groups
				// Groups: "groups",
			}
			c.SourceTargets = EventsFlag | UsersFlag
			c.Documentation = connector.Documentation
		}
		state.connectors[code] = &c
	}

	// Read all warehouse drivers.
	drivers := warehouses.Drivers()
	state.warehouseDrivers = make(map[string]WarehouseDriver, len(drivers))
	for _, driver := range drivers {
		state.warehouseDrivers[driver.Name] = WarehouseDriver{
			Name: driver.Name,
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
	state.organizations = map[uuid.UUID]*Organization{}
	err = tx.QueryScan(ctx, "SELECT id, name FROM organizations", func(rows *db.Rows) error {
		var id uuid.UUID
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
			organization.members = map[int]struct{}{}
			state.organizations[id] = organization
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Read all members.
	err = tx.QueryScan(ctx, "SELECT id, organization FROM members ORDER BY organization", func(rows *db.Rows) error {
		var id int
		var organization uuid.UUID
		var org *Organization
		for rows.Next() {
			if err := rows.Scan(&id, &organization); err != nil {
				return err
			}
			if org == nil || !bytes.Equal(org.ID[:], organization[:]) {
				org = state.organizations[organization]
			}
			org.members[id] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Read all workspaces.
	state.workspaces = map[int]*Workspace{}
	err = tx.QueryScan(ctx, "SELECT id, organization, name, warehouse_name,"+
		" warehouse_mode, warehouse_settings, warehouse_mcp_settings, alter_profile_schema_id,"+
		" alter_profile_schema_schema, alter_profile_schema_primary_sources, alter_profile_schema_operations,"+
		" alter_profile_schema_start_time, alter_profile_schema_end_time,"+
		" alter_profile_schema_error, profile_schema, resolve_identities_on_batch_import,"+
		" identifiers, ir_id, ir_start_time, ir_end_time, ui_profile_image,"+
		" ui_profile_first_name, ui_profile_last_name, ui_profile_extra, pipelines_to_purge "+
		"FROM workspaces",
		func(rows *db.Rows) error {
			var organizationID uuid.UUID
			var warehouseName string
			var warehouseMode WarehouseMode
			var userSchema []byte
			var alterProfileSchemaSchema []byte
			var warehouseSettings, warehouseMCPSettings []byte
			for rows.Next() {
				ws := &Workspace{
					mu:          new(sync.Mutex),
					connections: map[int]*Connection{},
					executions:  map[int]*PipelineExecution{},
					accounts:    map[int]*Account{},
				}
				if err := rows.Scan(&ws.ID, &organizationID, &ws.Name, &warehouseName,
					&warehouseMode, &warehouseSettings, &warehouseMCPSettings, &ws.AlterProfileSchema.ID,
					&alterProfileSchemaSchema, &ws.AlterProfileSchema.PrimarySources,
					&ws.AlterProfileSchema.Operations, &ws.AlterProfileSchema.StartTime,
					&ws.AlterProfileSchema.EndTime, &ws.AlterProfileSchema.Err, &userSchema,
					&ws.ResolveIdentitiesOnBatchImport, &ws.Identifiers, &ws.IR.ID,
					&ws.IR.StartTime, &ws.IR.EndTime, &ws.UIPreferences.Profile.Image,
					&ws.UIPreferences.Profile.FirstName, &ws.UIPreferences.Profile.LastName,
					&ws.UIPreferences.Profile.Extra, &ws.pipelinesToPurge); err != nil {
					return err
				}
				ws.organization = state.organizations[organizationID]
				if _, ok := state.warehouseDrivers[warehouseName]; !ok {
					return fmt.Errorf("warehouse driver for %q is required but not registered. (Possibly forgotten import?)", warehouseName)
				}
				ws.Warehouse.Name = warehouseName
				ws.Warehouse.Mode = warehouseMode
				ws.Warehouse.Settings = warehouseSettings
				if _json.Value(warehouseMCPSettings).IsNull() {
					ws.Warehouse.MCPSettings = nil
				} else {
					ws.Warehouse.MCPSettings = warehouseMCPSettings
				}
				err = json.Unmarshal(userSchema, &ws.ProfileSchema)
				if err != nil {
					return err
				}
				err = json.Unmarshal(alterProfileSchemaSchema, &ws.AlterProfileSchema.Schema)
				if err != nil {
					return err
				}
				ws.PrimarySources = map[string]int{}
				ws.organization.workspaces[ws.ID] = ws
				state.workspaces[ws.ID] = ws
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all access keys.
	state.accessKeyByToken = map[string]*AccessKey{}
	err = tx.QueryScan(ctx, "SELECT id, organization, workspace, type, token FROM access_keys",
		func(rows *db.Rows) error {
			for rows.Next() {
				k := AccessKey{}
				var token string
				var workspace *int
				if err := rows.Scan(&k.ID, &k.Organization, &workspace, &k.Type, &token); err != nil {
					return err
				}
				if workspace != nil {
					k.Workspace = *workspace
				}
				state.accessKeyByToken[token] = &k
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
			c.pipelines = map[int]*Pipeline{}
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
			if c.LinkedConnections == nil {
				targets := c.connector.SourceTargets
				if c.Role == Destination {
					targets = c.connector.DestinationTargets
				}
				if targets.Contains(TargetEvent) {
					c.LinkedConnections = []int{}
				}
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

	// Read all pipelines.
	err = tx.QueryScan(ctx, "SELECT id, connection, target, event_type, name, enabled, schedule_start,\n"+
		"schedule_period, in_schema, out_schema, filter, transformation_mapping, transformation_id,\n"+
		"transformation_version, transformation_language, transformation_source, transformation_preserve_json,\n"+
		"transformation_in_paths, transformation_out_paths, query, format, path, sheet, compression::TEXT,\n"+
		"order_by, format_settings, export_mode, matching_in, matching_out, update_on_duplicates, table_name,\n"+
		"table_key, identity_column, last_change_time_column, last_change_time_format, health, properties_to_unset\n"+
		"FROM pipelines",
		func(rows *db.Rows) error {
			for rows.Next() {
				var connectionID int
				var eventType string
				var rawInSchema, rawOutSchema, filter, mapping []byte
				var function TransformationFunction
				var format *string
				pipeline := Pipeline{}
				err := rows.Scan(&pipeline.ID, &connectionID, &pipeline.Target, &eventType, &pipeline.Name,
					&pipeline.Enabled, &pipeline.ScheduleStart, &pipeline.SchedulePeriod, &rawInSchema, &rawOutSchema,
					&filter, &mapping, &function.ID, &function.Version, &function.Language, &function.Source, &function.PreserveJSON,
					&pipeline.Transformation.InPaths, &pipeline.Transformation.OutPaths, &pipeline.Query, &format,
					&pipeline.Path, &pipeline.Sheet, &pipeline.Compression, &pipeline.OrderBy, &pipeline.FormatSettings, &pipeline.ExportMode,
					&pipeline.Matching.In, &pipeline.Matching.Out, &pipeline.UpdateOnDuplicates, &pipeline.TableName,
					&pipeline.TableKey, &pipeline.IdentityColumn, &pipeline.LastChangeTimeColumn,
					&pipeline.LastChangeTimeFormat, &pipeline.Health, &pipeline.propertiesToUnset)
				if err != nil {
					return err
				}
				c := state.connections[connectionID]
				if format != nil {
					pipeline.format = state.connectors[*format]
					if pipeline.format == nil {
						return fmt.Errorf("the %s connector required by the %s '%s' is not included in the executable. "+
							"Recompile the executable with the %s connector to resolve the issue", *format, c.Role, c.Name, *format)
					}
				}
				pipeline.mu = new(sync.Mutex)
				pipeline.connection = c
				pipeline.EventType = eventType
				err = pipeline.InSchema.UnmarshalJSON(rawInSchema)
				if err != nil {
					// TODO(marco) disable the pipeline instead of returning an error
					return err
				}
				err = pipeline.OutSchema.UnmarshalJSON(rawOutSchema)
				if err != nil {
					// TODO(marco) disable the pipeline instead of returning an error
					return err
				}
				if filter != nil {
					pipeline.Filter, err = unmarshalWhere(filter, pipeline.InSchema)
					if err != nil {
						return err
					}
				}
				if len(mapping) > 0 {
					err = json.Unmarshal(mapping, &pipeline.Transformation.Mapping)
					if err != nil {
						return err
					}
				}
				if function.Source != "" {
					pipeline.Transformation.Function = &function
				}
				state.pipelines[pipeline.ID] = &pipeline
				c.pipelines[pipeline.ID] = &pipeline
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read running pipeline executions.
	err = tx.QueryScan(ctx, "SELECT id, pipeline, cursor, incremental, start_time\n"+
		"FROM pipelines_executions\nWHERE end_time IS NULL",
		func(rows *db.Rows) error {
			for rows.Next() {
				exe := PipelineExecution{
					mu: &sync.Mutex{},
				}
				var pipelineID int
				err := rows.Scan(&exe.ID, &pipelineID, &exe.Cursor, &exe.Incremental, &exe.StartTime)
				if err != nil {
					return err
				}
				pipeline := state.pipelines[pipelineID]
				exe.pipeline = pipeline
				pipeline.execution = &exe
				ws := exe.pipeline.connection.workspace
				ws.executions[exe.ID] = &exe
			}
			return nil
		})
	if err != nil {
		return err
	}

	// Read all primary sources.
	err = tx.QueryScan(ctx, "SELECT source, path FROM primary_sources",
		func(rows *db.Rows) error {
			var source int
			var path string
			for rows.Next() {
				if err := rows.Scan(&source, &path); err != nil {
					return err
				}
				c := state.connections[source]
				c.workspace.PrimarySources[path] = source
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
