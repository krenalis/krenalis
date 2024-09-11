//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package warehouses

import (
	"fmt"
	"strings"

	"github.com/meergo/meergo/apis/errors"
)

// DataWarehouseError represents an error with the data warehouse.
type DataWarehouseError struct {
	Err error
}

func (e *DataWarehouseError) Error() string {
	return fmt.Sprintf("data warehouse error: %s", e.Err)
}

// Error returns a new *DataWarehouseError error.
func Error(err error) error {
	return &DataWarehouseError{Err: err}
}

// Errorf returns a new *DataWarehouseError error with a
// fmt.Errorf(format, a...) error.
func Errorf(format string, a ...any) error {
	return &DataWarehouseError{Err: fmt.Errorf(format, a...)}
}

// SettingsError represents a syntax error in the data warehouse settings.
type SettingsError struct {
	Err error
}

func (e *SettingsError) Error() string {
	return fmt.Sprintf("settings error: %s", e.Err)
}

// SettingsErrorf returns a new SettingsError error with a
// fmt.Errorf(format, a...) error.
func SettingsErrorf(format string, a ...any) error {
	return &SettingsError{Err: fmt.Errorf(format, a...)}
}

// ErrDataWarehouseNotInitialized is the error indicating that the data
// warehouse is not initialized.
var ErrDataWarehouseNotInitialized = errors.New("data warehouse not initialized")

// DataWarehouseNeedsRepairError represents an error indicating that the data
// warehouse needs to be repaired.
type DataWarehouseNeedsRepairError struct {
	toRepair []string
}

// NewDataWarehouseNeedsRepairError returns a new DataWarehouseNeedsRepairError
// error.
func NewDataWarehouseNeedsRepairError(toRepair []string) error {
	return &DataWarehouseNeedsRepairError{toRepair: toRepair}
}

func (e *DataWarehouseNeedsRepairError) Error() string {
	return fmt.Sprintf("data warehouse needs repairing: %s", strings.Join(e.toRepair, "; "))
}
