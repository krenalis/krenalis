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
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the FileConnection interface.
var _ connector.FileConnection = &connection{}

func init() {
	connector.RegisterFile(connector.File{
		Name: "CSV",
		Icon: icon,
		Open: open,
	})
}

type connection struct {
	ctx      context.Context
	role     connector.Role
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	Path             string
	Comma            string
	Comment          string
	FieldsPerRecord  int
	LazyQuotes       bool
	TrimLeadingSpace bool
	UseCRLF          bool
}

// open opens a CSV connection and returns it.
func open(ctx context.Context, conf *connector.FileConfig) (connector.FileConnection, error) {
	c := connection{ctx: ctx, role: conf.Role, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connection")
		}
	}
	return &c, nil
}

// Read reads the records from files and writes them to records.
func (c *connection) Read(files connector.FileReader, records connector.RecordWriter) error {

	r, timestamp, err := files.Reader(c.settings.Path)
	if err != nil {
		return err
	}
	defer r.Close()

	if err = records.Timestamp(timestamp); err != nil {
		return err
	}

	// Create a CSV reader.
	v := csv.NewReader(r)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	if c.settings.Comment != "" {
		v.Comment, _ = utf8.DecodeRuneInString(c.settings.Comment)
	}
	v.FieldsPerRecord = c.settings.FieldsPerRecord
	v.LazyQuotes = c.settings.LazyQuotes
	v.TrimLeadingSpace = c.settings.TrimLeadingSpace

	first := true
	for {
		// Read a record.
		record, err := v.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// Write the columns.
		if first {
			columns := make([]connector.Column, len(record))
			for i := range columns {
				columns[i].Name = record[i]
				columns[i].Type = types.Text()
			}
			err = records.Columns(columns)
			if err != nil {
				return err
			}
			first = false
		}
		// Write the record.
		err = records.RecordString(record)
		if err != nil {
			return err
		}
	}

	return nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings == nil {
			s.Comma = ","
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := c.SettingsUI(values)
		if err != nil {
			return nil, nil, err
		}
		err = c.firehose.SetSettings(s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "path", Label: "Path", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 1000},
			&ui.Input{Name: "comma", Label: "Comma", Placeholder: ",", Type: "text", MinLength: 1, MaxLength: 1},
			&ui.Input{Name: "comment", Label: "Comment", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 1, Role: ui.SourceRole},
			&ui.Input{Name: "fieldsPerRecord", Label: "Fields per record", Placeholder: "", Type: "number", Role: ui.SourceRole},
			&ui.Checkbox{Name: "trimLeadingSpace", Label: "Trim leading space", Role: ui.SourceRole},
			&ui.Checkbox{Name: "useCRLF", Label: "Use CRLF"},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// SettingsUI obtains the settings from UI values and returns them.
func (c *connection) SettingsUI(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate Path.
	if s.Path == "" {
		return nil, ui.Errorf("path cannot be empty")
	}
	if utf8.RuneCountInString(s.Path) > 1000 {
		return nil, ui.Errorf("path cannot be longer that 1000 characters")
	}
	// Validate Comma.
	if utf8.RuneCountInString(s.Comma) != 1 {
		return nil, ui.Errorf("comma must be a single character")
	}
	if c := s.Comma; c == "\n" || c == "\r" || c == "\uFFFD" {
		return nil, ui.Errorf("comma cannot be \\r, \\n, or the Unicode replacement character")
	}
	if c.role == connector.SourceRole {
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
	} else {
		s.Comment = ""
		s.FieldsPerRecord = 0
		s.TrimLeadingSpace = false
	}
	return json.Marshal(&s)
}

// Write writes to files the records read from records.
func (c *connection) Write(files connector.FileWriter, records connector.RecordReader) error {

	w, err := files.Writer(c.settings.Path, "text/csv; charset=UTF-8")
	if err != nil {
		return err
	}
	defer w.Close()

	v := csv.NewWriter(w)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	v.UseCRLF = c.settings.UseCRLF

	// Write the column names.
	columns := records.Columns()
	record := make([]string, len(columns))
	for i, c := range columns {
		record[i] = c.Name
	}
	err = v.Write(record)
	if err != nil {
		return err
	}

	// Write the records.
	for {
		record, err = records.RecordString()
		if err == io.EOF {
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

	v.Flush()
	if err := v.Error(); err != nil {
		return err
	}
	err = w.Close()

	return err
}
