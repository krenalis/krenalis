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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"chichi"
	"chichi/types"
	"chichi/ui"

	"github.com/xuri/excelize/v2"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Sheets interface.
var _ chichi.Sheets = (*Excel)(nil)

func init() {
	chichi.RegisterFile(chichi.File{
		Name:              "Excel",
		SourceDescription: "import users from an Excel file",
		Icon:              icon,
		Extension:         "xlsx",
	}, New)
}

// New returns a new Excel connection.
func New(conf *chichi.FileConfig) (*Excel, error) {
	c := Excel{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connection")
		}
	}
	return &c, nil
}

type Excel struct {
	conf     *chichi.FileConfig
	settings *settings
}

type settings struct {
	HasColumnNames bool
}

// ContentType returns the content type of the file.
func (exel *Excel) ContentType(ctx context.Context) string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// Read reads the records from r and writes them to records. sheet is the name
// of the sheet to be read. If the provided sheet does not exist, it returns the
// chichi.ErrSheetNotExist error.
func (exel *Excel) Read(ctx context.Context, r io.Reader, sheet string, records chichi.RecordWriter) error {

	f, err := excelize.OpenReader(r, excelize.Options{
		RawCellValue: true,
	})
	if err != nil {
		// Don't return a Zip error because it might be misleading.
		if err.Error() == "zip: not a valid zip file" {
			err = errors.New("not a valid Excel '.xlsx' file")
		}
		return err
	}
	defer f.Close()
	rows, err := f.Rows(sheet)
	if err != nil {
		if _, ok := err.(excelize.ErrSheetNotExist); ok {
			return chichi.ErrSheetNotExist
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
					name := chichi.SuggestPropertyName(header)
					if name == "" {
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
			if exel.settings.HasColumnNames {
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
func (exel *Excel) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if exel.settings != nil {
			s = *exel.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := exel.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		err = exel.conf.SetSettings(ctx, s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Checkbox{Name: "hasColumnNames", Label: "The first row contains the column names", Role: ui.Source},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// Sheets returns the sheets of the file read from r.
func (exel *Excel) Sheets(ctx context.Context, r io.Reader) ([]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		// Don't return a Zip error because it might be misleading.
		if err.Error() == "zip: not a valid zip file" {
			err = errors.New("not a valid Excel '.xlsx' file")
		}
		return nil, err
	}
	defer f.Close()
	return f.GetSheetList(), nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (exel *Excel) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	if exel.conf.Role != chichi.Source {
		s.HasColumnNames = false
	}
	return json.Marshal(&s)
}

// Write writes to w the records read from records.
// sheet is the name of the sheet to be written to.
func (exel *Excel) Write(ctx context.Context, w io.Writer, sheet string, records chichi.RecordReader) error {

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
	for i := 2; ; i++ {
		gid, record, err := records.Record(ctx)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		axis := "A" + strconv.Itoa(i)
		err = sw.SetRow(axis, record)
		if err != nil {
			return err
		}
		records.Ack(gid, nil)
	}

	err = sw.Flush()
	if err != nil {
		return err
	}

	// Write the records into the destination file.
	_, err = f.WriteTo(w)

	return err
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
