// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

const (
	usaNamesMass = uint64(1<<32 - 1)
)

// usaNameRow owns a scalar copy of an admitted international NameEntry.
type usaNameRow struct {
	display   string
	asciiAtom string
	weight    uint64
}
