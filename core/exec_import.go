// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/filters"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
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

	var records connections.Records

	switch connector.Type {
	case state.API:
		purge = execution.Cursor.IsZero()
		records, err = this.api().Users(ctx, action.InSchema, nil, execution.Cursor)
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
	// Cancel the writer, or does nothing if it is already closed.
	defer iw.Cancel(ctx)

	users := make([]connections.Record, 0, 100)
	transformationRecords := make([]transformers.Record, 0, 100)

	var cursor time.Time

	// Read the users.
	for user := range records.All(ctx) {

		if user.Err != nil {
			iw.Keep(user.ID)
			if err, ok := user.Err.(connections.InputValidationError); ok {
				this.core.metrics.ReceivePassed(action.ID, 1)
				this.core.metrics.InputValidationFailed(action.ID, 1, err.Error())
			} else {
				this.core.metrics.ReceiveFailed(action.ID, 1, user.Err.Error())
			}
			goto Next
		}

		this.core.metrics.ReceivePassed(action.ID, 1)
		this.core.metrics.InputValidationPassed(action.ID, 1)

		// In case the action has a filter, check if it applies to the user.
		if connector.Type != state.Database {
			if !filters.Applies(action.Filter, user.Attributes) {
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
				transformationRecords[i].Attributes = user.Attributes
			}
			err := transformer.Transform(ctx, transformationRecords)
			if err != nil {
				if _, ok := err.(transformers.FunctionExecError); ok {
					err = newActionError(metrics.TransformationStep, err)
				}
				return err
			}

			// Set the identities into the data warehouse.
			for i, record := range transformationRecords {
				user := users[i]
				if err := record.Err; err != nil {
					switch err.(type) {
					case transformers.RecordTransformationError:
						this.core.metrics.TransformationFailed(action.ID, 1, err.Error())
					case transformers.RecordValidationError:
						this.core.metrics.TransformationPassed(action.ID, 1)
						this.core.metrics.OutputValidationFailed(action.ID, 1, err.Error())
					}
					iw.Keep(user.ID)
					continue
				}
				user.Attributes = record.Attributes
				this.core.metrics.TransformationPassed(action.ID, 1)
				this.core.metrics.OutputValidationPassed(action.ID, 1)
				iw.Write(datastore.Identity{
					ID:             user.ID,
					Attributes:     user.Attributes,
					LastChangeTime: user.LastChangeTime,
				})
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
		if err == connectors.ErrSheetNotExist {
			err = fmt.Errorf("file does not contain any sheet named %q", action.Sheet)
		}
		return newActionError(metrics.ReceiveStep, err)
	}

	// TODO(Gianluca): calling Close may return error in case the warehouse mode
	// does not allow the closing (that is the flushing of users). However,
	// before handling that error, we should instead address
	// https://github.com/meergo/meergo/issues/1224.
	err = iw.Close(ctx)
	if err != nil {
		if err != datastore.ErrPurgeSkipped {
			return newActionError(metrics.FinalizeStep, err)
		}
		this.core.metrics.FinalizeFailed(action.ID, 0, "unimported records not deleted because at least one with errors could not be identified")
	}

	return nil
}
