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

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the File and UIHandler interfaces.
var _ interface {
	chichi.File
	chichi.UIHandler
} = (*CSV)(nil)

func init() {
	chichi.RegisterFile(chichi.FileInfo{
		Name:      "CSV",
		Icon:      icon,
		Extension: "csv",
	}, New)
}

// New returns a new CSV connector instance.
func New(conf *chichi.FileConfig) (*CSV, error) {
	c := CSV{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connector")
		}
	}
	return &c, nil
}

type CSV struct {
	conf     *chichi.FileConfig
	settings *Settings
}

type Settings struct {
	Comma            string
	Comment          string
	FieldsPerRecord  int
	LazyQuotes       bool
	TrimLeadingSpace bool
	UseCRLF          bool
	HasColumnNames   bool
}

// ContentType returns the content type of the file.
func (c *CSV) ContentType(ctx context.Context) string {
	return "text/csv; charset=UTF-8"
}

// Read reads the records from r and writes them to records.
func (c *CSV) Read(ctx context.Context, r io.Reader, sheet string, records chichi.RecordWriter) error {

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
					name := chichi.SuggestPropertyName(header)
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
					name := columnNumberToName(i + 1)
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
func (c *CSV) ServeUI(ctx context.Context, event string, values []byte) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if c.settings == nil {
			s.Comma = ","
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, c.saveValues(ctx, values)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "Comma", Label: "Comma", Placeholder: ",", Type: "text", MinLength: 1, MaxLength: 1},
			&chichi.Input{Name: "Comment", Label: "Comment", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 1, Role: chichi.Source},
			&chichi.Input{Name: "FieldsPerRecord", Label: "Fields per record", Placeholder: "", Type: "number", OnlyIntegerPart: true, Role: chichi.Source},
			&chichi.Checkbox{Name: "TrimLeadingSpace", Label: "Trim leading space", Role: chichi.Source},
			&chichi.Checkbox{Name: "UseCRLF", Label: "Use CRLF"},
			&chichi.Checkbox{Name: "HasColumnNames", Label: "The first row contains the column names", Role: chichi.Source},
		},
		Values: values,
	}

	return ui, nil
}

// Write writes to w the records read from records.
func (c *CSV) Write(ctx context.Context, w io.Writer, _ string, records chichi.RecordReader) error {

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
		gid, record, err := records.Record(ctx)
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
		records.Ack(gid, err)
	}

	v.Flush()
	err = v.Error()

	return err
}

// saveValues saves the user-entered values as settings.
func (c *CSV) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate Comma.
	if utf8.RuneCountInString(s.Comma) != 1 {
		return chichi.NewInvalidUIValuesError("comma must be a single character")
	}
	if c := s.Comma; c == "\n" || c == "\r" || c == "\uFFFD" {
		return chichi.NewInvalidUIValuesError("comma cannot be \\r, \\n, or the Unicode replacement character")
	}
	if c.conf.Role == chichi.Source {
		// Validate Comment.
		if c := s.Comment; c != "" {
			if utf8.RuneCountInString(c) != 1 {
				return chichi.NewInvalidUIValuesError("comment, if provided, must be a single character")
			}
			if c == "\n" || c == "\r" || c == "\uFFFD" {
				return chichi.NewInvalidUIValuesError("comment cannot be \\r, \\n, or the Unicode replacement character")
			}
			if c == s.Comma {
				return chichi.NewInvalidUIValuesError("comment cannot be equal to the comma")
			}
		}
		// Validate FieldsPerRecord.
		if f := s.FieldsPerRecord; f < 0 || f > 1000 {
			return chichi.NewInvalidUIValuesError("fields per record, if provided, must be in range [0,1000]")
		}
	} else {
		s.Comment = ""
		s.FieldsPerRecord = 0
		s.TrimLeadingSpace = false
		s.HasColumnNames = false
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = c.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	c.settings = &s
	return nil
}

// columnNumberToName returns a column name from a column number.
// Column numbers starts from 1.
func columnNumberToName(n int) string {
	// The code of this function has the following license:
	// https://github.com/qax-os/excelize/blob/master/LICENSE
	var c string
	for n > 0 {
		c = string(rune((n-1)%26+'A')) + c
		n = (n - 1) / 26
	}
	return c
}

// toString serializes v of type t as a string.
func toString(v any, t types.Type) string {
	if v == nil {
		return ""
	}
	switch k := t.Kind(); k {
	case types.BooleanKind:
		return strconv.FormatBool(v.(bool))
	case types.IntKind, types.YearKind:
		return strconv.Itoa(v.(int))
	case types.UintKind:
		return strconv.FormatUint(uint64(v.(uint)), 10)
	case types.FloatKind:
		return strconv.FormatFloat(v.(float64), 'g', -1, t.BitSize())
	case types.DecimalKind:
		return v.(decimal.Decimal).String()
	case types.DateTimeKind:
		return v.(time.Time).Format(time.RFC3339Nano)
	case types.DateKind:
		return v.(time.Time).Format(time.DateOnly)
	case types.TimeKind:
		return v.(time.Time).Format("15:04:05.999999999Z07:00")
	case types.UUIDKind, types.InetKind, types.TextKind:
		return v.(string)
	case types.JSONKind:
		return string(v.(json.RawMessage))
	case types.ArrayKind, types.ObjectKind, types.MapKind:
		return string(chichi.MarshalJSON(v, t))
	default:
		panic(fmt.Sprintf("unexpected kind %s", k))
	}
}
