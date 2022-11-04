//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package excel

// This package is the Excel connector.
// (http://www.office.microsoft.com/excel)

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"chichi/apis"
	"chichi/connector"

	"github.com/xuri/excelize/v2"
)

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

// Read reads the records from r and calls put for each record read.
func (c *connection) Read(r io.Reader, put func(record []string) error) error {
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
	for rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			return err
		}
		err = put(row)
		if err != nil {
			return err
		}
	}
	return rows.Close()
}

// Write writes the records read from get into w.
func (c *connection) Write(w io.Writer, get func() ([]string, error)) error {
	f := excelize.NewFile()
	sw, err := f.NewStreamWriter(c.settings.SheetName)
	if err != nil {
		return err
	}
	var row []any
	i := 1
	for {
		record, err := get()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if row == nil {
			row = make([]any, len(record))
		}
		for i, v := range record {
			row[i] = v
		}
		axis := "A" + strconv.Itoa(i)
		err = sw.SetRow(axis, row)
		if err != nil {
			return err
		}
		i++
	}
	err = sw.Flush()
	if err != nil {
		return err
	}
	_, err = f.WriteTo(w)
	return err
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, form []byte) (*connector.SettingsUI, error) {

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if c.settings != nil {
			s = *c.settings
		}
	case "save":
		// Save the settings.
		err := json.Unmarshal(form, &s)
		if err != nil {
			return nil, err
		}
		// Validate SheetName.
		if name := s.SheetName; name == "" || utf8.RuneCountInString(name) > 31 || strings.ContainsAny(name, ":\\/?*[]") {
			return nil, connector.UIErrorf("sheet name cannot be longer than 31 characters and cannot contain :, \\, /, ?, *, [ and ]")
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	default:
		return nil, connector.ErrEventNotExist
	}

	ui := &connector.SettingsUI{
		Components: []connector.Component{
			&connector.Input{Name: "sheetName", Value: s.SheetName, Label: "Sheet name", Placeholder: "Sheet 1", Type: "text", MinLength: 1, MaxLength: 31},
		},
		Actions: []connector.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return ui, nil
}
