// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package excel provides a connector for Excel.
// (https://learn.microsoft.com/en-us/openspecs/office_standards/ms-xlsx/f780b2d6-8252-4074-9fe3-5d7bc4830968)
//
// Microsoft and Excel are trademarks of Microsoft Corporation.
// This connector is not affiliated with or endorsed by Microsoft Corporation.
package excel

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"

	"github.com/xuri/excelize/v2"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterFile(connectors.FileSpec{
		Code:       "excel",
		Label:      "Excel",
		Categories: connectors.CategoryFile,
		AsSource: &connectors.AsSourceFile{
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsDestinationFile{
			Documentation: connectors.RoleDocumentation{
				Overview: destinationOverview,
			},
		},
		HasSheets: true,
		Extension: "xlsx",
	}, New)
}

// New returns a new connector instance for Excel.
func New(env *connectors.FileEnv) (*Excel, error) {
	c := Excel{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Excel")
		}
	}
	return &c, nil
}

type Excel struct {
	env      *connectors.FileEnv
	settings *innerSettings
}

type innerSettings struct {
	HasColumnNames bool
}

// ContentType returns the content type of the file.
func (exel *Excel) ContentType(ctx context.Context) string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

var errReadFile = errors.New("document is not a valid Excel (.xlsx) file or may be corrupted")

// Read reads the records from r and writes them to records.
func (exel *Excel) Read(ctx context.Context, r io.Reader, sheet string, records connectors.RecordWriter) error {

	f, err := excelize.OpenReader(r, excelize.Options{
		RawCellValue: true,
	})
	if err != nil {
		// Don't return a Zip error because it might be misleading.
		if err.Error() == "zip: not a valid zip file" {
			return errReadFile
		}
		return fmt.Errorf("%s: %s", errReadFile, err)
	}
	defer f.Close()
	rows, err := f.Rows(sheet)
	if err != nil {
		if _, ok := err.(excelize.ErrSheetNotExist); ok {
			return connectors.ErrSheetNotExist
		}
		return err
	}
	defer rows.Close()

	var nameOfHeader map[string]string

	first := true
	for rows.Next() {
		// Read a record.
		record, err := rows.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return err
		}
		// Writes the columns.
		if first {
			columns := make([]types.Property, len(record))
			for i := range columns {
				if exel.settings.HasColumnNames {
					header := record[i]
					name, ok := types.PropertyName(header)
					if !ok {
						return fmt.Errorf("header %q, of column %s, cannot be converted to a valid property name", header, columnNumberToName(i+1))
					}
					if nameOfHeader == nil {
						nameOfHeader = make(map[string]string, len(record))
					}
					for n, h := range nameOfHeader {
						if name == n {
							if header == h {
								return fmt.Errorf("header %q is repeated", header)
							}
							return fmt.Errorf("headers %q and %q cannot be converted into two different property names", header, h)
						}
					}
					columns[i].Name = name
					if name != record[i] {
						columns[i].Description = header
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
			if exel.settings.HasColumnNames {
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
func (exel *Excel) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if exel.settings != nil {
			s = *exel.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, exel.saveSettings(ctx, settings, role)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Checkbox{Name: "HasColumnNames", Label: "The first row contains the column names", Role: connectors.Source},
		},
		Settings: settings,
	}

	return ui, nil
}

// Sheets returns the sheets of the file read from r.
func (exel *Excel) Sheets(ctx context.Context, r io.Reader) ([]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		// Don't return a Zip error because it might be misleading.
		if err.Error() == "zip: not a valid zip file" {
			return nil, errReadFile
		}
		return nil, fmt.Errorf("%s: %s", errReadFile, err)
	}
	defer f.Close()
	return f.GetSheetList(), nil
}

// Write writes to w the records read from records.
func (exel *Excel) Write(ctx context.Context, w io.Writer, sheet string, records connectors.RecordReader) error {

	f := excelize.NewFile()
	defer f.Close()
	err := f.SetSheetName("Sheet1", sheet)
	if err != nil {
		return err
	}
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}

	// Write the column names.
	columns := records.Columns()
	record := make([]any, len(columns))
	for i, c := range columns {
		record[i] = c.Name
	}
	err = sw.SetRow("A1", record)
	if err != nil {
		return err
	}

	// Write the records.
	values := make([]any, len(columns))
	for i := 2; ; i++ {
		record, err := records.Record(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		axis := "A" + strconv.Itoa(i)
		for i, c := range columns {
			values[i] = record[c.Name]
		}
		err = sw.SetRow(axis, values)
		if err != nil {
			return err
		}
	}

	err = sw.Flush()
	if err != nil {
		return err
	}

	// Write the records into the destination file.
	_, err = f.WriteTo(w)

	return err
}

// saveSettings saves the settings.
func (exel *Excel) saveSettings(ctx context.Context, settings json.Value, role connectors.Role) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if role != connectors.Source {
		s.HasColumnNames = false
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = exel.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	exel.settings = &s
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
