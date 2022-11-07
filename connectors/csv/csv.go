//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package csv

// This package is the CSV connector.
// (https://www.ietf.org/rfc/rfc4180.txt)

import (
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"

	"chichi/apis"
	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon []byte

// Make sure it implements the FileConnector interface.
var _ connector.FileConnection = &connection{}

func init() {
	apis.RegisterFileConnector("CSV", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	Comma            string
	Comment          string
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
	c.firehose = fh
	return &c, nil
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "CSV",
		Type: connector.TypeFile,
		Icon: icon,
	}
}

// ContentType returns the content type of the data to write.
func (c *connection) ContentType() string {
	return "text/csv; charset=UTF-8"
}

// Read reads the records from r.
func (c *connection) Read(r io.Reader) error {
	v := csv.NewReader(r)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	if c.settings.Comment != "" {
		v.Comment, _ = utf8.DecodeRuneInString(c.settings.Comment)
	}
	v.FieldsPerRecord = c.settings.FieldsPerRecord
	v.LazyQuotes = c.settings.LazyQuotes
	v.TrimLeadingSpace = c.settings.TrimLeadingSpace
	var first bool
	for {
		record, err := v.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// Set the columns.
		if first {
			columns := make([]connector.Column, len(record))
			for i, c := range columns {
				c.Name = "column" + strconv.Itoa(i+1)
				c.Type = types.Text()
			}
			err = c.firehose.SetColumns(columns)
			if err != nil {
				return err
			}
			first = false
		}
		// Put the record.
		c.firehose.PutRecordString(record)
	}
	return nil
}

// Write writes the records to w.
func (c *connection) Write(w io.Writer) error {

	v := csv.NewWriter(w)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	v.UseCRLF = c.settings.UseCRLF

	// Write the column names.
	columns := c.firehose.Columns()
	record := make([]string, len(columns))
	for i, c := range columns {
		record[i] = c.Name
	}
	err := v.Write(record)
	if err != nil {
		return err
	}

	// Write the records.
	for {
		record, err = c.firehose.RecordString()
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
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, error) {

	var s settings

	switch event {
	case "load":
		// Load the Form.
		if c.settings == nil {
			s.Comma = ","
		} else {
			s = *c.settings
		}
	case "save":
		// Save the settings.
		err := json.Unmarshal(values, &s)
		if err != nil {
			return nil, err
		}
		// Validate Comma.
		if utf8.RuneCountInString(s.Comma) != 1 {
			return nil, ui.Errorf("comma must be a single character")
		}
		if c := s.Comma; c == "\n" || c == "\r" || c == "\uFFFD" {
			return nil, ui.Errorf("comma cannot be \\r, \\n, or the Unicode replacement character")
		}
		// Validate Comment.
		if c := s.Comment; c != "" {
			if utf8.RuneCountInString(c) != 1 {
				return nil, ui.Errorf("comment, if provided, must be a single character")
			}
			if c == "\n" || c == "\r" || c == "\uFFFD" {
				return nil, ui.Errorf("comment cannot be \\r, \\n, or the Unicode replacement character")
			}
			if c == s.Comma {
				return nil, ui.Errorf("comment cannot be equal to the comma")
			}
		}
		// Validate FieldsPerRecord.
		if f := s.FieldsPerRecord; f < 0 || f > 1000 {
			return nil, ui.Errorf("fields per record, if provided, must be in range [0,1000]")
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	default:
		return nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "comma", Value: s.Comma, Label: "Comma", Placeholder: ",", Type: "text", MinLength: 1, MaxLength: 1},
			&ui.Input{Name: "comment", Value: s.Comment, Label: "Comment", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 1},
			&ui.Input{Name: "fieldsPerRecord", Value: s.FieldsPerRecord, Label: "Fields per record", Placeholder: "", Type: "number"},
			&ui.Checkbox{Name: "trimLeadingSpace", Value: s.TrimLeadingSpace, Label: "Trim leading space"},
			&ui.Checkbox{Name: "useCRLF", Value: s.UseCRLF, Label: "Use CRLF"},
		},
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil
}
