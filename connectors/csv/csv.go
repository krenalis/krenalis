// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package csv provides a connector for CSV.
// (https://www.ietf.org/rfc/rfc4180.txt)
package csv

import (
	"context"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterFile(connectors.FileSpec{
		Code:       "csv",
		Label:      "CSV",
		Categories: connectors.CategoryFile,
		Extension:  "csv",
		AsSource: &connectors.AsSourceFile{
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsDestinationFile{
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new connector instance for CSV.
func New(env *connectors.FileEnv) (*CSV, error) {
	c := CSV{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for CSV")
		}
	}
	return &c, nil
}

type CSV struct {
	env      *connectors.FileEnv
	settings *innerSettings
}

type innerSettings struct {
	Separator        string
	NumberOfColumns  int
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
func (c *CSV) Read(ctx context.Context, r io.Reader, sheet string, records connectors.RecordWriter) error {

	// Create a CSV reader.
	v := csv.NewReader(r)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Separator)
	v.FieldsPerRecord = c.settings.NumberOfColumns
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
					name, ok := types.PropertyName(header)
					if !ok {
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
					nameOfHeader[header] = name
				} else {
					name := columnNumberToName(i + 1)
					columns[i].Name = name
				}
				columns[i].Type = types.String()
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
		err = records.RecordStrings(record)
		if err != nil {
			return err
		}
	}

	return nil
}

// ServeUI serves the connector's user interface.
func (c *CSV) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if c.settings == nil {
			s.Separator = ","
		} else {
			s = *c.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, c.saveSettings(ctx, settings, role)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "Separator", Label: "Separator", Placeholder: ",", Type: "text", MinLength: 1, MaxLength: 1},
			&connectors.Input{Name: "NumberOfColumns", Label: "Number of columns", Placeholder: "", HelpText: "When 0, it is determined from the first record.", Type: "number", OnlyIntegerPart: true, Role: connectors.Source},
			&connectors.Checkbox{Name: "TrimLeadingSpace", Label: "Trim leading space in fields", Role: connectors.Source},
			&connectors.Checkbox{Name: "UseCRLF", Label: "Use CRLF", Role: connectors.Destination},
			&connectors.Checkbox{Name: "HasColumnNames", Label: "The first row contains the column names", Role: connectors.Source},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes to w the records read from records.
func (c *CSV) Write(ctx context.Context, w io.Writer, _ string, records connectors.RecordReader) error {

	v := csv.NewWriter(w)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Separator)
	v.UseCRLF = c.settings.UseCRLF

	// Write the column names.
	columns := records.Columns()
	recordString := make([]string, len(columns))
	for i, c := range columns {
		recordString[i] = c.Name
	}
	err := v.Write(recordString)
	if err != nil {
		return err
	}

	// Write the records.
	for {
		record, err := records.Record(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		for i, c := range columns {
			recordString[i] = toString(record[c.Name], c.Type)
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

// saveSettings saves the settings.
func (c *CSV) saveSettings(ctx context.Context, settings json.Value, role connectors.Role) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Separator.
	if utf8.RuneCountInString(s.Separator) != 1 {
		return connectors.NewInvalidSettingsError("separator must be a single character")
	}
	if c := s.Separator; c == "\n" || c == "\r" || c == "\uFFFD" {
		return connectors.NewInvalidSettingsError("separator cannot be \\r, \\n, or the Unicode replacement character")
	}
	if role == connectors.Source {
		// Validate NumberOfColumns.
		if f := s.NumberOfColumns; f < 0 || f > 1000 {
			return connectors.NewInvalidSettingsError("number of columns, if provided, must be in range [0,1000]")
		}
	} else {
		s.NumberOfColumns = 0
		s.TrimLeadingSpace = false
		s.HasColumnNames = false
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = c.env.SetSettings(ctx, b)
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
	case types.StringKind, types.UUIDKind, types.IPKind:
		return v.(string)
	case types.BooleanKind:
		return strconv.FormatBool(v.(bool))
	case types.IntKind:
		if t.IsUnsigned() {
			return strconv.FormatUint(uint64(v.(uint)), 10)
		}
		return strconv.Itoa(v.(int))
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
	case types.YearKind:
		return strconv.Itoa(v.(int))
	case types.JSONKind:
		return string(v.(json.Value))
	case types.ArrayKind, types.ObjectKind, types.MapKind:
		data, _ := types.Marshal(v, t)
		return string(data)
	default:
		panic(fmt.Sprintf("unexpected kind %s", k))
	}
}
