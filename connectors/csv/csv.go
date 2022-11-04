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
	"unicode/utf8"

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

// Read reads the records from r and calls put for each record read.
func (c *connection) Read(r io.Reader, put func(record []string) error) error {
	v := csv.NewReader(r)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	if c.settings.Comment != "" {
		v.Comment, _ = utf8.DecodeRuneInString(c.settings.Comment)
	}
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
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
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

	var s settings

	if event == "save" {
		// Save the settings.
		err := json.Unmarshal(form, &s)
		if err != nil {
			return nil, err
		}
		// Validate Comma.
		if utf8.RuneCountInString(s.Comma) != 1 {
			return nil, connector.UIErrorf("comma must be a single character")
		}
		if c := s.Comma; c == "\n" || c == "\r" || c == "\uFFFD" {
			return nil, connector.UIErrorf("comma cannot be \\r, \\n, or the Unicode replacement character")
		}
		// Validate Comment.
		if c := s.Comment; c != "" {
			if utf8.RuneCountInString(c) != 1 {
				return nil, connector.UIErrorf("comment, if provided, must be a single character")
			}
			if c == "\n" || c == "\r" || c == "\uFFFD" {
				return nil, connector.UIErrorf("comment cannot be \\r, \\n, or the Unicode replacement character")
			}
			if c == s.Comma {
				return nil, connector.UIErrorf("comment cannot be equal to the comma")
			}
		}
		// Validate FieldsPerRecord.
		if f := s.FieldsPerRecord; f < 0 || f > 1000 {
			return nil, connector.UIErrorf("fields per record, if provided, must be in range [0,1000]")
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	}

	if c.settings != nil {
		s = *c.settings
	}

	ui := &connector.SettingsUI{
		Components: []connector.Component{
			&connector.Input{Name: "comma", Value: s.Comma, Label: "Comma", Placeholder: ",", Type: "text", MinLength: 1, MaxLength: 1},
			&connector.Input{Name: "comment", Value: s.Comment, Label: "Comment", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 1},
			&connector.Input{Name: "fieldsPerRecord", Value: s.FieldsPerRecord, Label: "Fields per record", Placeholder: "", Type: "number"},
			&connector.Checkbox{Name: "trimLeadingSpace", Value: s.TrimLeadingSpace, Label: "Trim leading space"},
			&connector.Checkbox{Name: "useCRLF", Value: s.UseCRLF, Label: "Use CRLF"},
		},
		Actions: []connector.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return ui, nil

}
