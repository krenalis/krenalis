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
	"io"
	"strconv"

	"chichi/apis/types"
	"chichi/connector"

	"github.com/xuri/excelize/v2"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Sheets interface.
var _ connector.Sheets = (*connection)(nil)

func init() {
	connector.RegisterFile(connector.File{
		Name:              "Excel",
		SourceDescription: "import users from an Excel file",
		Icon:              icon,
	}, open)
}

type connection struct {
	ctx      context.Context
	firehose connector.Firehose
}

// open opens an Excel connection and returns it.
func open(ctx context.Context, conf *connector.FileConfig) (*connection, error) {
	return &connection{ctx: ctx, firehose: conf.Firehose}, nil
}

// ContentType returns the content type of the file.
func (c *connection) ContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// Read reads the records from r and writes them to records.
// sheet is the name of the sheet to be read.
func (c *connection) Read(r io.Reader, sheet string, records connector.RecordWriter) error {

	f, err := excelize.OpenReader(r, excelize.Options{
		RawCellValue: true,
	})
	if err != nil {
		return err
	}
	defer f.Close()
	rows, err := f.Rows(sheet)
	if err != nil {
		return err
	}
	defer rows.Close()

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
				columns[i].Name = "column" + strconv.Itoa(i+1)
				columns[i].Label = record[i]
				columns[i].Type = types.Text()
			}
			err = records.Columns(columns)
			if err != nil {
				return err
			}
			first = false
			continue
		}
		// Write the record.
		err = records.RecordString(record)
		if err != nil {
			return err
		}
	}

	return nil
}

// Sheets returns the sheets of the file read from r.
func (c *connection) Sheets(r io.Reader) ([]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.GetSheetList(), nil
}

// Write writes to w the records read from records.
// sheet is the name of the sheet to be written to.
func (c *connection) Write(w io.Writer, sheet string, records connector.RecordReader) error {

	f := excelize.NewFile()
	defer f.Close()
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
