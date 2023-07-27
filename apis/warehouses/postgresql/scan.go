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
	"math"
	"net/netip"
	"time"

	"chichi/apis/normalization"
	"chichi/connector/types"

	"github.com/google/uuid"
)

var errPostgreSQLInvalidData = errors.New("PostgreSQL has returned invalid data")

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	property    types.Property
	rows        *[][]any
	columnIndex int
	columnCount int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(properties []types.Property, rows *[][]any) []any {
	values := make([]any, len(properties))
	for i, p := range properties {
		values[i] = scanValue{
			property:    p,
			rows:        rows,
			columnIndex: i,
			columnCount: len(properties),
		}
	}
	return values
}

func (sv scanValue) Scan(src any) error {
	p := sv.property
	if src != nil && p.Type.PhysicalType() == types.PtArray {
		var err error
		src, err = sv.scanArray(src)
		if err != nil {
			return err
		}
	}
	value, err := normalization.NormalizeDatabaseFileProperty(p.Name, p.Type, src, p.Nullable)
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
	if len(data) < 12 {
		return nil, errPostgreSQLInvalidData
	}
	dimensions := int(int32(binary.BigEndian.Uint32(data[:4])))
	if dimensions != 1 {
		if dimensions == 0 {
			return nil, errPostgreSQLInvalidData
		}
		return nil, errors.New("type is not supported")
	}
	oid := binary.BigEndian.Uint32(data[8 : 8+4])
	if len(data) < 12+8 {
		return nil, errPostgreSQLInvalidData
	}
	size := int(int32(binary.BigEndian.Uint32(data[12 : 12+4])))
	p := 20
	switch oid {
	case 16: // bool
		if len(data) < p+5*size {
			return nil, errPostgreSQLInvalidData
		}
		values := make([]bool, size)
		for i := range values {
			p += 4 // skip the length
			values[i] = data[p] == 1
			p += 1
		}
		return values, nil
	case
		20, // int8
		21, // in2
		23: // int4
		var v int
		values := make([]int, size)
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
		return values, nil
	case
		700, // float4
		701: // float8
		var v float64
		values := make([]float64, size)
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
			return values, nil
		}
	case 869: // inet
		values := make([]string, size)
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
			addr, ok := netip.AddrFromSlice(data[p+4 : p+l])
			if !ok {
				return nil, errPostgreSQLInvalidData
			}
			values[i] = addr.String()
			p += l
		}
		return values, nil
	case
		25,   // text
		1042, // bpchar
		1043: // varchar
		values := make([]string, size)
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
		return values, nil
	case 1082: // date
		if len(data) < p+8*size {
			return nil, errPostgreSQLInvalidData
		}
		values := make([]time.Time, size)
		for i := range values {
			p += 4 // skip length
			days := int(int32(binary.BigEndian.Uint32(data[p : p+4])))
			values[i] = time.Date(2000, 1, 1+days, 0, 0, 0, 0, time.UTC)
			p += 4
		}
		return values, nil
	case 1083: // time
		if len(data) < p+12*size {
			return nil, errPostgreSQLInvalidData
		}
		values := make([]time.Time, size)
		for i := range values {
			p += 4 // skip length
			us := time.Duration(int64(binary.BigEndian.Uint64(data[p:p+8]))) * time.Microsecond
			values[i] = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Add(us)
			p += 8
		}
		return values, nil
	case 1114: // timestamp
		if len(data) < p+12*size {
			return nil, errPostgreSQLInvalidData
		}
		epoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // PostgreSQL epoch
		values := make([]time.Time, size)
		for i := range values {
			p += 4 // skip length
			us := time.Duration(int64(binary.BigEndian.Uint64(data[p:p+8]))) * time.Microsecond
			values[i] = epoch.Add(us)
			p += 8
		}
		return values, nil
	case 2950: // uuid
		if len(data) < p+20*size {
			return nil, errPostgreSQLInvalidData
		}
		values := make([]string, size)
		for i := range values {
			p += 4 // skip length
			values[i] = uuid.Must(uuid.FromBytes(data[p : p+16])).String()
			p += 16
		}
		return values, nil
	case 3802: // jsonb
		values := make([][]byte, size)
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
		return values, nil
	}
	return nil, errors.New("unsupported type")
}
