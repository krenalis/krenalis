//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/statistics"
	"github.com/open2b/chichi/apis/transformers"
)

// importUsers imports the users of the action.
func (this *Action) importUsers(ctx context.Context, stats *statistics.ActionCollector) error {

	action := this.action
	connection := action.Connection()
	connector := connection.Connector()
	execution, _ := action.Execution()

	transformer, err := transformers.New(action.InSchema, action.OutSchema, action.Transformation, action.ID,
		this.apis.functionTransformer, nil)
	if err != nil {
		return err
	}

	var records connectors.Records

	switch connector.Type {
	case state.AppType:
		var lastChangeTime time.Time
		if !execution.Reimport {
			lastChangeTime = action.UserCursor
		}
		if execution.Reimport {
			err = this.connection.store.DeleteConnectionIdentities(ctx, action.Connection().ID)
			if err != nil {
				if err == datastore.ErrInspectionMode || err == datastore.ErrMaintenanceMode {
					return actionExecutionError{err}
				}
				if err, ok := err.(*warehouses.DataWarehouseError); ok {
					return actionExecutionError{fmt.Errorf("cannot delete the already-existing identities before starting the import: %s", err)}
				}
				return err
			}
		}
		records, err = this.app().Users(ctx, action.InSchema, lastChangeTime)
	case state.DatabaseType:
		replacer := func(name string) (string, bool) {
			if name == "limit" {
				return strconv.FormatUint(math.MaxInt64, 10), true
			}
			return "", false
		}
		database := this.database()
		defer database.Close()
		records, err = database.Records(ctx, action, replacer)
	case state.FileStorageType:
		var lastChangeTime time.Time
		if !execution.Reimport && action.LastChangeTimeProperty != "" {
			lastChangeTime = action.UserCursor
		}
		records, err = this.file().Records(ctx, lastChangeTime)
	default:
		return fmt.Errorf("invalid connector type %s", connector.Type)
	}
	if err != nil {
		if err, ok := err.(*connectors.SchemaError); ok {
			err.Msg += ". Please review and update the action before attempting to import the users."
		}
		return actionExecutionError{err}
	}
	defer records.Close()

	// Instantiate an identity writer.
	iw, err := this.connection.store.IdentityWriter(this.action.OutSchema, connection.ID, func(ids []string, err error) {
		if err != nil {
			stats.FailedCount(statistics.Finalizing, len(ids), err.Error())
			return
		}
		stats.PassedCount(statistics.Finalizing, len(ids))
	})
	if err != nil {
		if err == datastore.ErrInspectionMode || err == datastore.ErrMaintenanceMode {
			return actionExecutionError{err}
		}
		return err
	}
	defer iw.Close(ctx)

	var (
		users  = make([]connectors.Record, 0, 100)
		values = make([]map[string]any, 0, 100)
	)

	var cursor time.Time

	// Read the users.
	for user := range records.All(ctx) {

		if user.Err != nil {
			if _, ok := user.Err.(ValidationError); ok {
				stats.Passed(statistics.Receiving)
				stats.Failed(statistics.InputValidation, user.Err.Error())
				continue
			}
			stats.Failed(statistics.Receiving, user.Err.Error())
			continue
		}

		stats.Passed(statistics.Receiving)
		stats.Passed(statistics.InputValidation)

		if (connector.Type == state.AppType || connector.Type == state.FileStorageType) && user.LastChangeTime.After(cursor) {
			cursor = user.LastChangeTime
		}

		users = append(users, user)

		// Does a batch processing of users.
		if len(users) == 100 || records.Last() {

			// Transform the users.
			values = values[0:len(users)]
			for i, user := range users {
				values[i] = user.Properties
			}
			results, err := transformer.TransformValues(ctx, values)
			if err != nil {
				if err, ok := err.(transformers.FunctionExecutionError); ok {
					return actionExecutionError{err}
				}
				return err
			}

			// Set the identities into the data warehouse.
			for i, result := range results {
				user := users[i]
				if result.Err != nil {
					if _, ok := result.Err.(ValidationError); ok {
						stats.Passed(statistics.Transformation)
						stats.Failed(statistics.OutputValidation, result.Err.Error())
						continue
					}
					stats.Failed(statistics.Transformation, result.Err.Error())
					continue
				}
				user.Properties = result.Value
				stats.Passed(statistics.Transformation)
				stats.Passed(statistics.OutputValidation)
				err = iw.Write(datastore.Identity{
					Action:         action.ID,
					ID:             user.ID,
					Properties:     user.Properties,
					LastChangeTime: user.LastChangeTime,
				}, "")
				if err != nil {
					err := iw.Close(ctx)
					return actionExecutionError{err}
				}
			}

			// Set the user cursor.
			if connector.Type == state.AppType || connector.Type == state.FileStorageType {
				err = this.setUserCursor(ctx, cursor)
				if err != nil {
					return actionExecutionError{err}
				}
			}

			clear(users)
			users = users[0:0]

		}

	}
	if err = records.Err(); err != nil {
		if err == connectors.ErrSheetNotExist {
			err = fmt.Errorf("file does not contain any sheet named %q", action.Sheet)
		}
		return actionExecutionError{err}
	}

	err = iw.Close(ctx)
	if err != nil {
		return actionExecutionError{err}
	}

	users = nil

	// Run the Identity Resolution.
	err = this.connection.store.RunIdentityResolution(ctx)
	if err != nil {
		return actionExecutionError{fmt.Errorf("error while running the Identity Resolution at the end of the import: %s", err)}
	}

	return nil
}
