//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chichi/apis/mappings"
	"chichi/apis/normalization"
	"chichi/connector/types"
)

// exportUsersToFile exports the users to the file.
func (this *Action) exportUsersToFile(ctx context.Context) error {

	users, err := this.readUsersFromDataWarehouse(ctx, nil)
	if err != nil {
		return err
	}

	// Filter the users.
	if this.action.Filter != nil {
		filteredUsers := []userToExport{}
		for _, user := range users {
			ok, err := mappings.FilterApplies(this.action.Filter, user.Properties)
			if err != nil {
				return err
			}
			if ok {
				filteredUsers = append(filteredUsers, user)
			}
		}
		users = filteredUsers
	}

	connection := this.action.Connection()

	// Determine the columns of the exported file from the "users" schema.
	usersSchema, ok := connection.Workspace().Schemas["users"]
	if !ok {
		return actionExecutionError{errors.New("'users' schema not found")}
	}
	columns := usersSchema.Properties()

	// Prepare the users.
	rows := make([][]any, len(users))
	for i, u := range users {
		userSlice := make([]any, len(columns))
		for j, c := range columns {
			userSlice[j] = u.Properties[c.Name]
		}
		rows[i] = userSlice
	}

	// Write the file.
	path, err := replacePlaceholders(this.action.Path, newPathPlaceholderReplacer(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("invalid path: %s", err)
	}
	err = this.file().Write(ctx, path, this.action.Sheet, columns, rows)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot write file: %s", err)}
	}

	return nil
}

// importUsersFromFile imports the users from a file.
func (this *Action) importUsersFromFile(ctx context.Context) error {

	// Determine the input and the output schema.
	mapping, err := mappings.New(this.action.InSchema, this.action.OutSchema, this.action.Mapping,
		this.action.Transformation, this.action.ID, this.apis.transformer, nil)
	if err != nil {
		return err
	}

	properties := this.action.InSchema.PropertiesNames()

	// Read the records.
	err = this.file().ReadFunc(ctx, this.action.Path, this.action.Sheet, func(columns []types.Property, record map[string]any) error {

		// Determine and validate the external ID and the timestamp.
		var idCol, timestampCol types.Property
		for _, c := range columns {
			switch c.Name {
			case this.action.IdentityColumn:
				idCol = c
			case this.action.TimestampColumn:
				timestampCol = c
			}
		}
		if idCol.Name == "" {
			return actionExecutionError{fmt.Errorf("identity column '%s' does not exist in file", this.action.IdentityColumn)}
		}
		var externalID string
		{
			rawID, ok := record[idCol.Name]
			if !ok {
				return actionExecutionError{fmt.Errorf("column '%s' not present in file record", idCol.Name)}
			}
			switch pt := idCol.Type.PhysicalType(); pt {
			case types.PtInt, types.PtUint, types.PtJSON, types.PtText:
				externalID = fmt.Sprint(rawID)
			default:
				return actionExecutionError{fmt.Errorf("column '%s' with type %s cannot be used as identifier", idCol.Name, pt)}
			}
		}

		var timestamp time.Time
		if timestampCol.Name != "" {

			// Validate the physical type.
			switch pt := timestampCol.Type.PhysicalType(); pt {
			case types.PtText, types.PtJSON, types.PtDateTime:
				// Ok.
			default:
				return actionExecutionError{fmt.Errorf("column '%s' with type %s cannot be used as timestamp", timestampCol.Name, pt)}
			}

			// Retrieve the value for the timestamp.
			rawTimestamp, ok := record[timestampCol.Name]
			if !ok {
				return actionExecutionError{fmt.Errorf("no values for '%s' returned in file record", timestampCol.Name)}
			}

			// Normalize the value.
			rawTimestamp, err = normalization.NormalizeDatabaseFileProperty(timestampCol.Name, timestampCol.Type, rawTimestamp, false)
			if err != nil {
				return actionExecutionError{fmt.Errorf("column '%s' cannot be used as timestamp: %s", timestampCol.Name, err)}
			}

			// Determine the timestamp.
			switch ts := rawTimestamp.(type) {
			case string:
				timestamp, err = time.Parse(this.action.TimestampFormat, ts)
			case time.Time:
				timestamp, err = ts, nil
			default:
				return actionExecutionError{fmt.Errorf("invalid value for column '%s', cannot be used as timestamp", timestampCol.Name)}
			}
			if err != nil {
				return actionExecutionError{fmt.Errorf("invalid value for column '%s': %s", timestampCol.Name, err)}
			}

		}

		// Take only the necessary properties.
		props := make(map[string]any, len(properties))
		for _, name := range properties {
			if v, ok := record[name]; ok {
				props[name] = v
			}
		}

		// Normalize the user properties (read from the file) using the action's
		// mapping input schema.
		props, err := normalize(props, this.action.InSchema)
		if err != nil {
			if err, ok := err.(mappings.Error); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Map the properties of the user.
		mappedUser, err := mapping.Apply(ctx, props)
		if err != nil {
			if err, ok := err.(mappings.Error); ok {
				return actionExecutionError{err}
			}
			return err
		}

		// Set the identity into the data warehouse.
		err = this.connection.store.SetIdentity(ctx, mappedUser, externalID, "", this.action.ID, false, timestamp)
		if err != nil {
			return actionExecutionError{err}
		}

		// Update the connection stats.
		err = this.connection.updateConnectionsStats(ctx)
		if err != nil {
			return actionExecutionError{err}
		}

		return nil
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot read the file: %s", err)}
	}

	// Resolve and sync the users.
	err = this.connection.store.ResolveSyncUsers(ctx)
	if err != nil {
		return actionExecutionError{err}
	}

	return nil
}

// newPathPlaceholderReplacer returns a placeholder replacer that replaces the
// following placeholders using time.Now().UTC() as current time.
//
//	${today}  which renders to something like:  2035-10-30
//	${now}    which renders to something like:  2035-10-30-16-33-25
//	${unix}   which renders to something like:  2077374805
//
// These placeholders are case-insensitive, so ${TODAY} is handled like
// ${today}.
func newPathPlaceholderReplacer(t time.Time) func(string) (string, bool) {
	return func(name string) (string, bool) {
		switch strings.ToLower(name) {
		case "today":
			return t.Format(time.DateOnly), true
		case "now":
			return t.Format("2006-01-02-15-04-05"), true
		case "unix":
			return strconv.FormatInt(t.Unix(), 10), true
		}
		return "", false
	}
}
