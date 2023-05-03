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

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterFile(connector.File{
		Name:              "CSV",
		SourceDescription: "import users from a CSV file",
		Icon:              icon,
	}, open)
}

type connection struct {
	ctx      context.Context
	role     connector.Role
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

// open opens a CSV connection and returns it.
func open(ctx context.Context, conf *connector.FileConfig) (*connection, error) {
	c := connection{ctx: ctx, role: conf.Role, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connection")
		}
	}
	return &c, nil
}

// ContentType returns the content type of the file.
func (c *connection) ContentType() string {
	return "text/csv; charset=UTF-8"
}

// Read reads the records from r and writes them to records.
func (c *connection) Read(r io.Reader, _ string, records connector.RecordWriter) error {

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
			columns := make([]types.Property, len(record))
			for i := range columns {
				columns[i].Name = "column" + strconv.Itoa(i+1)
				columns[i].Label = record[i]
				columns[i].Type = types.Text()
			}
			err = records.Columns(columns)
			if err != nil {
				return err
			}
			first = false
			continue
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

// Write writes to w the records read from records.
func (c *connection) Write(w io.Writer, _ string, records connector.RecordReader) error {

	v := csv.NewWriter(w)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	v.UseCRLF = c.settings.UseCRLF

	// Write the column names.
	columns := records.Columns()
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

	return err
}
