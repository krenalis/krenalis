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

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/apis/connectors"
	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/schemas"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/statistics"
	"github.com/meergo/meergo/apis/transformers"
)

// importUsers imports the users of the action.
func (this *Action) importUsers(ctx context.Context, stats *statistics.Collector) error {

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
		if !execution.Reload {
			lastChangeTime = execution.Cursor
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
				if execution.Reload {
					v, _ = database.LastChangeTimeCondition(nil)
				} else {
					purge = execution.Cursor.IsZero()
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
		if !execution.Reload && action.LastChangeTimeProperty != "" {
			lastChangeTime = execution.Cursor
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
		return newActionError(statistics.InputValidationStep, err)
	}
	defer records.Close()

	// Instantiate a batch identity writer.
	iw, err := this.connection.store.BatchIdentityWriter(action, purge, func(ids []string, err error) {
		if err != nil {
			stats.FailedFinalizing(len(ids), err.Error())
			return
		}
		stats.PassedFinalizing(len(ids))
	})
	if err != nil {
		if err == datastore.ErrInspectionMode || err == datastore.ErrMaintenanceMode {
			return newActionError(statistics.FinalizingStep, err)
		}
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the output schema, " + err.Msg + ". Please review and update the action before attempting to import the users."
			return newActionError(statistics.OutputValidationStep, err)
		}
		return err
	}
	// TODO(Gianluca): calling Close may return error in case the warehouse mode
	// does not allow the closing (that is the flushing of users). However,
	// before handling that error, we should instead address
	// https://github.com/meergo/meergo/issues/1002.
	defer iw.Close(ctx)

	users := make([]connectors.Record, 0, 100)
	transformationRecords := make([]transformers.Record, 0, 100)

	var cursor time.Time

	// Read the users.
	for user := range records.All(ctx) {

		if user.Err != nil {
			iw.Keep(user.ID)
			if _, ok := user.Err.(ValidationError); ok {
				stats.PassedReceiving(1)
				stats.FailedInputValidation(1, user.Err.Error())
				goto Next
			}
			stats.FailedReceiving(1, user.Err.Error())
			goto Next
		}

		stats.PassedReceiving(1)
		stats.PassedInputValidation(1)

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
					return newActionError(statistics.TransformationStep, err)
				}
				return err
			}

			// Set the identities into the data warehouse.
			for i, record := range transformationRecords {
				user := users[i]
				if record.Err != nil {
					if _, ok := record.Err.(ValidationError); ok {
						stats.PassedTransformation(1)
						stats.FailedOutputValidation(1, record.Err.Error())
						continue
					}
					stats.FailedTransformation(1, record.Err.Error())
					continue
				}
				user.Properties = record.Properties
				stats.PassedTransformation(1)
				stats.PassedOutputValidation(1)
				err = iw.Write(datastore.Identity{
					ID:             user.ID,
					Properties:     user.Properties,
					LastChangeTime: user.LastChangeTime,
				}, "")
				if err != nil {
					err := iw.Close(ctx)
					return newActionError(statistics.FinalizingStep, err)
				}
			}

			// Set the cursor.
			err = this.setExecutionCursor(ctx, cursor)
			if err != nil {
				return err
			}

			clear(users)
			users = users[0:0]

		}

	}
	if err = records.Err(); err != nil {
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the input schema, " + err.Msg + ". Please review and update the action before attempting to import the users."
			return newActionError(statistics.InputValidationStep, err)
		}
		if err == meergo.ErrSheetNotExist {
			err = fmt.Errorf("file does not contain any sheet named %q", action.Sheet)
		}
		return newActionError(statistics.ReceivingStep, err)
	}

	err = iw.Close(ctx)
	if err != nil {
		return newActionError(statistics.FinalizingStep, err)
	}

	return nil
}
