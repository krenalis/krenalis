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

	"github.com/meergo/meergo/apis/connectors"
	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/schemas"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/statistics"
	"github.com/meergo/meergo/apis/transformers"
)

// importUsers imports the users of the action.
func (this *Action) importUsers(ctx context.Context, stats *statistics.ActionCollector) error {

	action := this.action
	connection := action.Connection()
	connector := connection.Connector()
	execution, _ := action.Execution()

	// purge specifies whether identities should be purged from the
	// data warehouse after all identities have been written.
	purge := true

	transformer, err := transformers.New(action, this.apis.transformerProvider, nil)
	if err != nil {
		return err
	}

	var records connectors.Records

	switch connector.Type {
	case state.App:
		var lastChangeTime time.Time
		if !execution.Reimport {
			lastChangeTime = action.UserCursor
		}
		purge = lastChangeTime.IsZero()
		records, err = this.app().Users(ctx, action.InSchema, lastChangeTime)
	case state.Database:
		database := this.database()
		defer database.Close()
		replacer := func(name string) (string, bool) {
			switch name {
			case "last_change_time":
				var v string
				if execution.Reimport {
					v, _ = database.LastChangeTimeCondition(nil)
				} else {
					purge = action.UserCursor.IsZero()
					v, _ = database.LastChangeTimeCondition(action)
				}
				return v, true
			case "limit":
				return strconv.FormatUint(math.MaxInt64, 10), true
			}
			return "", false
		}
		records, err = database.Records(ctx, action, replacer)
	case state.FileStorage:
		var lastChangeTime time.Time
		if !execution.Reimport && action.LastChangeTimeProperty != "" {
			lastChangeTime = action.UserCursor
		}
		purge = lastChangeTime.IsZero()
		records, err = this.file().Records(ctx, lastChangeTime)
	default:
		return fmt.Errorf("invalid connector type %s", connector.Type)
	}
	if err != nil {
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the input schema, " + err.Msg + ". Please review and update the action before attempting to import the users."
		}
		return actionExecutionError{err}
	}
	defer records.Close()

	// Instantiate a batch identity writer.
	iw, err := this.connection.store.BatchIdentityWriter(action, purge, func(ids []string, err error) {
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
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the output schema, " + err.Msg + ". Please review and update the action before attempting to import the users."
			return actionExecutionError{err}
		}
		return err
	}
	defer iw.Close(ctx)

	users := make([]connectors.Record, 0, 100)
	transformationRecords := make([]transformers.Record, 0, 100)

	var cursor time.Time

	// Read the users.
	for user := range records.All(ctx) {

		if user.Err != nil {
			iw.Keep(user.ID)
			if _, ok := user.Err.(ValidationError); ok {
				stats.Passed(statistics.Receiving)
				stats.Failed(statistics.InputValidation, user.Err.Error())
				goto Next
			}
			stats.Failed(statistics.Receiving, user.Err.Error())
			goto Next
		}

		stats.Passed(statistics.Receiving)
		stats.Passed(statistics.InputValidation)

		if user.LastChangeTime.After(cursor) {
			cursor = user.LastChangeTime
		}

		users = append(users, user)

	Next:

		// Does a batch processing of users.
		if len(users) == 100 || records.Last() {

			// Transform the users.
			transformationRecords = transformationRecords[0:len(users)]
			for i, user := range users {
				transformationRecords[i].Properties = user.Properties
			}
			err := transformer.Transform(ctx, transformationRecords)
			if err != nil {
				if err, ok := err.(transformers.FunctionExecutionError); ok {
					return actionExecutionError{err}
				}
				return err
			}

			// Set the identities into the data warehouse.
			for i, record := range transformationRecords {
				user := users[i]
				if record.Err != nil {
					if _, ok := record.Err.(ValidationError); ok {
						stats.Passed(statistics.Transformation)
						stats.Failed(statistics.OutputValidation, record.Err.Error())
						continue
					}
					stats.Failed(statistics.Transformation, record.Err.Error())
					continue
				}
				user.Properties = record.Properties
				stats.Passed(statistics.Transformation)
				stats.Passed(statistics.OutputValidation)
				err = iw.Write(datastore.Identity{
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
			err = this.setUserCursor(ctx, cursor)
			if err != nil {
				return actionExecutionError{err}
			}

			clear(users)
			users = users[0:0]

		}

	}
	if err = records.Err(); err != nil {
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the input schema, " + err.Msg + ". Please review and update the action before attempting to import the users."
		}
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

	return nil
}
