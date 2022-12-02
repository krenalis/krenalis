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
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"

	"github.com/xuri/excelize/v2"
)

// Connector icon.
var icon []byte

// Make sure it implements the FileConnection interface.
var _ connector.FileConnection = &connection{}

func init() {
	connector.RegisterFile(connector.File{
		Name:    "Excel",
		Icon:    icon,
		Connect: connect,
	})
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

// connect returns a new Excel connection.
func connect(ctx context.Context, conf *connector.FileConfig) (connector.FileConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Excel connection")
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

	f, err := excelize.OpenReader(r, excelize.Options{
		RawCellValue: true,
	})
	if err != nil {
		return err
	}
	defer f.Close()
	_ = r.Close()
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

	var s settings

	switch event {
	case "load":
		// Load the Form.
		if c.settings != nil {
			s = *c.settings
		}
	case "save":
		// Save the settings.
		err := json.Unmarshal(values, &s)
		if err != nil {
			return nil, nil, err
		}
		// Validate Path.
		if s.Path == "" {
			return nil, nil, ui.Errorf("path cannot be empty")
		}
		if utf8.RuneCountInString(s.Path) > 1000 {
			return nil, nil, ui.Errorf("path cannot be longer that 1000 characters")
		}
		// Validate SheetName.
		if name := s.SheetName; name == "" || utf8.RuneCountInString(name) > 31 || strings.ContainsAny(name, ":\\/?*[]") {
			return nil, nil, ui.Errorf("sheet name cannot be longer than 31 characters and cannot contain :, \\, /, ?, *, [ and ]")
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, nil, err
		}
		err = c.firehose.SetSettings(b)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "path", Value: s.Path, Label: "Path", Placeholder: "", Type: "text", MinLength: 1, MaxLength: 1000},
			&ui.Input{Name: "sheetName", Value: s.SheetName, Label: "Sheet name", Placeholder: "Sheet 1", Type: "text", MinLength: 1, MaxLength: 31},
		},
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// Write writes to files the records read from records.
func (c *connection) Write(files connector.FileWriter, records connector.RecordReader) error {

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
	w, err := files.Writer(c.settings.Path, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = f.WriteTo(w)
	if err != nil {
		return err
	}
	err = w.Close()

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
	case excelize.CellTypeUnset, excelize.CellTypeError, excelize.CellTypeString:
		return types.Text(), nil
	default:
		return types.Type{}, connector.NewNotSupportedTypeError(c, strconv.Itoa(int(t)))
	}
}
