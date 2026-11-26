// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// maxArrayScanBytes limits the combined size of an array's binary payload and
// result slice, counting 16 bytes per result element.
const maxArrayScanBytes = 16 << 20

var errInvalidData = errors.New("PostgreSQL has returned invalid data")

// scanner implements the warehouses.Rows interface to read and normalize the
// rows read from PostgreSQL.
type scanner struct {
	columns []warehouses.Column
	rows    pgx.Rows
	values  []any
	dest    []any
	index   int
}

// newScanner returns a new scanner.
func newScanner(columns []warehouses.Column, rows pgx.Rows) *scanner {
	s := &scanner{
		columns: columns,
		rows:    rows,
	}
	s.values = make([]any, len(columns))
	for i := range len(s.columns) {
		if columns[i].Type.IsArray() {
			s.values[i] = s
		} else {
			s.values[i] = scanValue{s}
		}
	}
	return s
}

func (s *scanner) Close() error {
	s.rows.Close()
	return nil
}

func (s *scanner) Err() error {
	return s.rows.Err()
}

func (s *scanner) Next() bool {
	return s.rows.Next()
}

// Scan copies the columns from the current row into dest. This differs from the
// Rows.Scan method in the sql package, which copies values into the locations
// pointed to by dest.
func (s *scanner) Scan(dest ...any) error {
	s.dest = dest
	err := s.rows.Scan(s.values...)
	s.dest = nil
	return err
}

// ScanBytes reads an array projected as bytea from the current row buffer.
func (s *scanner) ScanBytes(data []byte) error {
	var v any
	if data != nil {
		v = data
	}
	return scanValue{s}.Scan(v)
}

// normalize normalizes the value v read from PostgreSQL.
func (s *scanner) normalize(name string, typ types.Type, v any) (any, error) {
	switch typ.Kind() {
	case types.StringKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateString(name, typ, v)
		}
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.IntKind:
		if typ.IsUnsigned() {
			switch v := v.(type) {
			case int:
				if v >= 0 {
					return warehouses.ValidateUnsigned(name, typ, uint(v))
				}
			case int64:
				if v >= 0 {
					return warehouses.ValidateUnsigned(name, typ, uint(v))
				}
			case string:
				if v, err := strconv.ParseUint(v, 10, 64); err == nil {
					return warehouses.ValidateUnsigned(name, typ, uint(v))
				}
			}
		} else {
			switch v := v.(type) {
			case int:
				return warehouses.ValidateInt(name, typ, v)
			case int64:
				return warehouses.ValidateInt(name, typ, int(v))
			}
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
		switch v := v.(type) {
		case time.Time:
			return warehouses.ValidateTime(v)
		case string:
			return warehouses.ValidateTimeString(name, "15:04:05.999999", v)
		}
	case types.UUIDKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateUUID(name, v)
		}
	case types.JSONKind:
		return warehouses.ValidateJSON(name, v)
	case types.IPKind:
		if v, ok := v.(string); ok {
			// IP addresses are parsed directly here, without calling the
			// validation function inside warehouses, because the IP addresses
			// returned by PostgreSQL include the subnet mask, which must be
			// removed.
			rawIP, _, _ := strings.Cut(v, "/") // "127.0.0.1/32" -> "127.0.0.1"
			ip, ok := types.NormalizeIP(rawIP)
			if !ok {
				return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not an ip type", v, name)
			}
			return ip, nil
		}
	case types.ArrayKind:
		v, err := s.scanArray(name, typ, v)
		if err != nil {
			return nil, err
		}
		t := typ.Elem()
		for i := range v {
			v[i], err = s.normalize(name, t, v[i])
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	case types.MapKind:
		if v, ok := v.([]byte); ok {
			if v, err := types.Decode[map[string]any](bytes.NewReader(v), typ); err == nil {
				return v, nil
			}
		}
	}
	return nil, fmt.Errorf("PostgreSQL has returned an unsupported type %T for column %s", v, name)
}

// scanArray scans an array and returns the values.
func (s *scanner) scanArray(name string, typ types.Type, src any) ([]any, error) {
	data, ok := src.([]byte)
	if !ok {
		return nil, errors.New("PostgreSQL has returned an unexpected value type for an array")
	}
	if len(data) > maxArrayScanBytes {
		return nil, fmt.Errorf("binary array for column %s exceeds the %d-byte scan limit", name, maxArrayScanBytes)
	}
	p := 12
	if len(data) < p {
		return nil, errInvalidData
	}
	dimensions := int(int32(binary.BigEndian.Uint32(data[:4])))
	if dimensions < 0 {
		return nil, errInvalidData
	}
	if dimensions > 1 {
		return nil, errors.New("type is not supported")
	}
	oid := binary.BigEndian.Uint32(data[8 : 8+4])
	size := 0
	if dimensions > 0 {
		if len(data) < 12+8 {
			return nil, errInvalidData
		}
		size = int(int32(binary.BigEndian.Uint32(data[12 : 12+4])))
		p += 8
	}
	// Each element has a four-byte length prefix.
	if size < 0 || size > (len(data)-p)/4 {
		return nil, errInvalidData
	}
	if size < typ.MinElements() || size > typ.MaxElements() {
		return nil, fmt.Errorf("data warehouse returned an array with %d elements for column %s, "+
			"which is not within the expected range of [%d, %d]", size, name, typ.MinElements(), typ.MaxElements())
	}
	// Use the 64-bit interface size (16 bytes) on every architecture.
	if size > (maxArrayScanBytes-len(data))/16 {
		return nil, fmt.Errorf("array for column %s exceeds the %d-byte scan limit", name, maxArrayScanBytes)
	}
	values := make([]any, size)
	switch oid {
	case 16: // bool
		if len(data) < p+5*size {
			return nil, errInvalidData
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
			return nil, errInvalidData
		}
		var v int
		for i := range values {
			if len(data) < p+4 {
				return nil, errInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			p += 4
			if len(data) < p+l {
				return nil, errInvalidData
			}
			switch l {
			case 2:
				v = int(int16(binary.BigEndian.Uint16(data[p : p+2])))
			case 4:
				v = int(int32(binary.BigEndian.Uint32(data[p : p+4])))
			case 8:
				v = int(int64(binary.BigEndian.Uint64(data[p : p+8])))
			default:
				return nil, errInvalidData
			}
			values[i] = v
			p += l
		}
	case
		700, // float4
		701: // float8
		if len(data) < p+4*size {
			return nil, errInvalidData
		}
		var v float64
		for i := range values {
			if len(data) < p+4 {
				return nil, errInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			p += 4
			if len(data) < p+l {
				return nil, errInvalidData
			}
			switch l {
			case 4:
				v = float64(math.Float32frombits(binary.BigEndian.Uint32(data[p : p+4])))
			case 8:
				v = math.Float64frombits(binary.BigEndian.Uint64(data[p : p+8]))
			default:
				return nil, errInvalidData
			}
			values[i] = v
			p += l
		}
	case 869: // inet
		if len(data) < p+12*size {
			return nil, errInvalidData
		}
		for i := range values {
			if len(data) < p+4 {
				return nil, errInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l != 8 && l != 20 {
				return nil, errInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errInvalidData
			}
			addr, ok := netip.AddrFromSlice(data[p+4 : p+l])
			if !ok {
				return nil, errInvalidData
			}
			values[i], ok = types.NormalizeIP(addr)
			if !ok {
				return nil, errInvalidData
			}
			p += l
		}
	case
		25,   // text
		1042, // bpchar
		1043: // varchar
		for i := range values {
			if len(data) < p+4 {
				return nil, errInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l < 0 {
				return nil, errInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errInvalidData
			}
			values[i] = string(data[p : p+l])
			p += l
		}
	case 1082: // date
		if len(data) < p+8*size {
			return nil, errInvalidData
		}
		for i := range values {
			p += 4 // skip length
			days := int(int32(binary.BigEndian.Uint32(data[p : p+4])))
			values[i] = time.Date(2000, 1, 1+days, 0, 0, 0, 0, time.UTC)
			p += 4
		}
	case 1083: // time
		if len(data) < p+12*size {
			return nil, errInvalidData
		}
		for i := range values {
			p += 4 // skip length
			us := time.Duration(int64(binary.BigEndian.Uint64(data[p:p+8]))) * time.Microsecond
			values[i] = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Add(us)
			p += 8
		}
	case 1114: // timestamp
		if len(data) < p+12*size {
			return nil, errInvalidData
		}
		epoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // PostgreSQL epoch
		for i := range values {
			p += 4 // skip length
			us := time.Duration(int64(binary.BigEndian.Uint64(data[p:p+8]))) * time.Microsecond
			values[i] = epoch.Add(us)
			p += 8
		}
	case 1700: // numeric
		elem := typ.Elem()
		precision, scale := 20, 0 // uint64 is stored as numeric(20, 0)
		if elem.IsDecimal() {
			precision, scale = elem.Precision(), elem.Scale()
		} else if !elem.IsUnsigned() || !elem.IsInt(64) {
			return nil, errInvalidData
		}
		var v string
		var plan pgtype.ScanPlan
		for i := range values {
			if len(data)-p < 4 {
				return nil, errInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			p += 4
			if l < 8 || l > len(data)-p {
				return nil, errInvalidData
			}
			numeric := data[p : p+l]
			if !validBinaryNumeric(numeric, precision, scale) {
				return nil, errInvalidData
			}
			if i == 0 {
				plan = s.rows.Conn().TypeMap().PlanScan(oid, pgtype.BinaryFormatCode, &v)
			}
			err := plan.Scan(numeric, &v)
			if err != nil {
				return nil, err
			}
			values[i] = v
			p += l
		}
	case 2950: // uuid
		if len(data) < p+20*size {
			return nil, errInvalidData
		}
		for i := range values {
			p += 4 // skip length
			values[i] = uuid.UUID(data[p : p+16]).String()
			p += 16
		}
	case 114: // json
		if len(data) < p+4*size {
			return nil, errInvalidData
		}
		for i := range values {
			if len(data) < p+4 {
				return nil, errInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l <= 0 {
				return nil, errInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errInvalidData
			}
			v := make([]byte, l)
			copy(v, data[p:p+l])
			values[i] = v
			p += l
		}
	case 3802: // jsonb
		if len(data) < p+5*size {
			return nil, errInvalidData
		}
		for i := range values {
			if len(data) < p+4 {
				return nil, errInvalidData
			}
			l := int(int32(binary.BigEndian.Uint32(data[p:])))
			if l <= 0 {
				return nil, errInvalidData
			}
			p += 4
			if len(data) < p+l {
				return nil, errInvalidData
			}
			if data[p] != 1 {
				return nil, errInvalidData
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

// scanValue implements the sql.Scanner interface to read the values.
type scanValue struct {
	s *scanner
}

func (sv scanValue) Scan(v any) error {
	c := sv.s.columns[sv.s.index]
	var err error
	if v != nil {
		v, err = sv.s.normalize(c.Name, c.Type, v)
	} else if !c.Nullable {
		return fmt.Errorf("column %s is non-nullable, but PostgreSQL returned a NULL value", c.Name)
	}
	if err != nil {
		sv.s.index = 0
		if c.Type.IsArray() {
			return fmt.Errorf("column %s: %w", c.Name, err)
		}
		return err
	}
	sv.s.dest[sv.s.index] = v
	sv.s.index = (sv.s.index + 1) % len(sv.s.columns)
	return nil
}

// validBinaryNumeric reports whether a PostgreSQL binary numeric can be decoded
// within the given decimal precision and scale. data must contain its header.
func validBinaryNumeric(data []byte, precision, scale int) bool {

	count := int(binary.BigEndian.Uint16(data))
	weight := int(int16(binary.BigEndian.Uint16(data[2:])))
	sign := binary.BigEndian.Uint16(data[4:])
	dscale := int(binary.BigEndian.Uint16(data[6:]))
	if len(data) != 8+2*count || (sign != 0 && sign != 0x4000) || dscale > scale {
		return false
	}
	if count == 0 {
		return true
	}

	// Each base-10000 digit covers four decimal places. Bound both ends
	// before the codec expands the exponent or pads the fractional part.
	integerDigits := precision - scale
	lastWeight := weight - count + 1
	if weight >= (integerDigits+3)/4 || lastWeight < -(dscale+3)/4 {
		return false
	}
	for i := 8; i < len(data); i += 2 {
		if binary.BigEndian.Uint16(data[i:]) >= 10000 {
			return false
		}
	}

	// The first and last groups can straddle the allowed decimal places.
	if n := integerDigits % 4; n != 0 && weight == integerDigits/4 {
		if int(binary.BigEndian.Uint16(data[8:])) >= int(math.Pow10(n)) {
			return false
		}
	}
	if n := dscale % 4; n != 0 && lastWeight == -(dscale+3)/4 {
		if int(binary.BigEndian.Uint16(data[len(data)-2:]))%int(math.Pow10(4-n)) != 0 {
			return false
		}
	}

	return true
}
