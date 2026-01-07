// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/metrics"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

var ErrPipelineNotExist = errors.New("pipeline does not exist")

// ErrPurgeSkipped is returned when the purge phase is skipped because some
// records without known IDs encountered an error.
var ErrPurgeSkipped = errors.New("purge skipped")

var maxQueuedIdentityRows = 1000
var maxQueuedIdentityTime = 500 * time.Millisecond

// Identity is an identity
type Identity struct {
	ID          string                 // Identifier of the identity; it is empty for anonymous identities.
	AnonymousID string                 // AnonymousID of identities received via events.
	Attributes  map[string]interface{} // Attributes. Keys are profile schema's properties.
	UpdatedAt   time.Time              // Update time in UTC.
}

// identityKey represents a key in the meergo_identities table.
type identityKey struct {
	pipeline    int
	isAnonymous bool
	// identityID is the identity ID for non-anonymous identities, while it is
	// the anonymous ID for anonymous ones.
	identityID string
}

// BatchIdentityWriter writes identities into the data warehouse in the case
// when identities are imported in batch.
type BatchIdentityWriter struct {
	store      *Store
	pipeline   int
	connection int
	run        int
	flatter    *flatter
	columns    []warehouses.Column
	purge      bool
	skipPurge  bool

	mu           sync.Mutex
	index        map[identityKey]int // Access using 'mu'.
	rows         []map[string]any    // Access using 'mu'.
	metricsCount int
	timer        *time.Timer // Access using 'mu'.

	close struct {
		ctx    context.Context
		cancel context.CancelFunc
		atomic.Bool
		sync.WaitGroup
	}
}

// newBatchIdentityWriter returns an identity writer for writing identities in
// batch, relative to the given pipeline (which must be running) on the data
// warehouse. purge reports whether identities should be purged from the data
// warehouse after all identities have been written.
//
// If the pipeline's output schema does not align with the profile schema, it
// returns a *schemas.Error error.
func newBatchIdentityWriter(store *Store, pipeline *state.Pipeline, purge bool) (*BatchIdentityWriter, error) {

	connection := pipeline.Connection()
	run, ok := pipeline.Run()
	if !ok {
		return nil, fmt.Errorf("pipeline is not running")
	}

	// Check that pipeline's output schema is aligned with the profile schema.
	workspace := connection.Workspace()
	err := schemas.CheckAlignment(pipeline.OutSchema, workspace.ProfileSchema, nil)
	if err != nil {
		return nil, err
	}
	iw := BatchIdentityWriter{
		store:      store,
		pipeline:   pipeline.ID,
		connection: connection.ID,
		run:        run.ID,
		flatter:    newFlatter(pipeline.OutSchema, store.identityColumnByProperty()),
		index:      map[identityKey]int{},
		purge:      purge,
	}
	iw.close.ctx, iw.close.cancel = context.WithCancel(context.Background())

	iw.columns = make([]warehouses.Column, 7, 7+len(pipeline.Transformation.OutPaths))
	iw.columns[0] = warehouses.Column{Name: "_pipeline", Type: types.Int(32)}
	iw.columns[1] = warehouses.Column{Name: "_is_anonymous", Type: types.Boolean()}
	iw.columns[2] = warehouses.Column{Name: "_identity_id", Type: types.String()}
	iw.columns[3] = warehouses.Column{Name: "_connection", Type: types.Int(32)}
	iw.columns[4] = warehouses.Column{Name: "_anonymous_ids", Type: types.Array(types.String()), Nullable: true}
	iw.columns[5] = warehouses.Column{Name: "_updated_at", Type: types.DateTime()}
	iw.columns[6] = warehouses.Column{Name: "_run", Type: types.Int(32), Nullable: true}
	iw.columns = appendColumnsFromProperties(iw.columns, pipeline.Transformation.OutPaths, store.profileColumnByProperty())

	return &iw, nil
}

// Cancel cancels the writer, ensuring that the ongoing write operations are
// completed, while discarding the pending ones, and skipping the purge
// operation if was required.
//
// If the writer is already closed, it does nothing.
func (iw *BatchIdentityWriter) Cancel(ctx context.Context) {
	if iw.close.Swap(true) {
		return
	}
	// Cancel the flushes if the context is canceled.
	stop := context.AfterFunc(ctx, func() { iw.close.cancel() })
	defer stop()
	// Wait for the flushes to terminate.
	iw.close.Wait()
}

// Close closes the writer, ensuring the completion of all pending or ongoing
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
//
// If purge needs to be done, it purges all identities of the pipeline for which
// neither the Write method nor the Keep method has been called. If Keep was
// called with an empty identifier, the purge is skipped and ErrPurgeSkipped is
// returned.
//
// If the writer is already closed, it does nothing and returns immediately.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a
// *datastore.UnavailableError error.
//
// TODO(Gianluca): if these errors returned from Close seem strange, it's
// because we still need to discuss the issue
// https://github.com/meergo/meergo/issues/1224 and understand precisely what
// model we want to implement for the operations and compatible methods.
func (iw *BatchIdentityWriter) Close(ctx context.Context) error {
	if iw.close.Load() {
		return nil
	}
	ctx, done, err := iw.store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	// Mark as closed and return if it was already closed in the meantime.
	if iw.close.Swap(true) {
		return nil
	}
	// Cancel the flushes if the context is canceled.
	stop := context.AfterFunc(ctx, func() { iw.close.cancel() })
	defer stop()
	// Wait for the flushes to terminate.
	iw.close.Wait()
	// Perform a final flush.
	iw.mu.Lock()
	iw.flush()
	iw.mu.Unlock()
	iw.close.Wait()
	// Purge identities.
	if iw.purge {
		if err := ctx.Err(); err != nil {
			return err
		}
		if iw.skipPurge {
			return ErrPurgeSkipped
		}
		where := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
			warehouses.NewBaseExpr(warehouses.Column{Name: "_pipeline", Type: types.Int(32)}, warehouses.OpIs, iw.pipeline),
			warehouses.NewBaseExpr(warehouses.Column{Name: "_run", Type: types.Int(32)}, warehouses.OpIsNot, iw.run),
		})
		err := iw.store.warehouse().Delete(ctx, "meergo_identities", where)
		if err != nil {
			return err
		}
	}
	return nil
}

// Keep keeps the identity with the identifier id. Use Keep instead of Write
// when there is no need to modify the identity, but to ensure it is not purged
// in case of reload.
//
// If id is empty, the purge will be skipped because it would be impossible to
// determine which unimported records failed due to an error.
func (iw *BatchIdentityWriter) Keep(id string) {
	if iw.close.Load() {
		panic("call Keep on a closed identity writer")
	}
	if !iw.purge || iw.skipPurge {
		return
	}
	if id == "" {
		iw.skipPurge = true
		return
	}
	key := identityKey{pipeline: iw.pipeline, identityID: id}
	row := map[string]any{
		"$purge":        false,
		"_pipeline":     key.pipeline,
		"_is_anonymous": false,
		"_identity_id":  key.identityID,
		"_connection":   iw.connection,
		"_run":          iw.run,
	}
	iw.appendRow(key, row, false)
}

// Write writes an identity. If a valid profile schema has been provided, the
// attributes must comply with it. It returns immediately, deferring the
// validation of the attributes and the actual write operation to a later time.
//
// It panics if called on a closed writer.
func (iw *BatchIdentityWriter) Write(identity Identity) {
	if iw.close.Load() {
		panic("call Write on a closed identity writer")
	}
	key := identityKey{pipeline: iw.pipeline, identityID: identity.ID}
	row := identity.Attributes
	iw.flatter.flat(row)
	row["_pipeline"] = key.pipeline
	row["_is_anonymous"] = false
	row["_identity_id"] = key.identityID
	row["_connection"] = iw.connection
	row["_updated_at"] = identity.UpdatedAt
	row["_run"] = iw.run
	iw.appendRow(key, row, true)
}

// appendRow appends a row to the rows or replaces an existing row with the same key.
func (iw *BatchIdentityWriter) appendRow(key identityKey, row map[string]any, includeInMetrics bool) {
	iw.mu.Lock()
	if includeInMetrics {
		iw.metricsCount++
	}
	// If a row with the same key already exists, update that row rather than adding a duplicate.
	if i, ok := iw.index[key]; ok {
		iw.rows[i] = row
		iw.mu.Unlock()
		return
	}
	if len(iw.rows) == 0 {
		iw.timer = time.AfterFunc(maxQueuedIdentityTime, func() {
			iw.mu.Lock()
			iw.flush()
			iw.mu.Unlock()
		})
	}
	iw.index[key] = len(iw.rows)
	iw.rows = append(iw.rows, row)
	if len(iw.rows) == maxQueuedIdentityRows {
		iw.flush()
	}
	iw.mu.Unlock()
}

// flush flushes the rows, if any, into the data warehouse.
// It must be called while holding the iw.mu mutex.
func (iw *BatchIdentityWriter) flush() {
	metrics.Increment("BatchIdentityWriter.flush.calls", 1)
	if iw.rows == nil {
		return
	}
	rows := iw.rows
	iw.rows = nil
	count := iw.metricsCount
	iw.metricsCount = 0
	clear(iw.index)
	iw.timer = nil
	iw.close.Go(func() {
		err := iw.store.warehouse().MergeIdentities(iw.close.ctx, iw.columns, rows)
		if err != nil {
			iw.store.ds.metrics.FinalizeFailed(iw.pipeline, count, err.Error())
			return
		}
		iw.store.ds.metrics.FinalizePassed(iw.pipeline, count)
	})
}
