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
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"chichi/apis"
	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"

	"github.com/xuri/excelize/v2"
)

// Connector icon.
var icon []byte

// Make sure it implements the FileConnector interface.
var _ connector.FileConnection = &connection{}

func init() {
	apis.RegisterFileConnector("Excel", New)
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	SheetName string
}

// New returns a new Excel connection.
func New(ctx context.Context, settings []byte, fh connector.Firehose) (connector.FileConnection, error) {
	c := connection{ctx: ctx}
	if len(settings) > 0 {
		err := json.Unmarshal(settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Excel connection")
		}
	}
	c.firehose = fh
	return &c, nil
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "Excel",
		Type: connector.TypeFile,
		Icon: icon,
	}
}

// ContentType returns the content type of the data to write.
func (c *connection) ContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// Read reads the records from r and write them to records.
func (c *connection) Read(r io.Reader, records connector.RecordWriter) error {
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
				c.Type, err = columnType(t)
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
	_, err = f.WriteTo(w)

	return err
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, error) {

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
			return nil, err
		}
		// Validate SheetName.
		if name := s.SheetName; name == "" || utf8.RuneCountInString(name) > 31 || strings.ContainsAny(name, ":\\/?*[]") {
			return nil, ui.Errorf("sheet name cannot be longer than 31 characters and cannot contain :, \\, /, ?, *, [ and ]")
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	default:
		return nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "sheetName", Value: s.SheetName, Label: "Sheet name", Placeholder: "Sheet 1", Type: "text", MinLength: 1, MaxLength: 31},
		},
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil
}

// columnType returns the column type from the Excel column type t.
func columnType(t excelize.CellType) (types.Type, error) {
	switch t {
	case excelize.CellTypeBool:
		return types.Boolean(), nil
	case excelize.CellTypeDate:
		return types.Date(), nil
	case excelize.CellTypeNumber:
		return types.Decimal(0, 0), nil
	case excelize.CellTypeUnset, excelize.CellTypeError, excelize.CellTypeString:
		return types.Text(), nil
	default:
		return types.Type{}, fmt.Errorf("unexpected Excel type %d", t)
	}
}
