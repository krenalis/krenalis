// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

// Socioeconomic error categories.
const (
	ErrCompanyIndexOutOfRange    socioeconomicError = "company index out of range"
	ErrInvalidSocioeconomicWorld socioeconomicError = "invalid socioeconomic world"
)

type socioeconomicError string

// Error returns the error category's description.
func (e socioeconomicError) Error() string {
	return string(e)
}
