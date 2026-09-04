// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

// Layer 0 error categories.
const (
	ErrAllZeroWeights                foundationError = "all-zero weights"
	ErrComponentCountOverflow        foundationError = "component-count overflow"
	ErrDuplicateComponent            foundationError = "duplicate component"
	ErrEmptyWeightedTable            foundationError = "empty weighted table"
	ErrInvalidComposedID             foundationError = "invalid composed ID"
	ErrInvalidDeterministicComponent foundationError = "invalid deterministic component"
	ErrInvalidIdentityNamespace      foundationError = "invalid identity namespace"
	ErrInvalidIndex                  foundationError = "invalid index"
	ErrInvalidProbability            foundationError = "invalid probability"
	ErrInvalidStreamPath             foundationError = "invalid stream path"
	ErrInvalidStreamScope            foundationError = "invalid stream scope"
	ErrMalformedSyntheticID          foundationError = "malformed synthetic ID"
	ErrRangeOverflow                 foundationError = "range overflow"
	ErrTotalWeightOverflow           foundationError = "total-weight overflow"
	ErrUnassignedSyntheticID         foundationError = "unassigned synthetic ID"
	ErrUnsupportedSyntheticIDVersion foundationError = "unsupported synthetic-ID version"
	ErrWrongEntityKind               foundationError = "wrong entity kind"
	ErrWrongIdentityNamespace        foundationError = "wrong identity namespace"
	ErrZeroBound                     foundationError = "zero bound"
)

// foundationError identifies an error category in the deterministic foundation layer.
type foundationError string

// Error returns the error category's description.
func (e foundationError) Error() string {
	return string(e)
}
