// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// FenceSimulatedSystem advances persistent warehouse authority monotonically.
// It returns an *WarehouseError if the warehouse cannot establish the fence.
func (store *Store) FenceSimulatedSystem(ctx context.Context, system string, generation int64) error {

	store.mustBeOpen()
	if system == "" || generation < 0 {
		return errors.New("invalid simulated system writer authority")
	}
	ctx, done, err := store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	warehouse, ok := store.warehouse().(simulatedSystemWarehouse)
	if !ok {
		return wrapWarehouseError(errors.New("warehouse does not support simulated system writer fencing"))
	}

	return warehouse.FenceSimulatedSystem(ctx, system, generation)
}

type simulatedSystemWarehouse interface {
	FenceSimulatedSystem(context.Context, string, int64) error
	MergeSimulatedSystemRecords(context.Context, string, int64, warehouses.Table, [][]any) error
	DeleteSimulatedSystemRecords(context.Context, string, int64) error
}

type simulatedSystemWriter struct {
	system     string
	generation int64
}

type simulatedSystemWriterKey struct{}

// WithSimulatedSystemWriter carries a system's admitted warehouse generation.
// Even generations authorize writes; odd generations authorize cleanup only.
// Generation zero is reserved for Preparing generation and replay.
func WithSimulatedSystemWriter(ctx context.Context, system string, generation int64) (context.Context, error) {
	if system == "" || generation < 0 {
		return nil, errors.New("invalid simulated system writer authority")
	}
	return context.WithValue(ctx, simulatedSystemWriterKey{}, simulatedSystemWriter{system, generation}), nil
}
