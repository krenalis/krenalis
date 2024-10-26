//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var errPostgreSQLInvalidData = errors.New("PostgreSQL has returned invalid data")

// Normalize normalizes a value v returned by the Query method.
func (warehouse *PostgreSQL) Normalize(name string, typ types.Type, v any, nullable bool) (any, error) {
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
		var data []byte
		switch v := v.(type) {
		case []byte:
			data = v
		case string:
			data = []byte(v)
		}
		if data != nil {
			// PostgreSQL returns JSON with insignificant whitespace characters.
			data, err := json.Compact(data)
			if err != nil {
				return nil, fmt.Errorf("data warehouse returned an invalid JSON value for column %s", name)
			}
			return json.Value(data), nil
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
			return meergo.ValidateText(name, typ, v)
		}
	case types.ArrayKind:
		pool, err := warehouse.connectionPool(context.Background())
		if err != nil {
			return nil, err
		}
		conn, err := pool.Acquire(context.Background())
		if err != nil {
			return nil, err
		}
		defer conn.Release()
		tm := conn.Conn().TypeMap()
		v, err := scanArray(tm, v)
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
			v[i], err = warehouse.Normalize(name, t, v[i], false)
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	case types.MapKind:
		if v, err := json.DecodeBySchema(strings.NewReader(v.(string)), typ); err == nil {
			return v, nil
		}
	}
	return nil, fmt.Errorf("PostgreSQL has returned an unsupported type %T for column %s", v, name)
}

// scanArray scans an array and returns the values.
func scanArray(tm *pgtype.Map, src any) ([]any, error) {
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
	case 1700: // numeric
		var s string
		ps := tm.PlanScan(oid, pgtype.BinaryFormatCode, &s)
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
			err := ps.Scan(data[p:p+l], &s)
			if err != nil {
				return nil, err
			}
			values[i] = s
			p += l
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
