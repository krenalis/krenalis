//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"os"
	pkgPath "path"
	"time"

	"chichi/apis/state"
	"chichi/connector"

	"github.com/golang/snappy"
)

// errReadStopped is returned to a file connector when it calls w.Write and the
// storage has already finished reading without an error.
// If this error occurs, it indicates a bug in the storage connector.
var errReadStopped = errors.New("storage abruptly stopped reading")

// compressorStorage implements a storage capable of compressing and
// decompressing data read from or written to a connector.StorageConnection.
type compressorStorage struct {
	storage     connector.StorageConnection
	compression state.Compression
}

// newCompressedStorage returns a compressor storage that wraps s and performs
// file compression and decompression using c as the compression method.
// If c is NoCompression, it does not perform any compression or decompression.
func newCompressedStorage(s connector.StorageConnection, c state.Compression) *compressorStorage {
	return &compressorStorage{s, c}
}

// Open opens the file at the given path and returns a ReadCloser from which to
// read the file and its last update time.
// It is the caller's responsibility to close the returned reader.
func (cs compressorStorage) Open(path string) (io.ReadCloser, time.Time, error) {
	originalPath := path
	ext := cs.compression.Ext()
	path += ext
	r, t, err := cs.storage.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	switch cs.compression {
	case state.ZipCompression:
		var err error
		var fi *os.File
		defer func() {
			if err != nil {
				if fi != nil {
					_ = closeTempFile(fi)
				}
				_ = r.Close()
			}
		}()
		fi, err = os.CreateTemp("", "")
		if err != nil {
			return nil, time.Time{}, err
		}
		st, err := fi.Stat()
		if err != nil {
			return nil, time.Time{}, err
		}
		z, err := zip.NewReader(fi, st.Size())
		if err != nil {
			return nil, time.Time{}, err
		}
		name := pkgPath.Base(originalPath)
		r3, err := z.Open(name)
		if err != nil {
			return nil, time.Time{}, err
		}
		r = newFuncReadCloser(r3, func() error {
			err3 := r3.Close()
			err2 := closeTempFile(fi)
			err := r.Close()
			if err3 != nil {
				return err3
			}
			if err2 != nil {
				return err2
			}
			return err
		})
	case state.GzipCompression:
		r2, err := gzip.NewReader(r)
		if err != nil {
			_ = r.Close()
			return nil, time.Time{}, err
		}
		r = newFuncReadCloser(r2, func() error {
			err2 := r2.Close()
			err := r.Close()
			if err2 != nil {
				return err2
			}
			return err
		})
		r = r2
	case state.SnappyCompression:
		r2 := snappy.NewReader(r)
		r = newFuncReadCloser(r2, func() error {
			return r.Close()
		})
	}
	return r, t, nil
}

// Writer returns a Writer that compress the data if needed, and then writes it
// directly to the underlying storage.
//
// If the data should be compressed, it passes path to the underlying storage
// with an appended extension, and an appropriate content type.
//
// It is the caller's responsibility to call Close on the returned Writer.
func (cs compressorStorage) Writer(path, contentType string) (*storageWriteCloser, error) {
	pr, pw := io.Pipe()
	var w io.WriteCloser
	switch cs.compression {
	case state.NoCompression:
		w = pw
	case state.ZipCompression:
		z := zip.NewWriter(pw)
		name := pkgPath.Base(path)
		zw, err := z.Create(name)
		if err != nil {
			_ = z.Close()
			_ = pr.Close()
			_ = pw.Close()
			return nil, err
		}
		w = zipWriter{Writer: zw, z: z}
	case state.GzipCompression:
		w = gzip.NewWriter(pw)
	case state.SnappyCompression:
		w = snappy.NewBufferedWriter(pw)
	}
	path += cs.compression.Ext()
	if ct := cs.compression.ContentType(); ct != "" {
		contentType = ct
	}
	ch := make(chan error)
	go func() {
		err := cs.storage.Write(pr, path, contentType)
		if err != nil {
			_ = pr.CloseWithError(err)
		} else {
			// errReadStopped will be returned to the file connector only if it
			// calls w.Write when the storage is returned.
			_ = pr.CloseWithError(errReadStopped)
		}
		ch <- err
	}()
	wc := newFuncWriteCloser(w, func(err error) error {
		if w != pw {
			err2 := w.Close()
			if err == nil {
				err = err2
			}
		}
		_ = pw.CloseWithError(err)
		return <-ch
	})
	return wc, nil
}

// closeTempFile closes fi and returns the error, if any. Any error encountered
// will be logged.
func closeTempFile(fi *os.File) error {
	err := fi.Close()
	if err := os.Remove(fi.Name()); err != nil {
		log.Printf("[warning] cannot remove temporary file %q: %s", fi.Name(), err)
	}
	return err
}

// funcReadCloser wraps an io.Reader and implements io.ReadCloser. It calls a
// specified function when Close is invoked.
type funcReadCloser struct {
	io.Reader
	close func() error
}

// newFuncReadCloser returns an io.ReadCloser that wraps r and calls close when
// Close is invoked.
func newFuncReadCloser(r io.Reader, close func() error) io.ReadCloser {
	return funcReadCloser{r, close}
}

func (c funcReadCloser) Close() error {
	return c.close()
}

// storageWriteCloser wraps an io.Writer and implements io.WriteCloser. It calls a
// specified function when Close is invoked.
type storageWriteCloser struct {
	io.Writer
	close func(err error) error
}

// newFuncWriteCloser returns an io.WriteCloser that wraps w and calls close
// when Close is invoked.
func newFuncWriteCloser(w io.Writer, close func(err error) error) *storageWriteCloser {
	return &storageWriteCloser{w, close}
}

// Close closes the underlying writer. Storage will receive io.EOF from a read.
// It returns the error returned by the storage if any.
func (c storageWriteCloser) Close() error {
	return c.close(nil)
}

// CloseWithError closes the underlying writer. Storage will receive err as
// error from a read, or io.EOF is err is nil.
// It returns the error returned by the storage if any.
func (c storageWriteCloser) CloseWithError(err error) error {
	return c.close(err)
}

// zipWriter wraps a Writer and implements the Close method that closes a
// zip.Writer when called.
type zipWriter struct {
	z *zip.Writer
	io.Writer
}

func (zw zipWriter) Close() error {
	return zw.z.Close()
}
