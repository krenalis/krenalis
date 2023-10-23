//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package csv implements the CSV connector.
// (https://www.ietf.org/rfc/rfc4180.txt)
package csv

import (
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"

	"github.com/shopspring/decimal"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI interface.
var _ connector.UI = (*connection)(nil)

func init() {
	connector.RegisterFile(connector.File{
		Name:              "CSV",
		SourceDescription: "import users from a CSV file",
		Icon:              icon,
		Extension:         "csv",
	}, open)
}

// open opens a CSV connection and returns it.
func open(conf *connector.FileConfig) (*connection, error) {
	c := connection{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connection")
		}
	}
	return &c, nil
}

type connection struct {
	conf     *connector.FileConfig
	settings *settings
}

type settings struct {
	Comma            string
	Comment          string
	FieldsPerRecord  int
	LazyQuotes       bool
	TrimLeadingSpace bool
	UseCRLF          bool
	HasColumnNames   bool
}

// ContentType returns the content type of the file.
func (c *connection) ContentType(ctx context.Context) string {
	return "text/csv; charset=UTF-8"
}

// Read reads the records from r and writes them to records.
func (c *connection) Read(ctx context.Context, r io.Reader, _ string, records connector.RecordWriter) error {

	// Create a CSV reader.
	v := csv.NewReader(r)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	if c.settings.Comment != "" {
		v.Comment, _ = utf8.DecodeRuneInString(c.settings.Comment)
	}
	v.FieldsPerRecord = c.settings.FieldsPerRecord
	v.LazyQuotes = c.settings.LazyQuotes
	v.TrimLeadingSpace = c.settings.TrimLeadingSpace

	var nameOfHeader map[string]string

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
				if c.settings.HasColumnNames {
					header := record[i]
					name := connector.SuggestPropertyName(header)
					if name == "" {
						return fmt.Errorf("column name %q cannot be converted to a valid property name", header)
					}
					if nameOfHeader == nil {
						nameOfHeader = make(map[string]string, len(record))
					}
					for n, h := range nameOfHeader {
						if name == n {
							if header == h {
								return fmt.Errorf("column name %q is repeated", header)
							}
							return fmt.Errorf("column name %q and %q cannot be converted into two different property names", header, h)
						}
					}
					columns[i].Name = name
					if name != record[i] {
						columns[i].Label = header
					}
					nameOfHeader[header] = name
				} else {
					name := columnIndexToPropertyName(i + 1)
					columns[i].Name = name
				}
				columns[i].Type = types.Text()
			}
			err = records.Columns(columns)
			if err != nil {
				return err
			}
			first = false
			if c.settings.HasColumnNames {
				continue
			}
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
func (c *connection) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

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
		s, err := c.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		err = c.conf.SetSettings(ctx, s)
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
			&ui.Checkbox{Name: "hasColumnNames", Label: "The first row contains the column names", Role: ui.SourceRole},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
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
	if c.conf.Role == connector.SourceRole {
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
		s.HasColumnNames = false
	}
	return json.Marshal(&s)
}

// Write writes to w the records read from records.
func (c *connection) Write(ctx context.Context, w io.Writer, _ string, records connector.RecordReader) error {

	v := csv.NewWriter(w)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Comma)
	v.UseCRLF = c.settings.UseCRLF

	// Write the column names.
	columns := records.Columns()
	recordString := make([]string, len(columns))
	for i, c := range columns {
		if c.Label != "" {
			recordString[i] = c.Label
		} else {
			recordString[i] = c.Name
		}
	}
	err := v.Write(recordString)
	if err != nil {
		return err
	}

	// Write the records.
	for {
		record, err := records.Record()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for i, col := range columns {
			recordString[i] = toString(record[i], col.Type)
		}
		err = v.Write(recordString)
		if err != nil {
			return err
		}
	}

	v.Flush()
	err = v.Error()

	return err
}

// toString serializes v of type t as a string.
func toString(v any, t types.Type) string {
	if v == nil {
		return ""
	}
	switch pt := t.PhysicalType(); pt {
	case types.PtBoolean:
		return strconv.FormatBool(v.(bool))
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64, types.PtYear:
		return strconv.Itoa(v.(int))
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		return strconv.FormatUint(uint64(v.(uint)), 10)
	case types.PtFloat:
		return strconv.FormatFloat(v.(float64), 'g', -1, 64)
	case types.PtFloat32:
		return strconv.FormatFloat(v.(float64), 'g', -1, 32)
	case types.PtDecimal:
		return v.(decimal.Decimal).String()
	case types.PtDateTime:
		return v.(time.Time).Format(time.RFC3339Nano)
	case types.PtDate:
		return v.(time.Time).Format(time.DateOnly)
	case types.PtTime:
		return v.(time.Time).Format("15:04:05.999999999Z07:00")
	case types.PtUUID, types.PtInet, types.PtText:
		return v.(string)
	case types.PtJSON:
		return string(v.(json.RawMessage))
	case types.PtArray, types.PtObject, types.PtMap:
		return string(connector.MarshalJSON(v, t))
	default:
		panic(fmt.Sprintf("unexpected physical type %s", pt))
	}
}

// columnIndexToPropertyName returns a property name from a column index.
// Column indexes starts from 1.
func columnIndexToPropertyName(i int) string {
	// The code of this function has the following license:
	// https://github.com/qax-os/excelize/blob/master/LICENSE
	var c string
	for i > 0 {
		c = string(rune((i-1)%26+'A')) + c
		i = (i - 1) / 26
	}
	return c
}
