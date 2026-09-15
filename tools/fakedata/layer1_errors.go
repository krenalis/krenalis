// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

// Layer 1 error categories.
const (
	ErrCorruptFrozenDataset  layer1Error = "corrupt frozen dataset"
	ErrInvalidFaceCatalog    layer1Error = "invalid face catalog"
	ErrInvalidPhotoAsset     layer1Error = "invalid photo asset"
	ErrInvalidReferenceDate  layer1Error = "invalid reference date"
	ErrLayer1IndexOutOfRange layer1Error = "layer 1 index out of range"
	ErrNoCompatiblePhoto     layer1Error = "no compatible photo"
)

// layer1Error identifies an error category in the Layer 1 generator.
type layer1Error string

// Error returns the error category's description.
func (e layer1Error) Error() string {
	return string(e)
}
