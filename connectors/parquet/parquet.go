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
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	goparquet "github.com/fraugster/parquet-go"
	"github.com/fraugster/parquet-go/parquet"
	"github.com/fraugster/parquet-go/parquetschema"
	"github.com/google/uuid"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterFile(meergo.FileInfo{
		Name:          "Parquet",
		Icon:          icon,
		Extension:     "parquet",
		AsSource:      &meergo.AsSourceFile{},
		AsDestination: &meergo.AsDestinationFile{},
	}, New)
}

// New returns a new Parquet connector instance.
func New(conf *meergo.FileConfig) (*Parquet, error) {
	return &Parquet{}, nil
}

type Parquet struct{}

// ContentType returns the content type of the file.
func (pq *Parquet) ContentType(ctx context.Context) string {
	return "application/vnd.apache.parquet"
}

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
	type unitColumn struct {
		name string
		unit *parquet.TimeUnit
	}
	type decimalColumn struct {
		name   string
		phType parquet.Type
		prec   int
		scale  int
	}
	var decimalColumns []decimalColumn
	var dateColumns []string
	var timeColumns []unitColumn
	var int64TimestampColumns []unitColumn
	var int96Columns []string
	parquetColumns := fr.Columns()
	columns := make([]types.Property, 0, len(parquetColumns))
	for _, c := range parquetColumns {
		var pathFirstComponent string
		if len(c.Path()) > 0 {
			// I don't know if this check is necessary, but just in case, since
			// the library doesn't document whether path should always contain a
			// component or not, it seems preferable to do this check rather
			// than risk a panic in the connector.
			pathFirstComponent = c.Path()[0]
		}
		if len(c.Path()) > 1 {
			// Skip columns referring to groups and arrays (and possibly also to
			// other composite types in Parquet), which are not currently
			// imported correctly.
			//
			// TODO: see the issues:
			//  - https://github.com/meergo/meergo/issues/1369 (groups)
			//  - https://github.com/meergo/meergo/issues/1325 (arrays)
			records.Issue("Column %q is not supported because column groups (which are used to represent types as object, arrays, and maps) are not supported.", strings.Join(c.Path(), "."))
			continue
		}
		element := c.Element()
		typ, err := propertyType(element)
		if err != nil {
			return err
		}
		if !typ.Valid() {
			records.Issue("Column %q has an unsupported type.", pathFirstComponent)
			continue
		}
		if !types.IsValidPropertyName(pathFirstComponent) {
			// TODO: see https://github.com/meergo/meergo/issues/1405.
			records.Issue("Column %q does not have a valid property name. Valid names start with a letter or underscore, followed by only letters, numbers, or underscores.", pathFirstComponent)
			continue
		}
		name := strings.Join(c.Path(), ".")
		if *element.Type == parquet.Type_INT96 {
			int96Columns = append(int96Columns, name)
		}
		if *element.Type == parquet.Type_INT64 && element.LogicalType != nil &&
			element.LogicalType.TIMESTAMP != nil {
			int64TimestampColumns = append(int64TimestampColumns, unitColumn{
				name: name,
				unit: element.LogicalType.TIMESTAMP.Unit,
			})
		}
		if *element.Type == parquet.Type_INT32 && element.LogicalType != nil && element.LogicalType.DATE != nil {
			dateColumns = append(dateColumns, name)
		}
		if element.LogicalType != nil && element.LogicalType.TIME != nil {
			timeColumns = append(timeColumns, unitColumn{
				name: name,
				unit: element.LogicalType.TIME.Unit,
			})
		}
		if lt := element.LogicalType; lt != nil && lt.DECIMAL != nil {
			switch *element.Type {
			case
				parquet.Type_INT32,
				parquet.Type_INT64,
				parquet.Type_BYTE_ARRAY,
				parquet.Type_FIXED_LEN_BYTE_ARRAY:
				decimalColumns = append(decimalColumns, decimalColumn{
					name:   name,
					prec:   typ.Precision(),
					scale:  typ.Scale(),
					phType: *element.Type,
				})
			}
		}
		if ct := element.ConvertedType; ct != nil && *ct == parquet.ConvertedType_DECIMAL {
			switch *element.Type {
			case
				parquet.Type_INT32,
				parquet.Type_INT64,
				parquet.Type_BYTE_ARRAY,
				parquet.Type_FIXED_LEN_BYTE_ARRAY:
				decimalColumns = append(decimalColumns, decimalColumn{
					name:   name,
					prec:   typ.Precision(),
					scale:  typ.Scale(),
					phType: *element.Type,
				})
			}
		}
		if ct := element.ConvertedType; ct != nil {
			unit := parquet.NewTimeUnit()
			switch *ct {
			case parquet.ConvertedType_TIME_MILLIS:
				unit.MILLIS = parquet.NewMilliSeconds()
				timeColumns = append(timeColumns, unitColumn{
					name: name,
					unit: unit,
				})
			case parquet.ConvertedType_TIME_MICROS:
				unit.MICROS = parquet.NewMicroSeconds()
				timeColumns = append(timeColumns, unitColumn{
					name: name,
					unit: unit,
				})
			}
		}
		var nullable bool
		if rep := element.RepetitionType; rep != nil && *rep == parquet.FieldRepetitionType_OPTIONAL {
			nullable = true
		}
		columns = append(columns, types.Property{
			Name:     name,
			Type:     typ,
			Nullable: nullable,
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
		// Convert int64 type values representing timestamps from int64 to
		// time.Time.
		for _, column := range int64TimestampColumns {
			if v, ok := record[column.name].(int64); ok {
				record[column.name] = int64ToTimeTime(v, column.unit)
			}
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
		// Convert DATE values (int32, representing the number of days since
		// 1970-01-01) to time.Time values.
		for _, name := range dateColumns {
			if v, ok := record[name].(int32); ok {
				record[name] = time.Unix(int64(v)*3600*24, 0).UTC()
			}
		}
		// Convert TIME, TIME_MILLIS and TIME_MICROS values to time.Time values.
		for _, column := range timeColumns {
			if column.unit == nil {
				continue
			}
			if column.unit.MILLIS != nil {
				if milli, ok := record[column.name].(int32); ok {
					record[column.name] = time.UnixMilli(int64(milli)).UTC()
					continue
				}
			}
			if column.unit.MICROS != nil {
				if micro, ok := record[column.name].(int64); ok {
					record[column.name] = time.UnixMicro(micro).UTC()
					continue
				}
			}
			if column.unit.NANOS != nil {
				if nano, ok := record[column.name].(int64); ok {
					record[column.name] = time.Unix(0, nano).UTC()
					continue
				}
			}
		}
		for _, column := range decimalColumns {
			switch column.phType {
			case parquet.Type_INT32:
				if i32, ok := record[column.name].(int32); ok {
					record[column.name] = decimal.New(int64(i32), column.scale)
					continue
				}
			case parquet.Type_INT64:
				if i64, ok := record[column.name].(int64); ok {
					record[column.name] = decimal.New(i64, column.scale)
					continue
				}
			case
				parquet.Type_BYTE_ARRAY,
				parquet.Type_FIXED_LEN_BYTE_ARRAY:
				if bytes, ok := record[column.name].([]byte); ok {
					dec, err := decimal.Binary(bytes, column.prec, column.scale)
					if err != nil {
						return fmt.Errorf("invalid decimal value: %s", err)
					}
					record[column.name] = dec
					continue
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

// Write writes to w the records read from records.
func (pq *Parquet) Write(ctx context.Context, w io.Writer, sheet string, records meergo.RecordReader) error {
	columns := records.Columns()
	schema := types.Object(columns)
	schemaDef, err := schemaToParquetSchema(schema)
	if err != nil {
		return err
	}
	fw := goparquet.NewFileWriter(w,
		goparquet.WithCreator("Meergo"),
		goparquet.WithSchemaDefinition(schemaDef),
	)
	for {
		id, record, err := records.Record(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		data, err := convertToParquetData(schema, record)
		if err != nil {
			records.Ack(id, err)
			continue
		}
		err = fw.AddData(data)
		if err != nil {
			return err
		}
		records.Ack(id, nil)
	}
	err = fw.Close()
	if err != nil {
		return err
	}
	return nil

}

// Convert an int96 type value to a time.Time value.
// v must be a byte array with length 12.
// See https://stackoverflow.com/questions/53103762.
func convertInt96(v any) (time.Time, error) {
	r := reflect.ValueOf(v)
	t := r.Type()
	// Validate the argument.
	if t.Kind() != reflect.Array || t.Elem().Kind() != reflect.Uint8 {
		return time.Time{}, fmt.Errorf("expected byte array, got value type %q", t)
	}
	if l := t.Len(); l != 12 {
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

// convertToParquetData converts the records passed by Meergo into the format
// required by the Parquet library for export to file.
func convertToParquetData(schema types.Type, record map[string]any) (map[string]any, error) {
	converted := make(map[string]any, len(record))
	for _, p := range schema.Properties() {
		switch p.Type.Kind() {
		case types.IntKind:
			if p.Type.BitSize() <= 32 {
				if i64, ok := record[p.Name].(int); ok {
					converted[p.Name] = int32(i64)
					continue
				}
			} else {
				if i64, ok := record[p.Name].(int); ok {
					converted[p.Name] = int64(i64)
					continue
				}
			}
		case types.UintKind:
			if p.Type.BitSize() <= 32 {
				if u64, ok := record[p.Name].(uint); ok {
					converted[p.Name] = int32(u64)
					continue
				}
			} else {
				if u64, ok := record[p.Name].(uint); ok {
					converted[p.Name] = int64(u64)
					continue
				}
			}
		case types.FloatKind:
			if p.Type.BitSize() == 32 {
				if f64, ok := record[p.Name].(float64); ok {
					converted[p.Name] = float32(f64)
					continue
				}
			}
		case types.DecimalKind:
			dec, ok := record[p.Name].(decimal.Decimal)
			if !ok {
				continue
			}
			switch prec := p.Type.Precision(); {
			case prec <= 9:
				i64, ok := decimalToInt64(dec, p.Type.Scale())
				if !ok {
					// This never happens except for out of scale values read
					// from the data warehouse, because the type chosen for
					// export to Parquet is chosen based on the decimal
					// precision and scale to represent all decimal values
					// allowed for p.Type. We return error here to avoid strange
					// errors in the Parquet library.
					return nil, fmt.Errorf("decimal value read from Meergo cannot be represented with Parquet's INT32")
				}
				converted[p.Name] = int32(i64)
			case 10 <= prec && prec <= 18:
				i64, ok := decimalToInt64(dec, p.Type.Scale())
				if !ok {
					// This should never happen, see the comment above for more
					// details.
					return nil, fmt.Errorf("decimal value read from Meergo cannot be represented with Parquet's INT64")
				}
				converted[p.Name] = i64
			default:
				bytes, err := dec.Binary(p.Type.Scale())
				if err != nil {
					return nil, fmt.Errorf("cannot convert decimal value read from Meergo to Parquet binary representation: %s", err)
				}
				converted[p.Name] = bytes
			}
			continue
		case types.DateTimeKind:
			if ts, ok := record[p.Name].(time.Time); ok {
				ts, err := timeTimeToInt64(ts)
				if err != nil {
					return nil, errors.New("timestamp out of range")
				}
				converted[p.Name] = ts
				continue
			}
		case types.DateKind:
			if ts, ok := record[p.Name].(time.Time); ok {
				converted[p.Name] = int32(ts.Unix() / 3_600 / 24)
				continue
			}
		case types.TimeKind:
			if ts, ok := record[p.Name].(time.Time); ok {
				// Time values are exported with microseconds precision instead
				// of nanoseconds for this reason:
				// https://github.com/meergo/meergo/issues/1392.
				converted[p.Name] = ts.UnixMicro()
				continue
			}
		case types.YearKind:
			if y, ok := record[p.Name].(int); ok {
				converted[p.Name] = int32(y)
				continue
			}
		case types.UUIDKind:
			if u, ok := record[p.Name].(string); ok {
				array := [16]byte(uuid.MustParse(u))
				converted[p.Name] = array[:]
				continue
			}
		case types.JSONKind:
			if jsonValue, ok := record[p.Name].(json.Value); ok {
				converted[p.Name] = []byte(jsonValue)
				continue
			}
		case types.ObjectKind:
			obj, ok := record[p.Name].(map[string]any)
			if !ok {
				continue
			}
			var err error
			converted[p.Name], err = convertToParquetData(p.Type, obj)
			if err != nil {
				return nil, err
			}
			continue
		}
		converted[p.Name] = record[p.Name]
	}
	return converted, nil
}

// decimalToInt64 converts the decimal d to an "unscaled int" using the given
// scale, in the format expected by Parquet.
//
// The returned boolean reports whether d can be represented exactly with the
// provided scale.
func decimalToInt64(d decimal.Decimal, scale int) (int64, bool) {
	bytes, err := d.Binary(scale)
	if err != nil {
		return 0, false
	}
	u64 := binary.BigEndian.Uint64(bytes)
	return int64(u64), true
}

// determineDecimalType determines the Meergo decimal type based on available
// Parquet type information. If the decimal type cannot be determined, or the
// determined type is not valid in Meergo, types.Type{} and false are returned.
func determineDecimalType(precision, scale int, elem *parquet.SchemaElement) (types.Type, bool) {
	if precision == 0 && scale == 0 {
		switch *elem.Type {
		case parquet.Type_INT32:
			precision = 10 // Length of max int32.
		case parquet.Type_INT64:
			precision = 19 // Length of max int64.
		}
	}
	if precision < 1 || scale < 0 || scale > precision || precision > types.MaxDecimalPrecision || scale > types.MaxDecimalScale {
		return types.Type{}, false
	}
	return types.Decimal(precision, scale), true
}

// int64ToTimeTime converts an int64 timestamp, read from Parquet, to a
// time.Time. unit is the timestamp unit; if nil, it is considered nanoseconds.
func int64ToTimeTime(v int64, unit *parquet.TimeUnit) time.Time {
	if unit != nil && unit.IsSetMILLIS() {
		return time.UnixMilli(v).UTC()
	}
	if unit != nil && unit.IsSetMICROS() {
		return time.UnixMicro(v).UTC()
	}
	return time.Unix(0, v).UTC()
}

// schemaToParquetSchema returns the Parquet schema definition corresponding to
// the given Meergo schema.
//
// This method panics if schema is not an Object.
func schemaToParquetSchema(schema types.Type) (*parquetschema.SchemaDefinition, error) {
	columns, err := objectToColumns(schema)
	if err != nil {
		return nil, err
	}
	return &parquetschema.SchemaDefinition{
		RootColumn: &parquetschema.ColumnDefinition{
			Children: columns,
			// According to the documentation this is not necessary, but the
			// module panics if it is not set this way:
			SchemaElement: parquet.NewSchemaElement(),
		},
	}, nil
}

// objectToColumns returns the Parquet column definitions corresponding to the
// given Meergo object.
//
// This method panics if obj is not an Object.
func objectToColumns(obj types.Type) ([]*parquetschema.ColumnDefinition, error) {
	columns := []*parquetschema.ColumnDefinition{}
	for _, property := range obj.Properties() {

		// Create the column corresponding to the property.
		col := &parquetschema.ColumnDefinition{}
		col.SchemaElement = parquet.NewSchemaElement()
		col.SchemaElement.Name = property.Name

		// Set the column as optional.
		col.SchemaElement.RepetitionType = parquet.FieldRepetitionTypePtr(
			parquet.FieldRepetitionType_OPTIONAL)

		// Handle objects.
		columns = append(columns, col)
		if property.Type.Kind() == types.ObjectKind {
			var err error
			col.Children, err = objectToColumns(property.Type)
			if err != nil {
				return nil, err
			}
			continue
		}

		// Set the column type.
		switch property.Type.Kind() {
		case types.BooleanKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_BOOLEAN)
		case types.IntKind:
			switch property.Type.BitSize() {
			case 8:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
				col.SchemaElement.LogicalType = parquet.NewLogicalType()
				col.SchemaElement.LogicalType.INTEGER = parquet.NewIntType()
				col.SchemaElement.LogicalType.INTEGER.BitWidth = 8
				col.SchemaElement.LogicalType.INTEGER.IsSigned = true
			case 16:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
				col.SchemaElement.LogicalType = parquet.NewLogicalType()
				col.SchemaElement.LogicalType.INTEGER = parquet.NewIntType()
				col.SchemaElement.LogicalType.INTEGER.BitWidth = 16
				col.SchemaElement.LogicalType.INTEGER.IsSigned = true
			case 24:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
			case 32:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
			case 64:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT64)
			}
		case types.UintKind:
			switch property.Type.BitSize() {
			case 8:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
				col.SchemaElement.LogicalType = parquet.NewLogicalType()
				col.SchemaElement.LogicalType.INTEGER = parquet.NewIntType()
				col.SchemaElement.LogicalType.INTEGER.BitWidth = 8
				col.SchemaElement.LogicalType.INTEGER.IsSigned = false
			case 16:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
				col.SchemaElement.LogicalType = parquet.NewLogicalType()
				col.SchemaElement.LogicalType.INTEGER = parquet.NewIntType()
				col.SchemaElement.LogicalType.INTEGER.BitWidth = 16
				col.SchemaElement.LogicalType.INTEGER.IsSigned = false
			case 24:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
				col.SchemaElement.LogicalType = parquet.NewLogicalType()
				col.SchemaElement.LogicalType.INTEGER = parquet.NewIntType()
				col.SchemaElement.LogicalType.INTEGER.BitWidth = 32
				col.SchemaElement.LogicalType.INTEGER.IsSigned = false
			case 32:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
				col.SchemaElement.LogicalType = parquet.NewLogicalType()
				col.SchemaElement.LogicalType.INTEGER = parquet.NewIntType()
				col.SchemaElement.LogicalType.INTEGER.BitWidth = 32
				col.SchemaElement.LogicalType.INTEGER.IsSigned = false
			case 64:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT64)
				col.SchemaElement.LogicalType = parquet.NewLogicalType()
				col.SchemaElement.LogicalType.INTEGER = parquet.NewIntType()
				col.SchemaElement.LogicalType.INTEGER.BitWidth = 64
				col.SchemaElement.LogicalType.INTEGER.IsSigned = false
			}
		case types.FloatKind:
			switch property.Type.BitSize() {
			case 32:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_FLOAT)
			case 64:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_DOUBLE)
			}
		case types.DecimalKind:
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.DECIMAL = parquet.NewDecimalType()
			col.SchemaElement.LogicalType.DECIMAL.Precision = int32(property.Type.Precision())
			col.SchemaElement.LogicalType.DECIMAL.Scale = int32(property.Type.Scale())
			switch p := property.Type.Precision(); {
			case p <= 9:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
			case 10 <= p && p <= 18:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT64)
			default:
				col.SchemaElement.Type = parquet.TypePtr(parquet.Type_BYTE_ARRAY)
			}
		case types.DateTimeKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT64)
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.TIMESTAMP = parquet.NewTimestampType()
			col.SchemaElement.LogicalType.TIMESTAMP.IsAdjustedToUTC = true
			col.SchemaElement.LogicalType.TIMESTAMP.Unit = parquet.NewTimeUnit()
			col.SchemaElement.LogicalType.TIMESTAMP.Unit.NANOS = parquet.NewNanoSeconds()
		case types.DateKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.DATE = parquet.NewDateType()
		case types.TimeKind:
			// Time values are exported with microseconds precision instead of
			// nanoseconds for this reason:
			// https://github.com/meergo/meergo/issues/1392.
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT64)
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.TIME = parquet.NewTimeType()
			col.SchemaElement.LogicalType.TIME.IsAdjustedToUTC = true
			col.SchemaElement.LogicalType.TIME.Unit = parquet.NewTimeUnit()
			col.SchemaElement.LogicalType.TIME.Unit.MICROS = parquet.NewMicroSeconds()
		case types.YearKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_INT32)
		case types.UUIDKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_FIXED_LEN_BYTE_ARRAY)
			typeLength := int32(16)
			col.SchemaElement.TypeLength = &typeLength
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.UUID = parquet.NewUUIDType()
		case types.JSONKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_BYTE_ARRAY)
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.JSON = parquet.NewJsonType()
		case types.InetKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_BYTE_ARRAY)
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.STRING = parquet.NewStringType()
		case types.TextKind:
			col.SchemaElement.Type = parquet.TypePtr(parquet.Type_BYTE_ARRAY)
			col.SchemaElement.LogicalType = parquet.NewLogicalType()
			col.SchemaElement.LogicalType.STRING = parquet.NewStringType()
		case types.ArrayKind:
			return nil, errors.New("array properties are not supported") // TODO: see the issue https://github.com/meergo/meergo/issues/1325.
		case types.MapKind:
			return nil, errors.New("map properties are not supported") // TODO: see the issue https://github.com/meergo/meergo/issues/1371.
		}
	}
	return columns, nil
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
			typ, ok := determineDecimalType(int(d.Precision), int(d.Scale), elem)
			if !ok {
				return types.Type{}, nil
			}
			return typ, nil
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
				return types.Type{}, nil
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
			return types.Type{}, nil
		}
		if lt.JSON != nil {
			return types.JSON(), nil
		}
		if lt.BSON != nil {
			return types.Type{}, nil // TODO: see https://github.com/meergo/meergo/issues/1400.
		}
		if lt.UUID != nil {
			return types.UUID(), nil
		}
		return types.Type{}, nil
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
		case parquet.ConvertedType_JSON:
			return types.JSON(), nil
		case parquet.ConvertedType_BSON:
			return types.Type{}, nil // TODO: see https://github.com/meergo/meergo/issues/1400.
		case parquet.ConvertedType_DECIMAL:
			var prec, scale int
			if elem.Precision != nil {
				prec = int(*elem.Precision)
			}
			if elem.Scale != nil {
				scale = int(*elem.Scale)
			}
			typ, ok := determineDecimalType(prec, scale, elem)
			if !ok {
				return types.Type{}, nil
			}
			return typ, nil

		case parquet.ConvertedType_DATE:
			return types.Date(), nil
		case parquet.ConvertedType_TIMESTAMP_MICROS, parquet.ConvertedType_TIMESTAMP_MILLIS:
			// TODO: https://github.com/meergo/meergo/issues/1385
			return types.Type{}, nil
		case parquet.ConvertedType_TIME_MICROS, parquet.ConvertedType_TIME_MILLIS:
			return types.Time(), nil
		}
		return types.Type{}, nil
	}

	// Kinds.
	switch *elem.Type {
	case parquet.Type_INT32:
		return types.Int(32), nil
	case parquet.Type_INT64:
		return types.Int(64), nil
	case parquet.Type_INT96:
		// Parquet columns with physical type INT96 are treated as 'datetime' in
		// Meergo. This is because there does not seem to be any other practical
		// use, in fact, for such columns. Also, consider that INT96 types are
		// indeed deprecated, as timestamps are defined with other types, an
		// they are kept here in the connector on import only, for compatibility
		// with old Parquet files.
		return types.DateTime(), nil
	case parquet.Type_BYTE_ARRAY, parquet.Type_FIXED_LEN_BYTE_ARRAY:
		return types.Text(), nil
	}

	return types.Type{}, nil
}

// timeTimeToInt64 returns the int64 representation of the given time.Time
// value, that can be written to Parquet. The int64 has unit nanoseconds. If the
// year of ts is less than 1678, or it is greater than 2262, this function
// returns error.
func timeTimeToInt64(ts time.Time) (int64, error) {
	if y := ts.Year(); y < 1678 || y > 2262 {
		return 0, fmt.Errorf("timestamp year is out of range")
	}
	return ts.UnixNano(), nil
}
