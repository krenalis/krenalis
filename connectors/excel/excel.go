//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package excel

// This package is the Excel connector.
// (http://www.office.microsoft.com/excel)

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"

	"github.com/xuri/excelize/v2"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterFile(connector.File{
		Name:              "Excel",
		SourceDescription: "import users from an Excel file",
		Icon:              icon,
	}, open)
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	Path      string
	SheetName string
}

// open opens an Excel connection and returns it.
func open(ctx context.Context, conf *connector.FileConfig) (*connection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Excel connection")
		}
	}
	return &c, nil
}

// ContentType returns the content type of the file.
func (c *connection) ContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// Path returns the path of the file.
func (c *connection) Path() string {
	return c.settings.Path
}

// Read reads the records from r, with their last update time, and writes
// them to records.
func (c *connection) Read(r io.Reader, updateTime time.Time, records connector.RecordWriter) error {

	if err := records.Timestamp(updateTime); err != nil {
		return err
	}

	f, err := excelize.OpenReader(r, excelize.Options{
		RawCellValue: true,
	})
	if err != nil {
		return err
	}
	defer f.Close()
	sheetName := c.settings.SheetName
	if sheetName == "" {
		sheetName = f.GetSheetName(0)
	}
	rows, err := f.Rows(sheetName)
	if err != nil {
		return err
	}
	defer rows.Close()

	var first bool
	for rows.Next() {
		// Read a record.
		record, err := rows.Columns()
		if err != nil {
			return err
		}
		// Writes the columns.
		if first {
			columns := make([]connector.Column, len(record))
			for i, c := range columns {
				// Set the name.
				c.Name = "column" + strconv.Itoa(i+1)
				// Set the type.
				axis, err := excelize.CoordinatesToCellName(i+1, 1)
				if err != nil {
					return err
				}
				t, err := f.GetCellType(sheetName, axis)
				if err != nil {
					return err
				}
				c.Type, err = columnType(c.Name, t)
				if err != nil {
					return err
				}
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
		if c.settings != nil {
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
			&ui.Input{Name: "sheetName", Label: "Sheet name", Placeholder: "Sheet 1", Type: "text", MinLength: 1, MaxLength: 31},
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
	// Validate SheetName.
	if name := s.SheetName; name == "" || utf8.RuneCountInString(name) > 31 || strings.ContainsAny(name, ":\\/?*[]") {
		return nil, ui.Errorf("sheet name cannot be longer than 31 characters and cannot contain :, \\, /, ?, *, [ and ]")
	}
	return json.Marshal(&s)
}

// Write writes to w the records read from records.
func (c *connection) Write(w io.Writer, records connector.RecordReader) error {

	f := excelize.NewFile()
	defer f.Close()
	sw, err := f.NewStreamWriter(c.settings.SheetName)
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
		record, err := records.Record()
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
	}

	err = sw.Flush()
	if err != nil {
		return err
	}

	// Write the records into the destination file.
	_, err = f.WriteTo(w)
	if err != nil {
		return err
	}

	return err
}

// columnType returns the column type from an Excel column.
func columnType(c string, t excelize.CellType) (types.Type, error) {
	switch t {
	case excelize.CellTypeBool:
		return types.Boolean(), nil
	case excelize.CellTypeDate:
		return types.Date(""), nil // TODO(marco) set the layout
	case excelize.CellTypeNumber:
		return types.Decimal(0, 0), nil
	case excelize.CellTypeUnset, excelize.CellTypeError, excelize.CellTypeInlineString, excelize.CellTypeSharedString:
		return types.Text(), nil
	default:
		return types.Type{}, connector.NewNotSupportedTypeError(c, strconv.Itoa(int(t)))
	}
}
