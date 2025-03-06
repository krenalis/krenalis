//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/filters"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/schemas"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
)

// importUsers imports the users of the action.
//
// Returns an error if execution does not reach its natural completion.
// If the error is caused by the schema, the connector, or the data warehouse,
// it returns an *actionError, which is expected to be logged as is.
func (this *Action) importUsers(ctx context.Context) error {

	action := this.action
	connection := action.Connection()
	connector := connection.Connector()
	execution, _ := action.Execution()

	// purge specifies whether identities should be purged from the
	// data warehouse after all identities have been written.
	purge := true

	transformer, err := transformers.New(action, this.core.functionProvider, nil)
	if err != nil {
		return err
	}

	var records connectors.Records

	switch connector.Type {
	case state.App:
		purge = execution.Cursor.IsZero()
		records, err = this.app().Users(ctx, action.InSchema, execution.Cursor)
	case state.Database:
		database := this.database()
		defer database.Close()
		replacer := func(name string) (string, bool) {
			switch name {
			case "last_change_time":
				var v string
				if execution.Incremental {
					purge = execution.Cursor.IsZero()
					v, _ = database.LastChangeTimePlaceholder(action)
				} else {
					v, _ = database.LastChangeTimePlaceholder(nil)
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
		if !execution.Cursor.IsZero() && action.LastChangeTimeColumn != "" {
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
		return newActionError(metrics.InputValidationStep, err)
	}
	defer records.Close()

	// Instantiate a batch identity writer.
	iw, err := this.connection.store.NewBatchIdentityWriter(action, purge, func(ids []string, err error) {
		if err != nil {
			this.core.metrics.FinalizeFailed(action.ID, len(ids), err.Error())
			return
		}
		this.core.metrics.FinalizePassed(action.ID, len(ids))
	})
	if err != nil {
		if err == datastore.ErrInspectionMode || err == datastore.ErrMaintenanceMode {
			return newActionError(metrics.FinalizeStep, err)
		}
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the output schema, " + err.Msg + ". Please review and update the action before attempting to import the users."
			return newActionError(metrics.OutputValidationStep, err)
		}
		return err
	}
	// TODO(Gianluca): calling Close may return error in case the warehouse mode
	// does not allow the closing (that is the flushing of users). However,
	// before handling that error, we should instead address
	// https://github.com/meergo/meergo/issues/1224.
	defer iw.Close(ctx)

	users := make([]connectors.Record, 0, 100)
	transformationRecords := make([]transformers.Record, 0, 100)

	var cursor time.Time

	// Read the users.
	for user := range records.All(ctx) {

		if user.Err != nil {
			iw.Keep(user.ID)
			if _, ok := user.Err.(validationError); ok {
				this.core.metrics.ReceivePassed(action.ID, 1)
				this.core.metrics.InputValidationFailed(action.ID, 1, user.Err.Error())
				goto Next
			}
			this.core.metrics.ReceiveFailed(action.ID, 1, user.Err.Error())
			goto Next
		}

		this.core.metrics.ReceivePassed(action.ID, 1)
		this.core.metrics.InputValidationPassed(action.ID, 1)

		// In case the action has a filter, check if it applies to the user.
		if connector.Type != state.Database {
			if !filters.Applies(action.Filter, user.Properties) {
				this.core.metrics.FilterFailed(action.ID, 1)
				goto Next
			}
			this.core.metrics.FilterPassed(action.ID, 1)
		}

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
					return newActionError(metrics.TransformationStep, err)
				}
				return err
			}

			// Set the identities into the data warehouse.
			for i, record := range transformationRecords {
				user := users[i]
				if record.Err != nil {
					if _, ok := record.Err.(validationError); ok {
						this.core.metrics.TransformationPassed(action.ID, 1)
						this.core.metrics.OutputValidationFailed(action.ID, 1, record.Err.Error())
						continue
					}
					this.core.metrics.TransformationFailed(action.ID, 1, record.Err.Error())
					continue
				}
				user.Properties = record.Properties
				this.core.metrics.TransformationPassed(action.ID, 1)
				this.core.metrics.OutputValidationPassed(action.ID, 1)
				err = iw.Write(datastore.Identity{
					ID:             user.ID,
					Properties:     user.Properties,
					LastChangeTime: user.LastChangeTime,
				}, "")
				if err != nil {
					err := iw.Close(ctx)
					return newActionError(metrics.FinalizeStep, err)
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
			return newActionError(metrics.InputValidationStep, err)
		}
		if err == meergo.ErrSheetNotExist {
			err = fmt.Errorf("file does not contain any sheet named %q", action.Sheet)
		}
		return newActionError(metrics.ReceiveStep, err)
	}

	err = iw.Close(ctx)
	if err != nil {
		return newActionError(metrics.FinalizeStep, err)
	}

	return nil
}
