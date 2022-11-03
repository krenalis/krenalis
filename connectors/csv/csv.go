//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package csv

// This package is the CSV connector.
// (https://www.ietf.org/rfc/rfc4180.txt)

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"

	"chichi/apis"
	"chichi/connector"
)

// Make sure it implements the FileConnector interface.
var _ connector.FileConnection = &connection{}

func init() {
	apis.RegisterFileConnector("CSV", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
}

type settings struct {
	Comma            rune
	Comment          rune
	FieldsPerRecord  int
	LazyQuotes       bool
	TrimLeadingSpace bool
	UseCRLF          bool
}

// New returns a new CSV connection.
func New(ctx context.Context, settings []byte, fh connector.Firehose) (connector.FileConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connection")
		}
	}
	return &c, nil
}

// Read reads the records from r and calls put for each record read.
func (c *connection) Read(r io.Reader, put func(record []string) error) error {
	v := csv.NewReader(r)
	v.Comma = c.settings.Comma
	v.Comment = c.settings.Comment
	v.FieldsPerRecord = c.settings.FieldsPerRecord
	v.LazyQuotes = c.settings.LazyQuotes
	v.TrimLeadingSpace = c.settings.TrimLeadingSpace
	for {
		record, err := v.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = put(record)
		if err != nil {
			return err
		}
	}
	return nil
}

// Write writes the records read from get into w.
func (c *connection) Write(w io.Writer, get func() ([]string, error)) error {
	v := csv.NewWriter(w)
	v.Comma = c.settings.Comma
	v.UseCRLF = c.settings.UseCRLF
	for {
		record, err := get()
		if err == io.EOF {
			v.Flush()
			if err := v.Error(); err != nil {
				return err
			}
			break
		}
		if err != nil {
			return err
		}
		err = v.Write(record)
		if err != nil {
			return err
		}
	}
	return nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, form []byte) (*connector.SettingsUI, error) {
	return nil, nil
}
