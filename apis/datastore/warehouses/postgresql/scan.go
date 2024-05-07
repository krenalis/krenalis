//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"reflect"
	"strings"
	"time"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
)

var errPostgreSQLInvalidData = errors.New("PostgreSQL has returned invalid data")

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	column      types.Property
	rows        *[][]any
	columnIndex int
	columnCount int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(columns []types.Property, rows *[][]any) []any {
	values := make([]any, len(columns))
	for i, c := range columns {
		values[i] = scanValue{
			column:      c,
			rows:        rows,
			columnIndex: i,
			columnCount: len(columns),
		}
	}
	return values
}

func (sv scanValue) Scan(src any) error {
	c := sv.column
	if src != nil && c.Type.Kind() == types.ArrayKind {
		var err error
		src, err = sv.scanArray(src)
		if err != nil {
			return err
		}
	}
	value, err := normalize(c.Name, c.Type, src, c.Nullable)
	if err != nil {
		return err
	}
	var row []any
	if sv.columnIndex == 0 {
		row = make([]any, sv.columnCount)
		*sv.rows = append(*sv.rows, row)
	} else {
		row = (*sv.rows)[len(*sv.rows)-1]
	}
	row[sv.columnIndex] = value
	return nil
}

// scanArray scans an array and returns the values.
func (sv scanValue) scanArray(src any) (any, error) {
	data, ok := src.([]byte)
	if !ok {
		return nil, errors.New("PostgreSQL has returned an unexpected value type for an array")
	}
	p := 12
	if len(data) < p {
		return nil, errPostgreSQLInvalidData
	}
	dimensions := int(int32(binary.BigEndian.Uint32(data[:4])))
	if dimensions > 1 {
		return nil, errors.New("type is not supported")
	}
	oid := binary.BigEndian.Uint32(data[8 : 8+4])
	size := 0
	if dimensions > 0 {
		if len(data) < 12+8 {
			return nil, errPostgreSQLInvalidData
		}
		size = int(int32(binary.BigEndian.Uint32(data[12 : 12+4])))
		p += 8
	}
	values := make([]any, size)
	switch oid {
	case 16: // bool
		if len(data) < p+5*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			p += 4 // skip the length
			values[i] = data[p] == 1
			p += 1
		}
	case
		20, // int8
		21, // in2
		23: // int4
		if len(data) < p+2*size {
			return nil, errPostgreSQLInvalidData
		}
		var v int
		for i := range values {
			if len(data) < p+4 {
				return nil, errPostgreSQLInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			p += 4
			if len(data) < p+l {
				return nil, errPostgreSQLInvalidData
			}
			switch l {
			case 2:
				v = int(int16(binary.BigEndian.Uint16(data[p : p+2])))
			case 4:
				v = int(int32(binary.BigEndian.Uint32(data[p : p+4])))
			case 8:
				v = int(int64(binary.BigEndian.Uint64(data[p : p+8])))
			default:
				return nil, errPostgreSQLInvalidData
			}
			values[i] = v
			p += l
		}
	case
		700, // float4
		701: // float8
		if len(data) < p+4*size {
			return nil, errPostgreSQLInvalidData
		}
		var v float64
		for i := range values {
			if len(data) < p+4 {
				return nil, errPostgreSQLInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			p += 4
			if len(data) < p+l {
				return nil, errPostgreSQLInvalidData
			}
			switch l {
			case 4:
				v = float64(math.Float32frombits(binary.BigEndian.Uint32(data[p : p+4])))
			case 8:
				v = math.Float64frombits(binary.BigEndian.Uint64(data[p : p+8]))
			default:
				return nil, errPostgreSQLInvalidData
			}
			values[i] = v
			p += l
		}
	case 869: // inet
		if len(data) < p+12*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			if len(data) < p+4 {
				return nil, errPostgreSQLInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l != 8 && l != 20 {
				return nil, errPostgreSQLInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errPostgreSQLInvalidData
			}
			addr, ok := netip.AddrFromSlice(data[p+4 : p+l])
			if !ok {
				return nil, errPostgreSQLInvalidData
			}
			values[i] = addr.String()
			p += l
		}
	case
		25,   // text
		1042, // bpchar
		1043: // varchar
		if len(data) < p+5*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			if len(data) < p+4 {
				return nil, errPostgreSQLInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l <= 0 {
				return nil, errPostgreSQLInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errPostgreSQLInvalidData
			}
			values[i] = string(data[p : p+l])
			p += l
		}
	case 1082: // date
		if len(data) < p+8*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			p += 4 // skip length
			days := int(int32(binary.BigEndian.Uint32(data[p : p+4])))
			values[i] = time.Date(2000, 1, 1+days, 0, 0, 0, 0, time.UTC)
			p += 4
		}
	case 1083: // time
		if len(data) < p+12*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			p += 4 // skip length
			us := time.Duration(int64(binary.BigEndian.Uint64(data[p:p+8]))) * time.Microsecond
			values[i] = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Add(us)
			p += 8
		}
	case 1114: // timestamp
		if len(data) < p+12*size {
			return nil, errPostgreSQLInvalidData
		}
		epoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // PostgreSQL epoch
		for i := range values {
			p += 4 // skip length
			us := time.Duration(int64(binary.BigEndian.Uint64(data[p:p+8]))) * time.Microsecond
			values[i] = epoch.Add(us)
			p += 8
		}
	case 2950: // uuid
		if len(data) < p+20*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			p += 4 // skip length
			values[i] = uuid.Must(uuid.FromBytes(data[p : p+16])).String()
			p += 16
		}
	case 114: // json
		if len(data) < p+4*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			if len(data) < p+4 {
				return nil, errPostgreSQLInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l <= 0 {
				return nil, errPostgreSQLInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errPostgreSQLInvalidData
			}
			v := make([]byte, l)
			copy(v, data[p:p+l])
			values[i] = v
			p += l
		}
	case 3802: // jsonb
		if len(data) < p+5*size {
			return nil, errPostgreSQLInvalidData
		}
		for i := range values {
			if len(data) < p+4 {
				return nil, errPostgreSQLInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l <= 0 {
				return nil, errPostgreSQLInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errPostgreSQLInvalidData
			}
			if data[p] != 1 {
				return nil, errPostgreSQLInvalidData
			}
			v := make([]byte, l-1)
			copy(v, data[p+1:p+l])
			values[i] = v
			p += l
		}
	default:
		return nil, errors.New("unsupported type")
	}
	return values, nil
}

// normalize normalizes a value returned by PostgreSQL and returns its
// normalized form. If the value is not valid it returns an error.
func normalize(name string, typ types.Type, v any, nullable bool) (any, error) {
	if v == nil {
		if !nullable {
			return nil, fmt.Errorf("column %s is non-nullable, but PostgreSQL returned a NULL value", name)
		}
		return nil, nil
	}
	switch typ.Kind() {
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.IntKind:
		if v, ok := v.(int64); ok {
			return warehouses.ValidateInt(name, typ, int(v))
		}
	case types.FloatKind:
		if v, ok := v.(float64); ok {
			return warehouses.ValidateFloat(name, typ, v)
		}
	case types.DecimalKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateDecimalString(name, typ, v)
		}
	case types.DateTimeKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDateTime(name, v)
		}
	case types.DateKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDate(name, v)
		}
	case types.TimeKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateTimeString(name, "15:04:05.999999", v)
		}
	case types.UUIDKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateUUID(name, v)
		}
	case types.JSONKind:
		// Go type is string for both the PostgreSQL types "json" and "jsonb".
		if v, ok := v.(string); ok {
			return warehouses.ValidateJSONRaw(name, []byte(v))
		}
	case types.InetKind:
		if v, ok := v.(string); ok {
			// IP addresses are parsed directly here, without calling the
			// validation function inside warehouses, because the IP addresses
			// returned by PostgreSQL include the subnet mask, which must be
			// removed.
			rawIP, _, _ := strings.Cut(v, "/") // "127.0.0.1/32" -> "127.0.0.1"
			ip, err := netip.ParseAddr(rawIP)
			if err != nil {
				return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not an Inet type", v, name)
			}
			return ip.String(), nil
		}
	case types.TextKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateText(name, typ, v)
		}
	case types.ArrayKind:
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Slice {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		n := rv.Len()
		if n < typ.MinItems() || n > typ.MaxItems() {
			return nil, fmt.Errorf("data warehouse returned an array with %d items for column %s, which is not within the expected range of [%d, %d]",
				n, name, typ.MinItems(), typ.MaxItems())
		}
		a := make([]any, n)
		t := typ.Elem()
		var err error
		for i := 0; i < n; i++ {
			e := rv.Index(i).Interface()
			a[i], err = normalize(name, t, e, false)
			if err != nil {
				return nil, err
			}
		}
		return a, nil
	}
	return nil, fmt.Errorf("PostgreSQL has returned an unsupported type %T for column %s", v, name)
}
