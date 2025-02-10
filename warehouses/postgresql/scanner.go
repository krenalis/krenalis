//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var errInvalidData = errors.New("PostgreSQL has returned invalid data")

// scanner implements the meergo.Rows interface to read and normalize the rows
// read from PostgreSQL.
type scanner struct {
	columns []meergo.Column
	rows    pgx.Rows
	values  []any
	dest    []any
	index   int
}

// newScanner returns a new scanner.
func newScanner(columns []meergo.Column, rows pgx.Rows) *scanner {
	s := &scanner{
		columns: columns,
		rows:    rows,
	}
	s.values = make([]any, len(columns))
	for i := range len(s.columns) {
		s.values[i] = scanValue{s}
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

// normalize normalizes the value v read from PostgreSQL.
func (s *scanner) normalize(name string, typ types.Type, v any) (any, error) {
	switch typ.Kind() {
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.IntKind:
		switch v := v.(type) {
		case int:
			return meergo.ValidateInt(name, typ, v)
		case int64:
			return meergo.ValidateInt(name, typ, int(v))
		}
	case types.UintKind:
		switch v := v.(type) {
		case int:
			if v >= 0 {
				return meergo.ValidateUint(name, typ, uint(v))
			}
		case int64:
			if v >= 0 {
				return meergo.ValidateUint(name, typ, uint(v))
			}
		case string:
			if v, err := strconv.ParseUint(v, 10, 64); err == nil {
				return meergo.ValidateUint(name, typ, uint(v))
			}
		}
	case types.FloatKind:
		if v, ok := v.(float64); ok {
			return meergo.ValidateFloat(name, typ, v)
		}
	case types.DecimalKind:
		if v, ok := v.(string); ok {
			return meergo.ValidateDecimalString(name, typ, v)
		}
	case types.DateTimeKind:
		if v, ok := v.(time.Time); ok {
			return meergo.ValidateDateTime(name, v)
		}
	case types.DateKind:
		if v, ok := v.(time.Time); ok {
			return meergo.ValidateDate(name, v)
		}
	case types.TimeKind:
		switch v := v.(type) {
		case time.Time:
			return meergo.ValidateTime(v)
		case string:
			return meergo.ValidateTimeString(name, "15:04:05.999999", v)
		}
	case types.YearKind:
		switch v := v.(type) {
		case int:
			return meergo.ValidateYear(name, v)
		case int64:
			return meergo.ValidateYear(name, int(v))
		}
	case types.UUIDKind:
		if v, ok := v.(string); ok {
			return meergo.ValidateUUID(name, v)
		}
	case types.JSONKind:
		return meergo.ValidateJSON(name, v)
	case types.InetKind:
		if v, ok := v.(string); ok {
			// IP addresses are parsed directly here, without calling the
			// validation function inside warehouses, because the IP addresses
			// returned by PostgreSQL include the subnet mask, which must be
			// removed.
			rawIP, _, _ := strings.Cut(v, "/") // "127.0.0.1/32" -> "127.0.0.1"
			ip, err := netip.ParseAddr(rawIP)
			if err != nil {
				return nil, fmt.Errorf("data warehouse returned a value of %q for column %s which is not an inet type", v, name)
			}
			return ip.String(), nil
		}
	case types.TextKind:
		if v, ok := v.(string); ok {
			return meergo.ValidateText(name, typ, v)
		}
	case types.ArrayKind:
		v, err := s.scanArray(v)
		if err != nil {
			return nil, err
		}
		n := len(v)
		if n < typ.MinElements() || n > typ.MaxElements() {
			return nil, fmt.Errorf("data warehouse returned an array with %d elements for column %s, which is not within the expected range of [%d, %d]",
				n, name, typ.MinElements(), typ.MaxElements())
		}
		t := typ.Elem()
		for i := 0; i < n; i++ {
			v[i], err = s.normalize(name, t, v[i])
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	case types.MapKind:
		if v, ok := v.(string); ok {
			if v, err := types.Decode[map[string]any](strings.NewReader(v), typ); err == nil {
				return v, nil
			}
		}
	}
	return nil, fmt.Errorf("PostgreSQL has returned an unsupported type %T for column %s", v, name)
}

// scanArray scans an array and returns the values.
func (s *scanner) scanArray(src any) ([]any, error) {
	data, ok := src.([]byte)
	if !ok {
		return nil, errors.New("PostgreSQL has returned an unexpected value type for an array")
	}
	p := 12
	if len(data) < p {
		return nil, errInvalidData
	}
	dimensions := int(int32(binary.BigEndian.Uint32(data[:4])))
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
			values[i] = addr.String()
			p += l
		}
	case
		25,   // text
		1042, // bpchar
		1043: // varchar
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
		var v string
		tm := s.rows.Conn().TypeMap()
		ps := tm.PlanScan(oid, pgtype.BinaryFormatCode, &v)
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
			err := ps.Scan(data[p:p+l], &v)
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
			values[i] = uuid.Must(uuid.FromBytes(data[p : p+16])).String()
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
		return err
	}
	sv.s.dest[sv.s.index] = v
	sv.s.index = (sv.s.index + 1) % len(sv.s.columns)
	return nil
}
