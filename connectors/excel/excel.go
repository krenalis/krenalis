//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package excel implements the Excel connector.
// (http://www.office.microsoft.com/excel)
package excel

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"

	"github.com/xuri/excelize/v2"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	meergo.RegisterFile(meergo.FileInfo{
		Name:       "Excel",
		Categories: meergo.CategoryFile,
		AsSource: &meergo.AsSourceFile{
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsDestinationFile{
			Documentation: meergo.ConnectorRoleDocumentation{
				Overview: destinationOverview,
			},
		},
		HasSheets: true,
		Extension: "xlsx",
		Icon:      icon,
	}, New)
}

// New returns a new Excel connector instance.
func New(env *meergo.FileEnv) (*Excel, error) {
	c := Excel{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connector")
		}
	}
	return &c, nil
}

type Excel struct {
	env      *meergo.FileEnv
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
func (exel *Excel) Read(ctx context.Context, r io.Reader, sheet string, records meergo.RecordWriter) error {

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
			return meergo.ErrSheetNotExist
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
					name, ok := meergo.SuggestPropertyName(header)
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
func (exel *Excel) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

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
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Checkbox{Name: "HasColumnNames", Label: "The first row contains the column names", Role: meergo.Source},
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
func (exel *Excel) Write(ctx context.Context, w io.Writer, sheet string, records meergo.RecordReader) error {

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
func (exel *Excel) saveSettings(ctx context.Context, settings json.Value, role meergo.Role) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if role != meergo.Source {
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
