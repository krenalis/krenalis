//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package parquet

// This package is the Parquet connector.
// (https://github.com/apache/parquet-format)

import (
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"

	goparquet "github.com/fraugster/parquet-go"
	"github.com/fraugster/parquet-go/parquet"
)

// Connector icon.
var icon []byte

// Make sure it implements the FileConnection interface.
var _ connector.FileConnection = &connection{}

func init() {
	connector.RegisterFile("Parquet", newConnection)
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	Path string
}

// newConnection returns a new Parquet connection.
func newConnection(ctx context.Context, conf *connector.FileConfig) (connector.FileConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Excel connection")
		}
	}
	return &c, nil
}

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "Parquet",
		Type: connector.FileType,
		Icon: icon,
	}
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

	// Copy data read from r to a temporary file.
	dir := os.TempDir()
	fi, err := os.CreateTemp(dir, "")
	if err != nil {
		return err
	}
	defer func() {
		_ = fi.Close()
		_ = os.Remove(filepath.Join(dir, fi.Name()))
	}()
	_, err = io.Copy(fi, r)
	if err != nil {
		return err
	}
	_ = r.Close()
	_, err = fi.Seek(io.SeekStart, 0)
	if err != nil {
		return err
	}

	fr, err := goparquet.NewFileReaderWithOptions(fi, goparquet.WithReaderContext(c.ctx))
	if err != nil {
		return err
	}

	// Read the columns.
	var int96Columns []string
	parquetColumns := fr.Columns()
	columns := make([]connector.Column, len(parquetColumns))
	for i, c := range parquetColumns {
		name := strings.Join(c.Path(), ".")
		element := c.Element()
		columns[i].Name = name
		columns[i].Type, err = propertyType(name, element)
		if err != nil {
			return err
		}
		if *element.Type == parquet.Type_INT96 {
			int96Columns = append(int96Columns, name)
		}
	}
	// Write the columns.
	err = records.Columns(columns)
	if err != nil {
		return err
	}

	for {
		// Read a record.
		record, err := fr.NextRow()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// Convert int96 type values from []byte to time.Time.
		for _, name := range int96Columns {
			record[name], err = convertInt96(record[name])
			if err != nil {
				return fmt.Errorf("cannot convert value of column %q: %s", name, err)
			}
		}
		// Write the record.
		err = records.RecordMap(record)
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
			return nil, nil, ui.Errorf("path cannot be longer that 1000")
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
		},
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// Write writes to files the records read from records.
func (c *connection) Write(files connector.FileWriter, records connector.RecordReader) error {
	// TODO(marco)
	return nil
}

// propertyType returns the property type of the Parquet column with the given
// name and type (https://github.com/apache/parquet-format).
func propertyType(column string, elem *parquet.SchemaElement) (types.Type, error) {

	if elem.Type == nil {
		return types.Type{}, errors.New("unexpected Parquet nil type")
	}

	// Physical types.
	switch *elem.Type {
	case parquet.Type_BOOLEAN:
		return types.Boolean(), nil
	case parquet.Type_FLOAT:
		return types.Float32(), nil
	case parquet.Type_DOUBLE:
		return types.Float(), nil
	}

	// Logical types.
	// (https://github.com/apache/parquet-format/blob/master/LogicalTypes.md)
	if lt := elem.LogicalType; lt != nil {
		if lt.STRING != nil {
			return types.Text(), nil
		}
		if lt.ENUM != nil {
			return types.Text(), nil
		}
		if d := lt.DECIMAL; d != nil {
			if 0 < d.Precision && d.Precision <= types.MaxDecimalPrecision &&
				d.Scale <= types.MaxDecimalScale && d.Scale <= d.Precision {
				return types.Decimal(int(d.Precision), int(d.Scale)), nil
			}
			return types.Decimal(0, 0), nil
		}
		if lt.DATE != nil {
			return types.DateTime(""), nil // TODO(marco) set the layout
		}
		if lt.TIMESTAMP != nil {
			return types.DateTime(""), nil // TODO(marco) set the layout
		}
		if lt.TIME != nil {
			return types.Time(""), nil // TODO(marco) add unit of measure
		}
		if lt.INTEGER != nil {
			if lt.INTEGER.IsSigned {
				switch lt.INTEGER.BitWidth {
				case 8:
					return types.Int8(), nil
				case 16:
					return types.Int16(), nil
				case 32:
					return types.Int(), nil
				case 64:
					return types.Int64(), nil
				}
				return types.Type{}, fmt.Errorf("unexpected Parquet bitWidth value: %d", lt.INTEGER.BitWidth)
			}
			switch lt.INTEGER.BitWidth {
			case 8:
				return types.UInt8(), nil
			case 16:
				return types.UInt16(), nil
			case 32:
				return types.UInt(), nil
			case 64:
				return types.UInt64(), nil
			}
			return types.Type{}, fmt.Errorf("unexpected Parquet bitWidth value: %d", lt.INTEGER.BitWidth)
		}
		if lt.JSON != nil || lt.BSON != nil {
			return types.JSON(), nil
		}
		if lt.UUID != nil {
			return types.UUID(), nil
		}
		return types.Type{}, fmt.Errorf("unsupported logical Parquet type %q", lt)
	}

	// Converted types.
	if ct := elem.ConvertedType; ct != nil {
		switch *ct {
		case parquet.ConvertedType_UTF8, parquet.ConvertedType_ENUM:
			return types.Text(), nil
		case parquet.ConvertedType_INT_8:
			return types.Int8(), nil
		case parquet.ConvertedType_INT_16:
			return types.Int16(), nil
		case parquet.ConvertedType_INT_32:
			return types.Int(), nil
		case parquet.ConvertedType_INT_64:
			return types.Int64(), nil
		case parquet.ConvertedType_UINT_8:
			return types.UInt8(), nil
		case parquet.ConvertedType_UINT_16:
			return types.UInt16(), nil
		case parquet.ConvertedType_UINT_32:
			return types.UInt(), nil
		case parquet.ConvertedType_UINT_64:
			return types.UInt64(), nil
		case parquet.ConvertedType_JSON, parquet.ConvertedType_BSON:
			return types.JSON(), nil
		case parquet.ConvertedType_DECIMAL:
			if elem.Precision != nil && *elem.Precision <= types.MaxDecimalPrecision &&
				elem.Scale != nil && *elem.Scale <= types.MaxDecimalScale && *elem.Scale <= *elem.Precision {
				return types.Decimal(int(*elem.Precision), int(*elem.Scale)), nil
			}
			return types.Decimal(0, 0), nil
		case parquet.ConvertedType_DATE:
			return types.Date(""), nil // TODO(marco) set the layout
		case parquet.ConvertedType_TIMESTAMP_MICROS, parquet.ConvertedType_TIMESTAMP_MILLIS:
			return types.DateTime(""), nil // TODO(marco) set the layout
		case parquet.ConvertedType_TIME_MICROS, parquet.ConvertedType_TIME_MILLIS:
			return types.Time(""), nil // // TODO(marco) set the layout
		}
		return types.Type{}, fmt.Errorf("unsupported converted Parquet type %q", *ct)
	}

	// Physical types.
	switch *elem.Type {
	case parquet.Type_INT32:
		return types.Int(), nil
	case parquet.Type_INT64:
		return types.Int64(), nil
	case parquet.Type_INT96:
		return types.DateTime(""), nil // TODO(marco) set the layout
	case parquet.Type_BYTE_ARRAY, parquet.Type_FIXED_LEN_BYTE_ARRAY:
		return types.Text(), nil
	}

	return types.Type{}, connector.NewNotSupportedTypeError(column, (*elem.Type).String())
}

// Convert an int96 type value to a time.Time value.
// v must be a byte array with length in range [8,96].
// See https://stackoverflow.com/questions/53103762.
func convertInt96(v any) (time.Time, error) {
	r := reflect.ValueOf(v)
	t := r.Type()
	// Validate the argument.
	if t.Kind() != reflect.Array || t.Elem().Kind() != reflect.Uint8 {
		return time.Time{}, fmt.Errorf("unexpected value type %q, expecting byte array", r)
	}
	if l := t.Len(); l < 8 || l > 96 {
		return time.Time{}, fmt.Errorf("unexpected byte array length %d", l)
	}
	// Convert the array to a slice value.
	ra := reflect.New(t).Elem()
	ra.Set(r)
	p := ra.Slice(0, t.Len()).Interface().([]byte)
	// Convert the byte slice to a time.Time value.
	// The following code was taken from https://stackoverflow.com/a/53133964
	// and was written by https://stackoverflow.com/users/1912391/zaky.
	nano, dt := binary.LittleEndian.Uint64(p[:8]), binary.LittleEndian.Uint32(p[8:])
	l := dt + 68569
	n := 4 * l / 146097
	l = l - (146097*n+3)/4
	i := 4000 * (l + 1) / 1461001
	l = l - 1461*i/4 + 31
	j := 80 * l / 2447
	k := l - 2447*j/80
	l = j / 11
	j = j + 2 - 12*l
	i = 100*(n-49) + i + l
	tm := time.Date(int(i), time.Month(j), int(k), 0, 0, 0, 0, time.UTC)
	return tm.Add(time.Duration(nano)), nil
}
