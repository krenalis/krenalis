//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package parquet implements the Parquet connector.
// (https://github.com/apache/parquet-format)
package parquet

import (
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"

	goparquet "github.com/fraugster/parquet-go"
	"github.com/fraugster/parquet-go/parquet"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterFile(meergo.FileInfo{
		Name:      "Parquet",
		Icon:      icon,
		Extension: "parquet",
		AsSource:  &meergo.AsSourceFile{},
	}, New)
}

// New returns a new Parquet connector instance.
func New(conf *meergo.FileConfig) (*Parquet, error) {
	return &Parquet{}, nil
}

type Parquet struct{}

// Read reads the records from r and writes them to records.
func (pq *Parquet) Read(ctx context.Context, r io.Reader, sheet string, records meergo.RecordWriter) error {

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
	_, err = fi.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	fr, err := goparquet.NewFileReaderWithOptions(fi, goparquet.WithReaderContext(ctx))
	if err != nil {
		return err
	}

	// Read the columns.
	var int96Columns []string
	parquetColumns := fr.Columns()
	columns := make([]types.Property, 0, len(parquetColumns))
	for _, c := range parquetColumns {
		element := c.Element()
		typ, err := propertyType(element)
		if err != nil {
			return err
		}
		if !typ.Valid() {
			continue
		}
		name := strings.Join(c.Path(), ".")
		if *element.Type == parquet.Type_INT96 {
			int96Columns = append(int96Columns, name)
		}
		columns = append(columns, types.Property{
			Name:     name,
			Type:     typ,
			Nullable: true,
		})
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
			if v, ok := record[name]; ok {
				record[name], err = convertInt96(v)
				if err != nil {
					return fmt.Errorf("cannot convert value of column %q: %s", name, err)
				}
			}
		}
		// Add fields with a nil value.
		for _, c := range columns {
			if _, ok := record[c.Name]; !ok {
				record[c.Name] = nil
			}
		}
		// Write the record.
		err = records.Record(record)
		if err != nil {
			return err
		}
	}

	return nil
}

// propertyType returns the type of the Parquet column specified by the given
// SchemaElement. If the property type is not supported, it returns an invalid type.
// (https://github.com/apache/parquet-format).
func propertyType(elem *parquet.SchemaElement) (types.Type, error) {

	if elem.Type == nil {
		return types.Type{}, errors.New("unexpected Parquet nil type")
	}

	// Kinds.
	switch *elem.Type {
	case parquet.Type_BOOLEAN:
		return types.Boolean(), nil
	case parquet.Type_FLOAT:
		return types.Float(32), nil
	case parquet.Type_DOUBLE:
		return types.Float(64), nil
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
			return types.Date(), nil
		}
		if lt.TIMESTAMP != nil {
			return types.DateTime(), nil
		}
		if lt.TIME != nil {
			return types.Time(), nil
		}
		if lt.INTEGER != nil {
			if lt.INTEGER.IsSigned {
				switch lt.INTEGER.BitWidth {
				case 8:
					return types.Int(8), nil
				case 16:
					return types.Int(16), nil
				case 32:
					return types.Int(32), nil
				case 64:
					return types.Int(64), nil
				}
				return types.Type{}, fmt.Errorf("unexpected Parquet bitWidth value: %d", lt.INTEGER.BitWidth)
			}
			switch lt.INTEGER.BitWidth {
			case 8:
				return types.Uint(8), nil
			case 16:
				return types.Uint(16), nil
			case 32:
				return types.Uint(32), nil
			case 64:
				return types.Uint(64), nil
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
			return types.Int(8), nil
		case parquet.ConvertedType_INT_16:
			return types.Int(16), nil
		case parquet.ConvertedType_INT_32:
			return types.Int(32), nil
		case parquet.ConvertedType_INT_64:
			return types.Int(64), nil
		case parquet.ConvertedType_UINT_8:
			return types.Uint(8), nil
		case parquet.ConvertedType_UINT_16:
			return types.Uint(16), nil
		case parquet.ConvertedType_UINT_32:
			return types.Uint(32), nil
		case parquet.ConvertedType_UINT_64:
			return types.Uint(64), nil
		case parquet.ConvertedType_JSON, parquet.ConvertedType_BSON:
			return types.JSON(), nil
		case parquet.ConvertedType_DECIMAL:
			if elem.Precision != nil && *elem.Precision <= types.MaxDecimalPrecision &&
				elem.Scale != nil && *elem.Scale <= types.MaxDecimalScale && *elem.Scale <= *elem.Precision {
				return types.Decimal(int(*elem.Precision), int(*elem.Scale)), nil
			}
			return types.Decimal(0, 0), nil
		case parquet.ConvertedType_DATE:
			return types.Date(), nil
		case parquet.ConvertedType_TIMESTAMP_MICROS, parquet.ConvertedType_TIMESTAMP_MILLIS:
			return types.DateTime(), nil
		case parquet.ConvertedType_TIME_MICROS, parquet.ConvertedType_TIME_MILLIS:
			return types.Time(), nil
		}
		return types.Type{}, fmt.Errorf("unsupported converted Parquet type %q", *ct)
	}

	// Kinds.
	switch *elem.Type {
	case parquet.Type_INT32:
		return types.Int(32), nil
	case parquet.Type_INT64:
		return types.Int(64), nil
	case parquet.Type_INT96:
		return types.DateTime(), nil
	case parquet.Type_BYTE_ARRAY, parquet.Type_FIXED_LEN_BYTE_ARRAY:
		return types.Text(), nil
	}

	return types.Type{}, nil
}

// Convert an int96 type value to a time.Time value.
// v must be a byte array with length in range [8,96].
// See https://stackoverflow.com/questions/53103762.
func convertInt96(v any) (time.Time, error) {
	r := reflect.ValueOf(v)
	t := r.Type()
	// Validate the argument.
	if t.Kind() != reflect.Array || t.Elem().Kind() != reflect.Uint8 {
		return time.Time{}, fmt.Errorf("expected byte array, got value type %q", r)
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
