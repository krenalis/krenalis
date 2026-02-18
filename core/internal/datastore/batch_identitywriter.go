// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

var ErrPipelineNotExist = errors.New("pipeline does not exist")

// ErrPurgeSkipped is returned when the purge phase is skipped because some
// records without known IDs encountered an error.
var ErrPurgeSkipped = errors.New("purge skipped")

// Identity is an identity
type Identity struct {
	ID          string         // Identifier of the identity; it is empty for anonymous identities.
	AnonymousID string         // AnonymousID of identities received via events.
	Attributes  map[string]any // Attributes. Keys are profile schema's properties.
	UpdatedAt   time.Time      // Update time in UTC.
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
	workspace  int
	run        int
	flatter    *flatter
	columns    []warehouses.Column
	identities chan<- flusherRow[map[string]any]
	flusher    *flusher[map[string]any]
	purge      bool
	skipPurge  bool

	closed atomic.Bool
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
	w := BatchIdentityWriter{
		store:      store,
		pipeline:   pipeline.ID,
		connection: connection.ID,
		workspace:  workspace.ID,
		run:        run.ID,
		flatter:    newFlatter(pipeline.OutSchema, store.identityColumnByProperty()),
		purge:      purge,
	}

	w.columns = make([]warehouses.Column, 7, 7+len(pipeline.Transformation.OutPaths))
	w.columns[0] = warehouses.Column{Name: "_pipeline", Type: types.Int(32)}
	w.columns[1] = warehouses.Column{Name: "_is_anonymous", Type: types.Boolean()}
	w.columns[2] = warehouses.Column{Name: "_identity_id", Type: types.String()}
	w.columns[3] = warehouses.Column{Name: "_connection", Type: types.Int(32)}
	w.columns[4] = warehouses.Column{Name: "_anonymous_ids", Type: types.Array(types.String()), Nullable: true}
	w.columns[5] = warehouses.Column{Name: "_updated_at", Type: types.DateTime()}
	w.columns[6] = warehouses.Column{Name: "_run", Type: types.Int(32), Nullable: true}
	w.columns = appendColumnsFromProperties(w.columns, pipeline.Transformation.OutPaths, store.profileColumnByProperty())

	// Start the flusher.
	opts := flusherOptions{
		QueueSize:        32768,
		BatchSize:        20000,
		MaxBatchSize:     100000,
		MinFlushInterval: 500 * time.Millisecond,
		MaxFlushLatency:  15 * time.Second,
		IdleFlushDelay:   2 * time.Second,
		RateAlpha:        0.3,
		MetricsFinalizer: store.ds.metrics.FinalizePassed,
		LogError:         w.logError,
	}
	w.flusher = newFlusher(opts, store.mc.StartOperation, func(ctx context.Context, identities []map[string]any) error {
		return store.warehouse().MergeIdentities(ctx, w.columns, identities)
	})
	w.identities = w.flusher.Ch()

	return &w, nil
}

// Close closes the writer, ensuring the completion of all flushed
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
//
// If purge needs to be done, it purges all identities of the pipeline for which
// neither the Write method nor the Keep method has been called. If Keep was
// called with an empty identifier, the purge is skipped and ErrPurgeSkipped is
// returned.
//
// When Close is called, no other calls to EventIdentityWriter's methods should
// be in progress and no other shall be made.
//
// After Close is called, subsequent calls to Close or Stop do nothing.
func (w *BatchIdentityWriter) Close(ctx context.Context) error {
	if w.closed.Swap(true) {
		return nil
	}
	// Close the flusher.
	err := w.flusher.Close(ctx)
	// Purge identities.
	if err == nil && w.purge {
		if w.skipPurge {
			return ErrPurgeSkipped
		}
		where := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
			warehouses.NewBaseExpr(warehouses.Column{Name: "_pipeline", Type: types.Int(32)}, warehouses.OpIs, w.pipeline),
			warehouses.NewBaseExpr(warehouses.Column{Name: "_run", Type: types.Int(32)}, warehouses.OpIsNot, w.run),
		})
		err = w.store.warehouse().Delete(ctx, "meergo_identities", where)
	}
	return err
}

// Keep keeps the identity with the identifier id. Use Keep instead of Write
// when there is no need to modify the identity, but to ensure it is not purged
// in case of reload.
//
// If id is empty, the purge will be skipped because it would be impossible to
// determine which unimported records failed due to an error.
func (w *BatchIdentityWriter) Keep(ctx context.Context, id string) error {
	if !w.purge || w.skipPurge {
		return nil
	}
	if id == "" {
		w.skipPurge = true
		return nil
	}
	key := identityKey{pipeline: w.pipeline, identityID: id}
	row := map[string]any{
		"$purge":        false,
		"_pipeline":     key.pipeline,
		"_is_anonymous": false,
		"_identity_id":  key.identityID,
		"_connection":   w.connection,
		"_run":          w.run,
	}
	select {
	case w.identities <- flusherRow[map[string]any]{key: key, pipeline: w.pipeline, row: row}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops the writer, ensuring that the ongoing write operations are
// completed, while discarding the pending ones, and skipping the purge
// operation if was required.
//
// When Stop is called, no other calls to EventIdentityWriter's methods should
// be in progress and no other shall be made.
//
// After Stop is called, subsequent calls to Stop or Close do nothing.
func (w *BatchIdentityWriter) Stop(ctx context.Context) error {
	if w.closed.Swap(true) {
		return nil
	}
	// Stop the flusher.
	return w.flusher.Stop(ctx)
}

// Write writes an identity. If a valid profile schema has been provided, the
// attributes must comply with it. It returns immediately, deferring the
// validation of the attributes and the actual write operation to a later time.
func (w *BatchIdentityWriter) Write(ctx context.Context, identity Identity) error {
	key := identityKey{pipeline: w.pipeline, identityID: identity.ID}
	row := identity.Attributes
	w.flatter.flat(row)
	row["_pipeline"] = key.pipeline
	row["_is_anonymous"] = false
	row["_identity_id"] = key.identityID
	row["_connection"] = w.connection
	row["_updated_at"] = identity.UpdatedAt
	row["_run"] = w.run
	select {
	case w.identities <- flusherRow[map[string]any]{key: key, pipeline: w.pipeline, row: row}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// logError logs an error that occurred while flushing the identities.
func (w *BatchIdentityWriter) logError(err error) {
	slog.Warn("cannot flush batch identities to the data warehouse; retrying.", "workspace", w.workspace, "connection", w.connection, "pipeline", w.pipeline, "error", err)
}
