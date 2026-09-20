// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"unicode/utf8"
)

var sourceCSVHeader = []string{"source_record_id", "first_name", "last_name", "email", "phone", "photo_url", "country"}

// SourceCSVWriter serializes already generated ordinary records incrementally.
// It writes a header, comma-separated UTF-8 fields, and LF line endings.
// Absent fields become empty cells, matching the CSV connector's nil encoding.
type SourceCSVWriter struct {
	writer  *csv.Writer
	catalog *FaceCatalog
}

// NewSourceCSVWriter starts one CSV output with its fixed header.
func NewSourceCSVWriter(dst io.Writer, catalog *FaceCatalog) (*SourceCSVWriter, error) {

	if dst == nil || catalog == nil {
		return nil, fmt.Errorf("invalid CSV destination or photo catalog")
	}

	writer := csv.NewWriter(dst)
	err := writer.Write(sourceCSVHeader)
	if err != nil {
		return nil, err
	}

	return &SourceCSVWriter{writer: writer, catalog: catalog}, nil
}

// Flush sends buffered CSV bytes to the destination and reports writer errors.
func (w *SourceCSVWriter) Flush(ctx context.Context) error {
	if w == nil || w.writer == nil || ctx == nil {
		return fmt.Errorf("invalid CSV writer or context")
	}
	w.writer.Flush()
	err := w.writer.Error()
	if err != nil {
		return err
	}
	return ctx.Err()
}

// Write serializes one supplied record without making generation decisions.
// Call Flush to report errors that remain buffered by encoding/csv.
func (w *SourceCSVWriter) Write(ctx context.Context, record SourceRecord) error {

	if w == nil || w.writer == nil || ctx == nil || len(record.ID) == 0 || len(record.ID) > 128 ||
		!utf8.ValidString(record.ID) {
		return fmt.Errorf("invalid CSV record or context")
	}

	fields := [7]string{record.ID}
	for i, field := range [...]*string{record.FirstName, record.LastName, record.Email, record.Phone} {
		if field != nil {
			if len(*field) > 1024 || !utf8.ValidString(*field) {
				return fmt.Errorf("invalid CSV observation")
			}
			fields[i+1] = *field
		}
	}
	if record.PhotoID != nil {
		if len(*record.PhotoID) > 128 {
			return fmt.Errorf("invalid CSV photo reference")
		}
		asset, err := w.catalog.Asset(*record.PhotoID, PhotoSize256)
		if err != nil {
			return fmt.Errorf("resolve CSV photo asset: %w", err)
		}
		fields[5] = asset.Path
	}
	if record.Country != nil {
		if len(*record.Country) > 1024 || !utf8.ValidString(*record.Country) {
			return fmt.Errorf("invalid CSV observation")
		}
		fields[6] = *record.Country
	}

	err := ctx.Err()
	if err != nil {
		return err
	}

	return w.writer.Write(fields[:])
}
