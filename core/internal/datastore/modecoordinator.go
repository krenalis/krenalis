// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"
	"sync"

	"github.com/krenalis/krenalis/core/internal/state"
)

// newModeCoordinator returns a new modeCoordinator, which coordinates the
// changes in the warehouse mode of a workspace and its operations.
// initialMode is the mode with which the modeCoordinator is initialized.
func newModeCoordinator(initialMode state.WarehouseMode) *modeCoordinator {
	mc := &modeCoordinator{
		currentMode:  initialMode,
		changingTo:   initialMode,
		operations:   map[*operation]struct{}{},
		incompatible: [3]sync.WaitGroup{{}, {}, {}},
	}
	mc.change = sync.Cond{L: &mc.mu}
	return mc
}

// modeCoordinator coordinates the changes in the warehouse mode of a workspace
// and its operations.
type modeCoordinator struct {
	change       sync.Cond // for waiting the changing from one mode to another.
	mu           sync.Mutex
	currentMode  state.WarehouseMode     // access using 'mu'.
	changingTo   state.WarehouseMode     // access using 'mu'. Different from 'currentMode' only during mode changing.
	operations   map[*operation]struct{} // access using 'mu'.
	incompatible [3]sync.WaitGroup       // keeps track of operations incompatible with a certain mode.
}

// operation represents an operation in the modeCoordinator. Used internally by
// it, not meant to be used outside the modeCoordinator.
type operation struct {
	modes  allowedMode
	cancel func()
}

// ChangeMode changes the warehouse mode of the modeCoordinator to mode.
//
// If cancelIncompatibleOperations is true, then, during the transition to the
// mode, operations incompatible with that mode are canceled through their
// context, otherwise, the end of incompatible operations is awaited.
//
// The method returns when all operations incompatible with the new mode have
// been completed and the transition to the new mode has been finalized.
func (mc *modeCoordinator) ChangeMode(mode state.WarehouseMode, cancelIncompatibleOperations bool) {
	mc.mu.Lock()
	// If the mode to switch to is the same as the current one, do nothing.
	if mode == mc.currentMode {
		mc.mu.Unlock()
		return
	}
	// If a mode change is in progress, then wait for the change to be
	// completed.
	if mc.currentMode != mc.changingTo {
		mc.change.Wait()
		// Check again if the current mode (which has now changed because the
		// mode has changed) is equal to the one to switch to, returning in that
		// case.
		if mode == mc.currentMode {
			mc.mu.Unlock()
			return
		}
	}
	mc.changingTo = mode
	if cancelIncompatibleOperations {
		for op := range mc.operations {
			if !compatibleMode(op.modes, mode) {
				op.cancel()
			}
		}
	}
	mc.mu.Unlock()
	mc.incompatible[mode].Wait()
	mc.mu.Lock()
	mc.currentMode = mode
	mc.mu.Unlock()
	mc.change.Broadcast()
}

// Mode returns the data warehouse mode.
func (mc *modeCoordinator) Mode() state.WarehouseMode {
	mc.mu.Lock()
	mode := mc.currentMode
	mc.mu.Unlock()
	return mode
}

// allowedMode represents the mode in which an operation can operate.
type allowedMode uint8

const (
	normalMode      allowedMode = 1 << iota // a flag representing the Normal warehouse mode.
	inspectionMode                          // a flag representing the Inspection warehouse mode.
	maintenanceMode                         // a flag representing the Maintenance warehouse mode.
)
const anyMode allowedMode = 0xFF // a flag representing every warehouse mode.

var warehouseModes = [...]state.WarehouseMode{
	state.Normal,
	state.Inspection,
	state.Maintenance,
}

// StartOperation starts an operation, which is compatible with the given
// warehouse modes.
//
// Returns a context, instantiated from ctx, which should be used by the caller
// to perform the operation, and a 'done' function which should be called by the
// caller when the operation is completed.
//
// If the operation is not compatibile with the current warehouse mode, one of
// ErrNormalMode, ErrInspectionMode or ErrMaintenanceMode error is returned,
// depending on the current mode.
func (mc *modeCoordinator) StartOperation(ctx context.Context, modes allowedMode) (newCtx context.Context, done func(), err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	// If the requested operation is not compatibile with the current mode,
	// return error.
	if !compatibleMode(modes, mc.currentMode) {
		return nil, nil, modeError(mc.currentMode)
	}
	// If the requested operation is compatible with the current mode but not
	// with the future mode to which it is changing, then wait for the mode
	// change to be completed and then return an error for incompatibility with
	// the future mode.
	if !compatibleMode(modes, mc.changingTo) {
		mc.change.Wait()
		return nil, nil, modeError(mc.changingTo)
	}
	newCtx, cancel := context.WithCancel(ctx)
	op := &operation{
		modes:  modes,
		cancel: cancel,
	}
	for _, mode := range warehouseModes {
		if !compatibleMode(modes, mode) {
			mc.incompatible[mode].Add(1)
		}
	}
	mc.operations[op] = struct{}{}
	done = func() {
		for _, mode := range warehouseModes {
			if !compatibleMode(modes, mode) {
				mc.incompatible[mode].Done()
			}
		}
		mc.mu.Lock()
		delete(mc.operations, op)
		mc.mu.Unlock()
	}
	return newCtx, done, nil
}

// compatibleMode reports whether mode is compatible with one of the allowed
// modes.
func compatibleMode(allowedModes allowedMode, mode state.WarehouseMode) bool {
	return allowedModes&(1<<mode) != 0
}

// modeError returns the error (ErrNormalMode, ErrInspectionMode, etc...)
// corresponding to the specified mode.
func modeError(mode state.WarehouseMode) error {
	switch mode {
	case state.Normal:
		return ErrNormalMode
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	default:
		panic(fmt.Sprintf("unexpected mode %d", mode))
	}
}
