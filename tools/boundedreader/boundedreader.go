// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package boundedreader provides a Reader that fails when its input exceeds
// a configured size limit.
//
// Unlike io.LimitedReader, which behaves as if the underlying stream ended
// at the limit, a BoundedReader reports an error when additional data is
// encountered beyond the limit.
//
// A BoundedReader never returns bytes beyond the configured limit.
package boundedreader

import (
	"errors"
	"io"
)

type BoundedReader struct {
	r    io.Reader
	n    int
	errf func() error
	err  error
}

var ErrTooLarge = errors.New("data is too large")

// New returns a BoundedReader that reads from r and enforces a limit of n
// bytes.
func New(r io.Reader, n int) *BoundedReader {
	if n < 0 {
		panic("n must be >= 0")
	}
	return &BoundedReader{r: r, n: n}
}

// SetErrorFunc sets the error reported when a read would exceed the
// configured size limit. The error is obtained by calling f. If f is nil,
// ErrTooLarge is reported.
func (br *BoundedReader) SetErrorFunc(f func() error) {
	br.errf = f
}

// Read implements io.Reader.
func (br *BoundedReader) Read(p []byte) (n int, err error) {
	if br.err != nil {
		return 0, br.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) > br.n {
		p = p[:br.n+1]
	}
	n, err = br.r.Read(p)
	if n > br.n {
		br.n = 0
		if br.errf != nil {
			br.err = br.errf()
		} else {
			br.err = ErrTooLarge
		}
		return n - 1, br.err
	}
	br.n -= n
	return n, err
}
