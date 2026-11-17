// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
)

func TestCompatibleMode(t *testing.T) {
	tests := []struct {
		modes    allowedMode
		mode     state.WarehouseMode
		expected bool
	}{
		{normalMode, state.Normal, true},
		{normalMode, state.Inspection, false},
		{maintenanceMode, state.Normal, false},
		{inspectionMode, state.Inspection, true},
		{maintenanceMode, state.Maintenance, true},
		{normalMode | inspectionMode, state.Inspection, true},
		{normalMode | inspectionMode, state.Maintenance, false},
		{normalMode | maintenanceMode, state.Inspection, false},
		{normalMode | maintenanceMode, state.Maintenance, true},
		{inspectionMode | maintenanceMode, state.Normal, false},
		{inspectionMode | maintenanceMode, state.Inspection, true},
		{normalMode | inspectionMode | maintenanceMode, state.Normal, true},
		{normalMode | inspectionMode | maintenanceMode, state.Inspection, true},
		{normalMode | inspectionMode | maintenanceMode, state.Maintenance, true},
	}

	for _, test := range tests {
		got := compatibleMode(test.modes, test.mode)
		if got != test.expected {
			t.Errorf("containsMode(%v, %v) = %v; expected %v", test.modes, test.mode, got, test.expected)
		}
	}
}

// TestModeTransitions checks behavioral filter read operations during mode
// changes.
func TestModeTransitions(t *testing.T) {
	for _, test := range []struct {
		name   string
		next   state.WarehouseMode
		cancel bool
	}{
		{"inspection", state.Inspection, false}, {"inspection cancel", state.Inspection, true},
		{"maintenance wait", state.Maintenance, false}, {"maintenance cancel", state.Maintenance, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				mc := newModeCoordinator(state.Normal)
				ctx, done, err := mc.StartOperation(t.Context(), normalMode|inspectionMode)
				if err != nil {
					t.Fatalf("expected initial read admission, got %v", err)
				}
				finish := sync.OnceFunc(done)
				defer finish()
				changed := make(chan struct{})
				go func() {
					mc.ChangeMode(test.next, test.cancel)
					close(changed)
				}()
				synctest.Wait()

				if test.next == state.Inspection {
					select {
					case <-changed:
					default:
						t.Fatal("expected compatible mode transition, got pending transition")
					}
					if ctx.Err() != nil {
						t.Fatalf("expected live compatible context, got %v", ctx.Err())
					}
					_, otherDone, err := mc.StartOperation(t.Context(), normalMode|inspectionMode)
					if err != nil {
						t.Fatalf("expected inspection read admission, got %v", err)
					}
					otherDone()
					finish()
					return
				}
				select {
				case <-changed:
					t.Fatal("expected transition to await active read, got completed transition")
				default:
				}
				if test.cancel {
					err := ctx.Err()
					if err != nil {
						if !errors.Is(err, context.Canceled) {
							t.Fatalf("expected context cancellation, got %v", err)
						}
					}
					if err == nil {
						t.Fatal("expected cancellation of incompatible read, got live context")
					}
				} else if ctx.Err() != nil {
					t.Fatalf("expected live context while transition waits, got %v", ctx.Err())
				}
				if mc.Mode() != state.Normal {
					t.Fatalf("expected previous mode until read finishes, got %v", mc.Mode())
				}
				admission := make(chan error, 1)
				go func() {
					_, done, err := mc.StartOperation(t.Context(), normalMode|inspectionMode)
					if err != nil {
						admission <- err
						return
					}
					done()
					admission <- nil
				}()
				synctest.Wait()
				select {
				case err := <-admission:
					t.Fatalf("expected admission to wait for transition, got %v", err)
				default:
				}
				finish()
				synctest.Wait()
				select {
				case <-changed:
				default:
					t.Fatal("expected completed transition, got pending transition")
				}
				err = <-admission
				if err != nil {
					if !errors.Is(err, ErrMaintenanceMode) {
						t.Fatalf("expected maintenance rejection, got %v", err)
					}
				}
				if err == nil {
					t.Fatal("expected maintenance rejection, got success")
				}
				if mc.Mode() != state.Maintenance {
					t.Fatalf("expected maintenance mode, got %v", mc.Mode())
				}
				mc.ChangeMode(state.Normal, false)
				_, done, err = mc.StartOperation(t.Context(), normalMode|inspectionMode)
				if err != nil {
					t.Fatalf("expected read admission after recovery, got %v", err)
				}
				done()

			})
		})
	}
}

func TestModes(t *testing.T) {
	// Check that the number of elements in 'warehouseModes' is correct.
	if len(warehouseModes) != 3 {
		t.Fatalf("warehouseModes should have 3 elements, got %d", len(warehouseModes))
	}
	// Test 'anyMode'.
	for _, mode := range warehouseModes {
		if !compatibleMode(anyMode, mode) {
			t.Fatalf("the constant 'anyMode' should contain the mode %q, but it doesn't", mode)
		}
	}
}
