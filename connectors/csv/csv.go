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
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	meergo.RegisterFile(meergo.FileInfo{
		Name:       "CSV",
		Categories: meergo.CategoryFile,
		Icon:       icon,
		Extension:  "csv",
		AsSource: &meergo.AsSourceFile{
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsDestinationFile{
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: destinationOverview,
			},
		},
	}, New)
}

// New returns a new CSV connector instance.
func New(env *meergo.FileEnv) (*CSV, error) {
	c := CSV{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connector")
		}
	}
	return &c, nil
}

type CSV struct {
	env      *meergo.FileEnv
	settings *innerSettings
}

type innerSettings struct {
	Separator        string
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
func (c *CSV) Read(ctx context.Context, r io.Reader, sheet string, records meergo.RecordWriter) error {

	// Create a CSV reader.
	v := csv.NewReader(r)
	v.Comma, _ = utf8.DecodeRuneInString(c.settings.Separator)
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
					name, ok := meergo.SuggestPropertyName(header)
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
		err = records.RecordStrings(record)
		if err != nil {
			return err
		}
	}

	return nil
}

// ServeUI serves the connector's user interface.
func (c *CSV) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

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
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "Separator", Label: "Separator", Placeholder: ",", Type: "text", MinLength: 1, MaxLength: 1},
			&meergo.Input{Name: "FieldsPerRecord", Label: "Fields per record", Placeholder: "", Type: "number", OnlyIntegerPart: true, Role: meergo.Source},
			&meergo.Checkbox{Name: "TrimLeadingSpace", Label: "Trim leading space", Role: meergo.Source},
			&meergo.Checkbox{Name: "UseCRLF", Label: "Use CRLF", Role: meergo.Destination},
			&meergo.Checkbox{Name: "HasColumnNames", Label: "The first row contains the column names", Role: meergo.Source},
		},
		Settings: settings,
	}

	return ui, nil
}

// Write writes to w the records read from records.
func (c *CSV) Write(ctx context.Context, w io.Writer, _ string, records meergo.RecordReader) error {

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
		id, record, err := records.Record(ctx)
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
		records.Ack(id, nil)
	}

	v.Flush()
	err = v.Error()

	return err
}

// saveSettings saves the settings.
func (c *CSV) saveSettings(ctx context.Context, settings json.Value, role meergo.Role) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate Separator.
	if utf8.RuneCountInString(s.Separator) != 1 {
		return meergo.NewInvalidSettingsError("separator must be a single character")
	}
	if c := s.Separator; c == "\n" || c == "\r" || c == "\uFFFD" {
		return meergo.NewInvalidSettingsError("separator cannot be \\r, \\n, or the Unicode replacement character")
	}
	if role == meergo.Source {
		// Validate FieldsPerRecord.
		if f := s.FieldsPerRecord; f < 0 || f > 1000 {
			return meergo.NewInvalidSettingsError("fields per record, if provided, must be in range [0,1000]")
		}
	} else {
		s.FieldsPerRecord = 0
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
		return string(v.(json.Value))
	case types.ArrayKind, types.ObjectKind, types.MapKind:
		data, _ := types.Marshal(v, t)
		return string(data)
	default:
		panic(fmt.Sprintf("unexpected kind %s", k))
	}
}
